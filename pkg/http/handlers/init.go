package handlers

import "github.com/infectieradar-nl/self-swabbing-extension/pkg/db"

type HttpEndpoints struct {
	dbService            *db.SelfSwabbingExtDBService
	apiKeys              []string
	allowEntryCodeUpload bool
}

func NewHTTPHandler(
	dbService *db.SelfSwabbingExtDBService,
	apiKeys []string,
	allowEntryCodeUpload bool,
) *HttpEndpoints {
	return &HttpEndpoints{
		dbService:            dbService,
		apiKeys:              apiKeys,
		allowEntryCodeUpload: allowEntryCodeUpload,
	}
}
