package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
	"github.com/influenzanet/go-utils/pkg/api_types"
	"github.com/influenzanet/messaging-service/pkg/api/email_client_service"
	umAPI "github.com/influenzanet/user-management-service/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	ENV_ALLOWED_REFERER_FOR_STREPTOKIDS_REG  = "ALLOWED_REFERER_FOR_STREPTOKIDS_REG"
	ENV_STREPTOKIDS_USERMANAGEMENT_URL       = "STREPTOKIDS_USERMANAGEMENT_URL"
	ENV_STREPTOKIDS_EMAIL_CLIENT_SERVICE_URL = "STREPTOKIDS_EMAIL_CLIENT_SERVICE_URL"
	ENV_STREPTOKIDS_EMAIL_INVITE_SUBJECT     = "STREPTOKIDS_EMAIL_INVITE_SUBJECT"
	ENV_STREPTOKIDS_PATH_TO_TEMPLATE_FILE    = "STREPTOKIDS_PATH_TO_TEMPLATE_FILE"

	DefaultGRPCMaxMsgSize = 4194304
	maxCodeAge            = -14 * 24 * time.Hour
)

func (h *HttpEndpoints) AddStreptokidsAPI(rg *gin.RouterGroup) {
	rand.Seed(time.Now().UnixNano())

	g := rg.Group("/streptokids")
	g.Use(mw.HasValidAPIKey(h.apiKeys))
	{
		g.POST("/registration", mw.RequirePayload(), h.streptokidsRegisterNewParticipant)
		g.GET("/check-code", h.streptokidsCheckControlAccessCode) // ?code=123123123132
		g.GET("/code-used", h.streptokidsControlAccessCodeUsed)   // ?code=123123123132
	}

	authGroup := rg.Group("/streptokids")
	authGroup.Use(ExtractToken())
	authGroup.Use(ValidateToken())
	authGroup.Use(HasRole([]string{"RESEARCHER", "ADMIN"}))
	{
		authGroup.GET("/registration", h.streptokidsFetchRegistrations) // ?since=1545345341&includeInvited=false
		authGroup.POST("/invite", mw.RequirePayload(), h.streptokidsSendControlInvitations)
		authGroup.DELETE("/expired-registrations", h.streptokidsRemoveExpiredRegistrations)
	}
}

func (h *HttpEndpoints) streptokidsRegisterNewParticipant(c *gin.Context) {

	currentRef := c.Request.Referer()
	if !hasAllowedReferer(currentRef) {
		logger.Error.Printf("unexpected referer in the request: %s", currentRef)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req types.StreptokidsControlRegistration
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.SubmittedAt = time.Now().Unix()

	_, err := h.dbService.StreptokidsAddControlContact(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "contact saved"})
}

func (h *HttpEndpoints) streptokidsFetchRegistrations(c *gin.Context) {
	since, err := strconv.ParseInt(c.DefaultQuery("since", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	includeInvited := c.DefaultQuery("includeInvited", "false") == "true"

	contacts, err := h.dbService.StreptokidsFetchControlContacts(since, includeInvited)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"contact": contacts})
}

func (h *HttpEndpoints) streptokidsCheckControlAccessCode(c *gin.Context) {
	code := c.DefaultQuery("code", "")
	if code == "" {
		time.Sleep(6 * time.Second)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
		return
	}
	code = SanitizeCode(code)

	// clean old codes:
	before := time.Now().Add(maxCodeAge).Unix()
	_, err := h.dbService.StreptokidsDeleteControlCodesBefore(before)
	if err != nil {
		logger.Error.Println(err)
	}

	_, err = h.dbService.StreptokidsFindControlCode(code)
	if err != nil {
		time.Sleep(6 * time.Second)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "code accepted"})
}

func (h *HttpEndpoints) streptokidsControlAccessCodeUsed(c *gin.Context) {
	code := c.DefaultQuery("code", "")
	if code == "" {
		time.Sleep(6 * time.Second)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
		return
	}
	code = SanitizeCode(code)

	_, err := h.dbService.StreptokidsDeleteControlCode(code)
	if err != nil {
		time.Sleep(6 * time.Second)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "code successfully deleted"})
}

type SendInviteToIDsReq struct {
	IDs []string `json:"ids"`
}

