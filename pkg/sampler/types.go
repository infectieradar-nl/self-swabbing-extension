package sampler

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Sampler struct {
	instanceID string
	dbService  SamplerDBService
	SlotCurve  SlotCurve
}

type OpenSlots struct {
	T     int `bson:"t" json:"t"`
	Value int `bson:"value" json:"value"`
}

type SlotCurve struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	IntervalStart time.Time          `bson:"intervalStart,omitempty" json:"intervalStart,omitempty"`
	OpenSlots     []OpenSlots        `bson:"openSlots,omitempty" json:"openSlots,omitempty"`
}

type SamplerDBService interface {
	LoadLatestSlotCurve(instanceID string) (res SlotCurve, err error)
	SaveNewSlotCurve(instanceID string, res SlotCurve) (err error)
	GetUsedSlotsSince(instanceID string, ref time.Time) (count int, err error)
}
