package handlers

import (
	"github.com/coneno/logger"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/db"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/sampler"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
)

type HttpEndpoints struct {
	instanceID           string
	dbService            *db.SelfSwabbingExtDBService
	apiKeys              []string
	allowEntryCodeUpload bool
	samplerConfig        types.SamplerConfig
	sampler              *sampler.Sampler
}

func NewHTTPHandler(
	instanceID string,
	dbService *db.SelfSwabbingExtDBService,
	apiKeys []string,
	allowEntryCodeUpload bool,
	samplerConfig types.SamplerConfig,
) *HttpEndpoints {

	// in init:
	s := sampler.NewSampler(instanceID, dbService)
	s.LoadSlotCurveFromDB()
	if s.NeedsRefresh() {
		logger.Debug.Println("creating new slot curve from sample")
		s.InitFromSampleCSV(samplerConfig.SampleFilePath, samplerConfig.TargetSamples, samplerConfig.OpenSlotsAtStart)
		s.SaveSlotCurveToDB()
	}

	return &HttpEndpoints{
		instanceID:           instanceID,
		dbService:            dbService,
		apiKeys:              apiKeys,
		allowEntryCodeUpload: allowEntryCodeUpload,
		samplerConfig:        samplerConfig,
		sampler:              s,
	}
}
