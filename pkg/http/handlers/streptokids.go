package handlers

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
)

const (
	ENV_ALLOWED_REFERER_FOR_STREPTOKIDS_REG = "ALLOWED_REFERER_FOR_STREPTOKIDS_REG"
	ENV_STREPTOKIDS_USERMANAGEMENT_URL      = "STREPTOKIDS_USERMANAGEMENT_URL"
	ENV_STREPTOKIDS_MESSAGING_SERVICE_URL   = "STREPTOKIDS_MESSAGING_SERVICE_URL"
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
	// authGroup.Use(mw.HasValidAPIKey(h.apiKeys))
	// authGroup.Use(mw.ExtractToken())
	//authGroup.Use(mw.ValidateToken(clientUserManagement))
	//authGroup.Use(mw.HasRole([]string{"RESEARCHER", "ADMIN"}))
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

func (h *HttpEndpoints) streptokidsSendControlInvitations(c *gin.Context) {

}

func (h *HttpEndpoints) streptokidsRemoveExpiredRegistrations(c *gin.Context) {

	referenceTime := time.Now().AddDate(-1, 0, 0)

	// TODO: call DB method
	logger.Info.Printf("Successfully removed registrtions before %s", referenceTime)
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
