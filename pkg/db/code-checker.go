package db

import (
	"errors"
	"time"

	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (dbService *SelfSwabbingExtDBService) CreateIndexForEntryCodes(instanceID string) error {
	ctx, cancel := dbService.getContext()
	defer cancel()

	_, err := dbService.collectionRefEntryCodes(instanceID).Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{
				"code": 1,
			},
			Options: options.Index().SetUnique(true),
		},
	)
	return err
}

func (dbService *SelfSwabbingExtDBService) AddEntryCode(instanceID string, entryCode string) (string, error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	newEntryCode := types.ValidationCode{
		Code:       entryCode,
		UploadedAt: time.Now().Unix(),
	}

	res, err := dbService.collectionRefEntryCodes(instanceID).InsertOne(ctx, newEntryCode)
	if err != nil {
		return "", err
	}
	id := res.InsertedID.(primitive.ObjectID)
	return id.Hex(), err
}

func (dbService *SelfSwabbingExtDBService) FindEntryCodeInfo(instanceID string, code string) (entryCode types.ValidationCode, err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{
		"code": code,
	}

	if err = dbService.collectionRefEntryCodes(instanceID).FindOne(
		ctx,
		filter,
		options.FindOne(),
	).Decode(&entryCode); err != nil {
		return entryCode, err
	}

	return entryCode, err
}

func (dbService *SelfSwabbingExtDBService) MarkEntryCodeAsUsed(instanceID string, code string, usedBy string) (err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{
		"$and": bson.A{
			bson.M{"code": code},
			bson.M{"usedAt": bson.M{"$lt": 1}},
		},
	}
	update := bson.M{"$set": bson.M{
		"usedAt": time.Now().Unix(),
		"usedBy": usedBy,
	}}
	res, err := dbService.collectionRefEntryCodes(instanceID).UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.ModifiedCount < 1 {
		return errors.New("not modified")
	}
	return nil
}
