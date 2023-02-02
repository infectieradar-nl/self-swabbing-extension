package types

import "go.mongodb.org/mongo-driver/bson/primitive"

type StreptokidsControlRegistration struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SubmittedAt     int64              `bson:"submittedAt" json:"submittedAt"`
	Email           string             `bson:"email" json:"email"`
	Age             int                `bson:"age" json:"age"`
	ControlResponse string             `bson:"controlResponse" json:"controlResponse"`
	InvitedAt       int64              `bson:"invitedAt" json:"invitedAt"`
}
