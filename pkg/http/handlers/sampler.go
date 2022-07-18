package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
	"github.com/influenzanet/study-service/pkg/studyengine"
)

func (h *HttpEndpoints) AddSamplerAPI(rg *gin.RouterGroup) {
	samplerGroup := rg.Group("/sampler/:instanceID")
	samplerGroup.Use(mw.HasValidInstanceID())
	samplerGroup.Use(mw.HasValidAPIKey(h.apiKeys))
	{

		samplerGroup.POST("/is-selected", h.samplerIsSelected)
		samplerGroup.POST("/invite-response", mw.RequirePayload(), h.recordBodyHandl) // h.samplerInviteResponse)
	}

}

func (h *HttpEndpoints) samplerIsSelected(c *gin.Context) {
	var req studyengine.ExternalEventPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error.Printf("error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instanceID := req.InstanceID
	if instanceID != h.instanceID {
		msg := fmt.Sprintf("unexpected instanceID: %s", req.InstanceID)
		logger.Error.Printf(msg)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// in every request:
	// TODO: clean up unconfirmed reserved slots
	h.dbService.CleanUpExpiredSlotReservations(h.instanceID)

	if h.sampler.NeedsRefresh() {
		logger.Debug.Println("creating new slot curve from sample")
		h.sampler.InitFromSampleCSV(h.samplerConfig.SampleFilePath, h.samplerConfig.TargetSamples, h.samplerConfig.OpenSlotsAtStart)
		h.sampler.SaveSlotCurveToDB()
	}
	if h.sampler.HasAvailableFreeSlots() {
		// reserve slot
	}
	c.JSON(http.StatusOK, gin.H{"value": true})
}

func (h *HttpEndpoints) samplerInviteResponse(c *gin.Context) {
	var req studyengine.ExternalEventPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error.Printf("error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instanceID := req.InstanceID
	if instanceID != h.instanceID {
		msg := fmt.Sprintf("unexpected instanceID: %s", req.InstanceID)
		logger.Error.Printf(msg)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	// if confirmed -> confirm slot
	// if rejected -> remove reservation
}

func (h *HttpEndpoints) recordBodyHandl(c *gin.Context) {
	req, err := ioutil.ReadAll(c.Request.Body)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "Unable to read request body",
		})
		return
	}

	filename := fmt.Sprintf("%s.json", time.Now().Format("2006-01-02-15-04-05"))
	err = ioutil.WriteFile(filename, req, 0644)
	if err != nil {
		logger.Error.Println(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "Unable to save the file",
		})
		return
	}

	// File saved successfully. Return proper result
	c.JSON(http.StatusOK, gin.H{"message": "Your file has been successfully saved."})
}
