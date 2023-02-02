package db

import (
	"github.com/coneno/logger"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (dbService *SelfSwabbingExtDBService) StreptokidsFetchControlContacts(since int64, includeInvited bool) (contacts []types.StreptokidsControlRegistration, err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{"submittedAt": bson.M{"$gt": since}}
	if !includeInvited {
		filter["invitedAt"] = bson.M{"$lt": 1}
	}

	opts := &options.FindOptions{
		Sort: bson.D{
			primitive.E{Key: "submittedAt", Value: -1},
		},
	}

	cur, err := dbService.collectionRefStreptokidsControls().Find(
		ctx,
		filter,
		opts,
	)
	if err != nil {
		return contacts, err
	}
	defer cur.Close(ctx)

	contacts = []types.StreptokidsControlRegistration{}
	for cur.Next(ctx) {
		var result types.StreptokidsControlRegistration
		err := cur.Decode(&result)
		if err != nil {
			return contacts, err
		}

		contacts = append(contacts, result)
	}
	if err := cur.Err(); err != nil {
		return contacts, err
	}

	return contacts, nil
}
