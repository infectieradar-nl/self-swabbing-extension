package handlers

import (
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

	return &HttpEndpoints{
		instanceID:           instanceID,
		dbService:            dbService,
		apiKeys:              apiKeys,
		allowEntryCodeUpload: allowEntryCodeUpload,
		samplerConfig:        samplerConfig,
		sampler:              s,
	}
}