func (h *HttpEndpoints) streptokidsSendControlInvitations(c *gin.Context) {
	var req SendInviteToIDsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token := c.MustGet("validatedToken").(*api_types.TokenInfos)
	logger.Info.Printf("user %s initiated streptokids invites", token.Id)

	emailClient, emailServiceClose := ConnectToEmailService(os.Getenv(ENV_STREPTOKIDS_EMAIL_CLIENT_SERVICE_URL), DefaultGRPCMaxMsgSize)
	defer emailServiceClose()

	// read template file:
	emailTemplate, err := os.ReadFile(os.Getenv(ENV_STREPTOKIDS_PATH_TO_TEMPLATE_FILE))
	if err != nil {
		logger.Error.Printf("unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	count := 0
	for _, id := range req.IDs {
		contact, err := h.dbService.StreptokidsFindOneControlContact(id)
		if err != nil {
			logger.Error.Printf("streptokids contact cannot be found: %s", id)
			continue
		}

		if contact.InvitedAt > 0 {
			logger.Error.Printf("streptokids contact already invited: %v", contact)
			continue
		}

		// generate 12 digit code and save to DB (with retries)
		var code, codePretty string
		for i := 0; i < 10; i++ {
			codeVal := generateRandomCode()
			code = randomCodeToString(codeVal, false)
			codePretty = randomCodeToString(codeVal, true)

			_, err = h.dbService.StreptokidsAddControlCode(code)
			if err != nil {
				// if code already exists, try again
				logger.Error.Println(err)
				continue
			}
			break
		}

		content, err := ResolveTemplate(
			"streptokidsInvite",
			string(emailTemplate),
			map[string]string{
				"code":       code,
				"codePretty": codePretty,
				"age":        fmt.Sprintf("%d", contact.Age),
			},
		)
		if err != nil {
			logger.Error.Printf("streptokids contact message could not be generated: %v", err)
			continue
		}

		// send email
		_, err = emailClient.SendEmail(context.TODO(), &email_client_service.SendEmailReq{
			To:      []string{contact.Email},
			Subject: os.Getenv(ENV_STREPTOKIDS_EMAIL_INVITE_SUBJECT),
			Content: content,
		})
		if err != nil {
			logger.Error.Printf("streptokids contact message could not be sent for id %s: %v", id, err)
			continue
		}

		count += 1
		err = h.dbService.StreptokidsMarkControlContactInvited(id)
		if err != nil {
			logger.Error.Printf("streptokids contact could not be marked as invited: %s", id)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"msg":   "message sending finished",
		"count": count,
	})
}

func (h *HttpEndpoints) streptokidsRemoveExpiredRegistrations(c *gin.Context) {
	referenceTime := time.Now().AddDate(-1, 0, 0)
	count, err := h.dbService.StreptokidsDeleteContactsBefore(referenceTime.Unix())
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info.Printf("Successfully removed %d registrations before %s", count, referenceTime)
	c.JSON(http.StatusOK, gin.H{"msg": "remove finished",
		"count": count,
	})
}

func hasAllowedReferer(currentReferer string) bool {
	allowedRefs := strings.Split(os.Getenv(ENV_ALLOWED_REFERER_FOR_STREPTOKIDS_REG), ",")

	for _, ref := range allowedRefs {
		if strings.HasPrefix(currentReferer, ref) {
			return true
		}
	}
	return false
}

func ExtractToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request

		var token string
		tokens, ok := req.Header["Authorization"]
		if ok && len(tokens) > 0 {
			token = tokens[0]
			token = strings.TrimPrefix(token, "Bearer ")
			if len(token) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "no Authorization token found"})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no Authorization token found"})
			c.Abort()
			return
		}

		c.Set("encodedToken", token)
		c.Next()
	}
}

func ValidateToken() gin.HandlerFunc {
	return func(c *gin.Context) {

		authClient, userManagementClose := ConnectToUserManagement(os.Getenv(ENV_STREPTOKIDS_USERMANAGEMENT_URL), DefaultGRPCMaxMsgSize)
		defer userManagementClose()

		token := c.MustGet("encodedToken").(string)
		parsedToken, err := authClient.ValidateJWT(context.Background(), &umAPI.JWTRequest{
			Token: token,
		})
		if err != nil {
			st := status.Convert(err)
			logger.Error.Println(st.Message())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "error during token validation"})
			c.Abort()
			return
		}
		c.Set("validatedToken", parsedToken)
		c.Next()
	}
}

func HasRole(targetRoles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.MustGet("validatedToken").(*api_types.TokenInfos)

		if val, ok := token.Payload["roles"]; ok {
			roles := strings.Split(val, ",")
			for _, r := range roles {
				for _, tRole := range targetRoles {
					if r == tRole {
						c.Next()
						return
					}
				}

			}
		}
		logger.Warning.Printf("user attempted to access resources with inproper roles: %v", token)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect role"})
		c.Abort()
	}
}

func ConnectToUserManagement(addr string, maxMsgSize int) (client umAPI.UserManagementApiClient, close func() error) {
	serverConn := connectToGRPCServer(addr, maxMsgSize)
	return umAPI.NewUserManagementApiClient(serverConn), serverConn.Close
}

func ConnectToEmailService(addr string, maxMsgSize int) (client email_client_service.EmailClientServiceApiClient, close func() error) {
	serverConn := connectToGRPCServer(addr, maxMsgSize)
	return email_client_service.NewEmailClientServiceApiClient(serverConn), serverConn.Close
}

func connectToGRPCServer(addr string, maxMsgSize int) *grpc.ClientConn {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(maxMsgSize),
		grpc.MaxCallSendMsgSize(maxMsgSize),
	))
	if err != nil {
		logger.Error.Fatalf("failed to connect to %s: %v", addr, err)
	}
	return conn
}

func generateRandomCode() int {
	v := rand.Intn(999999999999 - 100000000000)
	return v
}

func randomCodeToString(code int, pretty bool) string {
	codeStr := fmt.Sprintf("%d", code)
	if !pretty || len(codeStr) < 12 {
		return codeStr
	}

	return fmt.Sprintf("%s-%s-%s", codeStr[0:4], codeStr[4:8], codeStr[8:])
}

func ResolveTemplate(tempName string, templateDef string, contentInfos map[string]string) (content string, err error) {
	if strings.TrimSpace(templateDef) == "" {
		logger.Error.Printf("error: empty template %s", tempName)
		return "", errors.New("empty template `" + tempName)
	}
	tmpl, err := template.New(tempName).Parse(templateDef)
	if err != nil {
		logger.Error.Printf("error when parsing template %s: %v", tempName, err)
		return "", err
	}
	var tpl bytes.Buffer

	err = tmpl.Execute(&tpl, contentInfos)
	if err != nil {
		logger.Error.Printf("error when executing template %s: %v", tempName, err)
		return "", err
	}
	return tpl.String(), nil
}
