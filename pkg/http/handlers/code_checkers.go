package handlers

import (
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
)

func (h *HttpEndpoints) AddCodeCheckerAPI(rg *gin.RouterGroup) {
	codeCheckGroup := rg.Group("/code-checker")
	codeCheckGroup.Use(mw.HasValidAPIKey(h.apiKeys))

}
