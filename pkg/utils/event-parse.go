package utils

import (
	"errors"
	"strings"

	"github.com/coneno/logger"
	"github.com/influenzanet/study-service/pkg/types"
)

func FindSurveyItemResponse(response []types.SurveyItemResponse, itemKey string) (types.SurveyItemResponse, error) {
	for _, resp := range response {
		if strings.Contains(resp.Key, itemKey) {
			return resp, nil
		}
	}
	return types.SurveyItemResponse{}, errors.New("Could not find response item")
}

func FindResponseSlot(rootItem *types.ResponseItem, slotKey string) (*types.ResponseItem, error) {
	keyParts := strings.Split(slotKey, ".")
	if len(keyParts) > 1 {
		for _, item := range rootItem.Items {
			res, err := FindResponseSlot(&item, strings.Join(keyParts[1:], "."))
			if err == nil {
				return res, nil
			}
		}
	} else {
		if rootItem.Key == keyParts[0] {
			return rootItem, nil
		}
	}
	logger.Debug.Println(keyParts)

	return nil, errors.New("could not find response slot")
}
