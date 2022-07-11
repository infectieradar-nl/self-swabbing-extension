package db

import (
	"errors"
	"time"

	"github.com/coneno/logger"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/sampler"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (dbService *SelfSwabbingExtDBService) CreateIndexesForSampler(instanceID string) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	_, err := dbService.collectionRefSlotCurves(instanceID).Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{
				"intervalStart": -1,
			},
			Options: options.Index().SetUnique(true),
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}

	_, err = dbService.collectionRefUsedSlots(instanceID).Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{
				"time": -1,
			},
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}

	_, err = dbService.collectionRefUsedSlots(instanceID).Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "time", Value: -1},
				{Key: "participantID", Value: 1},
			},
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}
}

func (dbService *SelfSwabbingExtDBService) LoadLatestSlotCurve(instanceID string) (res sampler.SlotCurve, err error) {
	return res, errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) SaveNewSlotCurve(instanceID string, res sampler.SlotCurve) (err error) {
	return errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) GetUsedSlotsSince(instanceID string, ref time.Time) (count int, err error) {
	return 0, errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) ReserveSlot(instanceID string, participantID string) error {
	return errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) CancelSlotReservation(instanceID string, participantID string) error {
	return errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) ConfirmSlot(instanceID string, participantID string) error {
	return errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) CleanUpExpiredSlotReservations(instanceID string) error {
	return errors.New("unimplemented")
}
