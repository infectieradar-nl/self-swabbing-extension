package handlers

import (
	"fmt"
	"net/http"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/utils"
	"github.com/influenzanet/study-service/pkg/studyengine"
)

func (h *HttpEndpoints) AddSamplerAPI(rg *gin.RouterGroup) {
	samplerGroup := rg.Group("/sampler/:instanceID")
	samplerGroup.Use(mw.HasValidInstanceID())
	samplerGroup.Use(mw.HasValidAPIKey(h.apiKeys))
	{
		samplerGroup.POST("/is-selected", mw.RequirePayload(), h.samplerIsSelected)
		samplerGroup.POST("/invite-response", mw.RequirePayload(), h.samplerInviteResponse)
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

	// clean up unconfirmed reserved slots
	err := h.dbService.CleanUpExpiredSlotReservations(h.instanceID)
	if err != nil {
		logger.Error.Println(err)
	}

	if h.sampler.NeedsRefresh() {
		logger.Debug.Println("creating new slot curve from sample")
		h.sampler.InitFromSampleCSV(h.samplerConfig.SampleFilePath, h.samplerConfig.TargetSamples, h.samplerConfig.OpenSlotsAtStart)
		h.sampler.SaveSlotCurveToDB()
	}

	if !h.sampler.HasAvailableFreeSlots() {
		logger.Debug.Println("no free slots available")
		c.JSON(http.StatusOK, gin.H{"value": false})
		return
	}

	// reserve slot:
	err = h.dbService.ReserveSlot(instanceID, req.ParticipantState.ParticipantID)
	if err != nil {
		logger.Error.Println(err)
		c.JSON(http.StatusOK, gin.H{"value": false})
		return
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

	confirmSurveyItem, err := utils.FindSurveyItemResponse(req.Response.Responses, "SwabSample.Confirm")
	if err != nil {
		logger.Debug.Printf("%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	confirmedResponse, err := utils.FindResponseSlot(confirmSurveyItem.Response, "rg.scg")
	if err != nil {
		logger.Debug.Printf("%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Debug.Println(confirmedResponse)

	if len(confirmedResponse.Items) != 1 {
		msg := fmt.Sprintf("unexpected rsponse slot info: %v", confirmedResponse)
		logger.Error.Printf(msg)
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	if confirmedResponse.Items[0].Key == "1" {
		// Confirmed participation:
		err := h.dbService.ConfirmSlot(instanceID, req.ParticipantState.ParticipantID)
		if err != nil {
			logger.Error.Printf("%v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		// rejected participation:
		err := h.dbService.CancelSlotReservation(instanceID, req.ParticipantState.ParticipantID)
		if err != nil {
			logger.Error.Printf("%v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
}
