package handlers

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
	"github.com/influenzanet/go-utils/pkg/api_types"
	umAPI "github.com/influenzanet/user-management-service/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	ENV_ALLOWED_REFERER_FOR_STREPTOKIDS_REG = "ALLOWED_REFERER_FOR_STREPTOKIDS_REG"
	ENV_STREPTOKIDS_USERMANAGEMENT_URL      = "STREPTOKIDS_USERMANAGEMENT_URL"
	ENV_STREPTOKIDS_MESSAGING_SERVICE_URL   = "STREPTOKIDS_MESSAGING_SERVICE_URL"

	DefaultGRPCMaxMsgSize = 4194304
)

func (h *HttpEndpoints) AddStreptokidsAPI(rg *gin.RouterGroup,

//clientUserManagement todo

) {
	g := rg.Group("/streptokids")
	g.Use(mw.HasValidAPIKey(h.apiKeys))
	{
		g.POST("/registration", mw.RequirePayload(), h.streptokidsRegisterNewParticipant)
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
	logger.Debug.Println(token)
	// TODO connect to messaging

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

		// TODO
		// TODO
		// TODO send email
		// TODO
		// TODO
		count += 1
		err = h.dbService.StreptokidsMarkControlContactInvited(id)
		if err != nil {
			logger.Error.Printf("streptokids contact could not be marked as invited: %s", id)
		}
	}

	// TODO close connection to messaging

	c.JSON(http.StatusOK, gin.H{"msg": "message sending finished",
		"count": count,
	})
}

func (h *HttpEndpoints) streptokidsRemoveExpiredRegistrations(c *gin.Context) {

	referenceTime := time.Now().AddDate(-1, 0, 0)

	// TODO: call DB method
	logger.Info.Printf("Successfully removed registrations before %s", referenceTime)
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
