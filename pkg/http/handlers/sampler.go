package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
)

func (h *HttpEndpoints) AddSamplerAPI(rg *gin.RouterGroup) {
	samplerGroup := rg.Group("/sampler/:instanceID")
	samplerGroup.Use(mw.HasValidInstanceID())
	samplerGroup.Use(mw.HasValidAPIKey(h.apiKeys))
	{

		samplerGroup.GET("/is-selected", h.recordBodyHandl)                           // h.samplerIsSelected)
		samplerGroup.POST("/invite-response", mw.RequirePayload(), h.recordBodyHandl) // h.samplerInviteResponse)
	}

}

func (h *HttpEndpoints) samplerIsSelected(c *gin.Context) {
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
}

func (h *HttpEndpoints) samplerInviteResponse(c *gin.Context) {
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
