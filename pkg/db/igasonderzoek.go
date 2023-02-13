package db

import (
	"time"

	"github.com/coneno/logger"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (dbService *SelfSwabbingExtDBService) collectionRefIgasonderzoekControls() *mongo.Collection {
	return dbService.DBClient.Database(dbService.DBNamePrefix + "igasonderzoek").Collection("control-contacts")
}

func (dbService *SelfSwabbingExtDBService) collectionRefIgasonderzoekControlCodes() *mongo.Collection {
	return dbService.DBClient.Database(dbService.DBNamePrefix + "igasonderzoek").Collection("control-codes")
}

func (dbService *SelfSwabbingExtDBService) CreateIndexesForIgasonderzoek() {
	ctx, cancel := dbService.getContext()
	defer cancel()

	_, err := dbService.collectionRefIgasonderzoekControls().Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{
				"invitedAt": -1,
			},
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}

	_, err = dbService.collectionRefIgasonderzoekControls().Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{
				"submittedAt": 1,
			},
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}

	_, err = dbService.collectionRefIgasonderzoekControls().Indexes().CreateOne(
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

	_, err = dbService.collectionRefIgasonderzoekControlCodes().Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{
				"code": 1,
			},
			Options: options.Index().SetUnique(true),
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}

	_, err = dbService.collectionRefIgasonderzoekControlCodes().Indexes().CreateOne(
		ctx, mongo.IndexModel{
			Keys: bson.M{
				"createdAt": 1,
			},
		},
	)
	if err != nil {
		logger.Error.Println(err)
	}
}

func (dbService *SelfSwabbingExtDBService) IgasonderzoekAddControlContact(contact types.IgasonderzoekControlRegistration) (string, error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	res, err := dbService.collectionRefIgasonderzoekControls().InsertOne(ctx, contact)
	if err != nil {
		return "", err
	}
	id := res.InsertedID.(primitive.ObjectID)
	return id.Hex(), err
}

func (dbService *SelfSwabbingExtDBService) IgasonderzoekFetchControlContacts(since int64, includeInvited bool) (contacts []types.IgasonderzoekControlRegistration, err error) {
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

	cur, err := dbService.collectionRefIgasonderzoekControls().Find(
		ctx,
		filter,
		opts,
	)
	if err != nil {
		return contacts, err
	}
	defer cur.Close(ctx)

	contacts = []types.IgasonderzoekControlRegistration{}
	for cur.Next(ctx) {
		var result types.IgasonderzoekControlRegistration
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

func (dbService *SelfSwabbingExtDBService) IgasonderzoekFindOneControlContact(id string) (contact types.IgasonderzoekControlRegistration, err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return contact, err
	}
	filter := bson.M{"_id": _id}

	if err = dbService.collectionRefIgasonderzoekControls().FindOne(
		ctx,
		filter,
		options.FindOne(),
	).Decode(&contact); err != nil {
		return contact, err
	}

	return contact, err
}

func (dbService *SelfSwabbingExtDBService) IgasonderzoekMarkControlContactInvited(id string) error {
	ctx, cancel := dbService.getContext()
	defer cancel()

	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": _id}
	update := bson.M{"$set": bson.M{
		"invitedAt": time.Now().Unix(),
	}}
	_, err = dbService.collectionRefIgasonderzoekControls().UpdateOne(ctx, filter, update)
	return err
}

func (dbService *SelfSwabbingExtDBService) IgasonderzoekAddControlCode(code string) (string, error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	newEntryCode := types.IgasonderzoekControlCode{
		Code:      code,
		CreatedAt: time.Now().Unix(),
	}

	res, err := dbService.collectionRefIgasonderzoekControlCodes().InsertOne(ctx, newEntryCode)
	if err != nil {
		return "", err
	}
	id := res.InsertedID.(primitive.ObjectID)
	return id.Hex(), err
}

func (dbService *SelfSwabbingExtDBService) IgasonderzoekFindControlCode(code string) (entryCode types.IgasonderzoekControlCode, err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{
		"code": code,
	}

	if err = dbService.collectionRefIgasonderzoekControlCodes().FindOne(
		ctx,
		filter,
		options.FindOne(),
	).Decode(&entryCode); err != nil {
		return entryCode, err
	}

	return entryCode, err
}

func (dbService *SelfSwabbingExtDBService) IgasonderzoekDeleteContactsBefore(before int64) (count int64, err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{"submittedAt": bson.M{"$lt": before}}

	res, err := dbService.collectionRefIgasonderzoekControls().DeleteMany(ctx, filter)
	return res.DeletedCount, err
}

func (dbService *SelfSwabbingExtDBService) IgasonderzoekDeleteControlCodesBefore(before int64) (count int64, err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{"createdAt": bson.M{"$lt": before}}

	res, err := dbService.collectionRefIgasonderzoekControlCodes().DeleteMany(ctx, filter)
	return res.DeletedCount, err
}

func (dbService *SelfSwabbingExtDBService) IgasonderzoekDeleteControlCode(code string) (count int64, err error) {
	ctx, cancel := dbService.getContext()
	defer cancel()

	filter := bson.M{"code": code}

	res, err := dbService.collectionRefIgasonderzoekControlCodes().DeleteOne(ctx, filter)
	return res.DeletedCount, err
}
