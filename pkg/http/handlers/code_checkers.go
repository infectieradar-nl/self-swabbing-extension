package handlers

import (
	"fmt"
	"net/http"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
)

func (h *HttpEndpoints) AddCodeCheckerAPI(rg *gin.RouterGroup) {
	codeCheckGroup := rg.Group("/:instanceID/entry-codes")
	codeCheckGroup.Use(mw.HasValidInstanceID())
	codeCheckGroup.Use(mw.HasValidAPIKey(h.apiKeys))
	{
		codeCheckGroup.POST("", mw.RequirePayload(), h.addNewEntryCodesHandl)
		codeCheckGroup.GET("/is-valid", h.validateEntryCodeHandl)
		codeCheckGroup.POST("/submit", mw.RequirePayload(), h.studyEventWithEntryCodeHandl)
	}

}

func (h *HttpEndpoints) addNewEntryCodesHandl(c *gin.Context) {
	instanceID := c.Param("instanceID")

	var req types.NewCodeList
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.dbService.CreateIndexForEntryCodes(instanceID)
	if err != nil {
		logger.Error.Printf("unexpected error when creating index: %v", err)
	}

	counter := 0
	for _, c := range req.Codes {
		_, err := h.dbService.AddEntryCode(instanceID, c)
		if err != nil {
			logger.Error.Printf("unexpected error when saving entry code '%s': %v", c, err)
		} else {
			counter += 1
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d / %d codes saved", counter, len(req.Codes))})
}

func (h *HttpEndpoints) validateEntryCodeHandl(c *gin.Context) {
	instanceID := c.Param("instanceID")

	code := c.DefaultQuery("code", "")
	if code == "" {
		logger.Warning.Println("empty entry code attempt")
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty entry code attempt"})
		return
	}

	codeInfos, err := h.dbService.FindEntryCodeInfo(instanceID, code)
	if err != nil {
		logger.Error.Printf("error when looking up code infos for '%s': %v", code, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong entry code"})
		return
	}

	if codeInfos.UsedAt > 0 {
		logger.Error.Printf("attempt to use expired code '%s': %v", code, codeInfos)
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong entry code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isValid": true})
}

func (h *HttpEndpoints) studyEventWithEntryCodeHandl(c *gin.Context) {
	// TODO: receive and parse study event

	// TODO: find survey item and response item with code

	// TODO: update code in DB that is was used by participant
}
