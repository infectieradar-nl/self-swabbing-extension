package db

import (
	"github.com/coneno/logger"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (dbService *SelfSwabbingExtDBService) collectionRefStreptokidsControls() *mongo.Collection {
	return dbService.DBClient.Database(dbService.DBNamePrefix + "streptokids").Collection("control-contacts")
}

func (dbService *SelfSwabbingExtDBService) CreateIndexesForStreptokids() {
	ctx, cancel := dbService.getContext()
	defer cancel()

	_, err := dbService.collectionRefStreptokidsControls().Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{
				"invitedAt": -1,
			},
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}

	_, err = dbService.collectionRefStreptokidsControls().Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{
				"submittedAt": 1,
			},
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}

	_, err = dbService.collectionRefStreptokidsControls().Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.D{
				{Key: "submittedAt", Value: -1},
				{Key: "invitedAt", Value: 1},
			},
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}
}

func (dbService *SelfSwabbingExtDBService) StreptokidsAddControlContact(contact types.StreptokidsControlRegistration) (string, error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	res, err := dbService.collectionRefStreptokidsControls().InsertOne(ctx, contact)
	if err != nil {
		return "", err
	}
	id := res.InsertedID.(primitive.ObjectID)
	return id.Hex(), err
}
