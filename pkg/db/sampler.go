package db

import (
	"errors"

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
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{}
	opts := options.FindOne()
	opts.SetSort(bson.D{{Key: "intervalStart", Value: -1}})

	if err = dbService.collectionRefSlotCurves(instanceID).FindOne(
		ctx,
		filter,
		opts,
	).Decode(&res); err != nil {
		return res, err
	}
	return res, nil
}

func (dbService *SelfSwabbingExtDBService) SaveNewSlotCurve(instanceID string, obj sampler.SlotCurve) (err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	_, err = dbService.collectionRefSlotCurves(instanceID).InsertOne(ctx, obj)
	return err
}

func (dbService *SelfSwabbingExtDBService) GetUsedSlotsSince(instanceID string, ref int64) (count int, err error) {
	return 0, errors.New("TODO: unimplemented")
}

func (dbService *SelfSwabbingExtDBService) ReserveSlot(instanceID string, participantID string) error {
	return errors.New("TODO: unimplemented")
}

func (dbService *SelfSwabbingExtDBService) CancelSlotReservation(instanceID string, participantID string) error {
	return errors.New("TODO: unimplemented")
}

func (dbService *SelfSwabbingExtDBService) ConfirmSlot(instanceID string, participantID string) error {
	return errors.New("TODO: unimplemented")
}

func (dbService *SelfSwabbingExtDBService) CleanUpExpiredSlotReservations(instanceID string) error {
	return errors.New("TODO: unimplemented")
}
