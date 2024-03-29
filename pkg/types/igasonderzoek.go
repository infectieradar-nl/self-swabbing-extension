package types

import "go.mongodb.org/mongo-driver/bson/primitive"

type IgasonderzoekControlRegistration struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SubmittedAt     int64              `bson:"submittedAt" json:"submittedAt"`
	Email           string             `bson:"email" json:"email"`
	ControlResponse string             `bson:"controlResponse" json:"controlResponse"`
	InvitedAt       int64              `bson:"invitedAt" json:"invitedAt"`
	ControleCode    string             `bson:"controlCode" json:"controlCode"`
	Children        []ChildInfos       `bson:"children" json:"children"`
}

type ChildInfos struct {
	Birthyear  int    `bson:"birthyear" json:"birthyear"`
	Birthmonth int    `bson:"birthmonth" json:"birthmonth"`
	Gender     string `bson:"gender" json:"gender"`
}

type IgasonderzoekControlCode struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Code      string             `bson:"code,omitempty" json:"code,omitempty"`
	CreatedAt int64              `bson:"createdAt" json:"createdAt"`
}
