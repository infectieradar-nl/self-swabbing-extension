package handlers

import (
	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	mw "github.com/infectieradar-nl/self-swabbing-extension/pkg/http/middlewares"
)

func (h *HttpEndpoints) AddSamplerAPI(rg *gin.RouterGroup) {
	samplerGroup := rg.Group("/sampler/:instanceID")
	samplerGroup.Use(mw.HasValidInstanceID())
	samplerGroup.Use(mw.HasValidAPIKey(h.apiKeys))
	{

		samplerGroup.GET("/is-selected", h.samplerIsSelected)
		samplerGroup.POST("/invite-response", mw.RequirePayload(), h.samplerInviteResponse)
	}

}

func (h *HttpEndpoints) samplerIsSelected(c *gin.Context) {
	// in every request:
	// TODO: clean up unconfirmed reserved slots
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
