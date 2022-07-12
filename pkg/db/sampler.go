package db

import (
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

func (dbService *SelfSwabbingExtDBService) GetUsedSlotsCountSince(instanceID string, ref int64) (count int64, err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{
		"time": bson.M{"$gt": ref},
	}
	count, err = dbService.collectionRefUsedSlots(instanceID).CountDocuments(ctx, filter)
	return
}

type UsedSlot struct {
	Time          int64  `bson:"time" json:"time"`
	ParticipantID string `bson:"participantID" json:"participantID"`
	Status        string `bson:"status" json:"status"`
}

const (
	USED_SLOT_STATUS_RESERVED  = "reserved"
	USED_SLOT_STATUS_CONFIRMED = "confirmed"
)

func (dbService *SelfSwabbingExtDBService) ReserveSlot(instanceID string, participantID string) error {
	ctx, cancel := dbService.getContext()
	defer cancel()

	newUsedSlot := UsedSlot{
		Time:          time.Now().Unix(),
		ParticipantID: participantID,
		Status:        USED_SLOT_STATUS_RESERVED,
	}

	_, err := dbService.collectionRefUsedSlots(instanceID).InsertOne(ctx, newUsedSlot)
	return err
}

func (dbService *SelfSwabbingExtDBService) CancelSlotReservation(instanceID string, participantID string) error {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{
		"participantID": participantID,
		"status":        USED_SLOT_STATUS_RESERVED,
	}

	var res UsedSlot
	opts := options.FindOneAndDelete()
	opts.SetSort(bson.D{{Key: "time", Value: -1}})
	err := dbService.collectionRefUsedSlots(instanceID).FindOneAndDelete(ctx, filter, opts).Decode(&res)

	return err
}

func (dbService *SelfSwabbingExtDBService) ConfirmSlot(instanceID string, participantID string) error {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{
		"participantID": participantID,
		"status":        USED_SLOT_STATUS_RESERVED,
	}

	update := bson.M{"$set": bson.M{"status": USED_SLOT_STATUS_CONFIRMED}}

	var res UsedSlot
	opts := options.FindOneAndUpdate()
	opts.SetSort(bson.D{{Key: "time", Value: -1}})
	err := dbService.collectionRefUsedSlots(instanceID).FindOneAndUpdate(ctx, filter, update, opts).Decode(&res)

	return err
}

func (dbService *SelfSwabbingExtDBService) CleanUpExpiredSlotReservations(instanceID string) error {
	ctx, cancel := dbService.getContext()
	defer cancel()

	ref := time.Now().AddDate(0, 0, -7).Unix()
	filter := bson.M{
		"$and": bson.A{
			bson.M{"time": bson.M{"$lt": ref}},
			bson.M{"status": USED_SLOT_STATUS_RESERVED},
		},
	}
	_, err := dbService.collectionRefUsedSlots(instanceID).DeleteMany(ctx, filter)
	return err
}
