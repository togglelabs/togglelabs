package models

import (
	"context"
	"errors"
	"time"

	"github.com/Roll-Play/togglelabs/pkg/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const FeatureFlagCollectionName = "feature_flag"

type FeatureFlagModel struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func NewFeatureFlagModel(db *mongo.Database) *FeatureFlagModel {
	return &FeatureFlagModel{
		db:         db,
		collection: db.Collection(FeatureFlagCollectionName),
	}
}

type RevisionStatus = string

const (
	Live     RevisionStatus = "live"
	Draft    RevisionStatus = "draft"
	Archived RevisionStatus = "archived"
)

type Rule struct {
	Predicate string `json:"predicate" bson:"predicate" validate:"required"`
	Value     string `json:"value" bson:"value" validate:"required"`
	Env       string `json:"env" bson:"env" validate:"required"`
	IsEnabled bool   `json:"is_enabled" bson:"is_enabled" validate:"required,boolean"`
}

type Revision struct {
	ID           primitive.ObjectID `json:"_id,omitempty" bson:"_id"`
	UserID       primitive.ObjectID `json:"user_id" bson:"user_id"`
	Status       RevisionStatus     `json:"status" bson:"status"`
	DefaultValue string             `json:"default_value" bson:"default_value"`
	Rules        []Rule
}

type FlagType = string

const (
	Boolean FlagType = "boolean"
	JSON    FlagType = "json"
	String  FlagType = "string"
	Number  FlagType = "number"
)

type FeatureFlagRecord struct {
	ID             primitive.ObjectID `json:"_id,omitempty" bson:"_id"`
	OrganizationID primitive.ObjectID `json:"organization_id" bson:"organization_id"`
	UserID         primitive.ObjectID `json:"user_id" bson:"user_id"`
	Version        int                `json:"version" bson:"version"`
	Name           string             `json:"name" bson:"name"`
	Type           FlagType           `json:"type" bson:"type"`
	Revisions      []Revision         `json:"revisions" bson:"revisions"`
	storage.Timestamps
}

func NewFeatureFlagRecord(
	name,
	defaultValue string,
	flagType FlagType,
	rules []Rule,
	organizationID,
	userID primitive.ObjectID,
) *FeatureFlagRecord {
	return &FeatureFlagRecord{
		OrganizationID: organizationID,
		UserID:         userID,
		Version:        1,
		Name:           name,
		Type:           flagType,
		Revisions: []Revision{
			{
				ID:           primitive.NewObjectID(),
				UserID:       userID,
				Status:       Draft,
				DefaultValue: defaultValue,
				Rules:        rules,
			},
		},
		Timestamps: storage.Timestamps{
			CreatedAt: primitive.NewDateTimeFromTime(time.Now().UTC()),
			UpdatedAt: primitive.NewDateTimeFromTime(time.Now().UTC()),
		},
	}
}

func NewRevisionRecord(defaultValue string, rules []Rule, userID primitive.ObjectID) *Revision {
	return &Revision{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		Status:       Draft,
		DefaultValue: defaultValue,
		Rules:        rules,
	}
}

func (ffm *FeatureFlagModel) InsertOne(ctx context.Context, rec *FeatureFlagRecord) (primitive.ObjectID, error) {
	rec.ID = primitive.NewObjectID()
	result, err := ffm.collection.InsertOne(ctx, rec)
	if err != nil {
		return primitive.NilObjectID, err
	}

	objectID, ok := result.InsertedID.(primitive.ObjectID)

	if !ok {
		return primitive.NilObjectID, errors.New("unable to assert type of objectID")
	}

	return objectID, nil
}

func (ffm *FeatureFlagModel) FindByID(ctx context.Context, id primitive.ObjectID) (*FeatureFlagRecord, error) {
	record := new(FeatureFlagRecord)
	if err := ffm.collection.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(record); err != nil {
		return nil, err
	}
	return record, nil
}

var EmptyFeatureRecordList = []FeatureFlagRecord{}

func (ffm *FeatureFlagModel) FindMany(
	ctx context.Context,
	organizationID primitive.ObjectID,
) ([]FeatureFlagRecord, error) {
	records := make([]FeatureFlagRecord, 0)
	cursor, err := ffm.collection.Find(ctx, bson.D{{Key: "organization_id", Value: organizationID}})
	if err != nil {
		return EmptyFeatureRecordList, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		record := new(FeatureFlagRecord)
		if err := cursor.Decode(record); err != nil {
			return EmptyFeatureRecordList, err
		}

		records = append(records, *record)
	}

	return records, nil
}

func (ffm *FeatureFlagModel) PushOne(
	ctx context.Context,
	id primitive.ObjectID,
	newValues bson.M,
) (primitive.ObjectID, error) {
	filter := bson.D{{Key: "_id", Value: id}}
	update := bson.D{{Key: "$push", Value: newValues}}
	_, err := ffm.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return primitive.ObjectID{}, err
	}

	return id, nil
}

func (ffm *FeatureFlagModel) UpdateOne(
	ctx context.Context,
	filters,
	newValues bson.D,
) (primitive.ObjectID, error) {
	update := bson.D{{Key: "$set", Value: newValues}}
	_, err := ffm.collection.UpdateOne(ctx, filters, update)
	if err != nil {
		return primitive.ObjectID{}, err
	}

	return primitive.NilObjectID, nil
}
