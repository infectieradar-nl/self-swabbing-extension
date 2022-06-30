package types

import "go.mongodb.org/mongo-driver/bson/primitive"

type ValidationCode struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Code       string             `bson:"code,omitempty" json:"code,omitempty"`
	UploadedAt int64              `bson:"uploadedAt" json:"uploadedAt"`
	UsedAt     int64              `bson:"usedAt" json:"usedAt"`
	UsedBy     string             `bson:"usedBy" json:"usedBy"`
}

type NewCodeList struct {
	Codes []string `json:"codes"`
}
