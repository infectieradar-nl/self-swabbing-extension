package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
)

const (
	ENV_ALLOWED_REFERER_FOR_STREPTOKIDS_REG = "ALLOWED_REFERER_FOR_STREPTOKIDS_REG"
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
	authGroup.Use(mw.HasValidAPIKey(h.apiKeys))
	// authGroup.Use(mw.ExtractToken())
	//authGroup.Use(mw.ValidateToken(clientUserManagement))
	//authGroup.Use(mw.HasRole([]string{"RESEARCHER", "ADMIN"}))
	{
		authGroup.GET("/registration", h.streptokidsFetchRegistrations)
		authGroup.POST("/invite", mw.RequirePayload(), h.streptokidsSendControlInvitations)
	}
}

func (h *HttpEndpoints) streptokidsRegisterNewParticipant(c *gin.Context) {

	currentRef := c.Request.Referer()
	if !hasAllowedReferer(currentRef) {
		logger.Error.Printf("unexpected referer in the request: %s", currentRef)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	logger.Debug.Println()
	// TODO: save new registration entry into db
	// TODO: response with success

}
func (h *HttpEndpoints) streptokidsFetchRegistrations(c *gin.Context) {
	// TODO:
}
func (h *HttpEndpoints) streptokidsSendControlInvitations(c *gin.Context) {

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
