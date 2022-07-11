package db

import (
	"errors"
	"time"

	"github.com/infectieradar-nl/self-swabbing-extension/pkg/sampler"
)

func (dbService *SelfSwabbingExtDBService) LoadLatestSlotCurve(instanceID string) (res sampler.SlotCurve, err error) {
	return res, errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) SaveNewSlotCurve(instanceID string, res sampler.SlotCurve) (err error) {
	return errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) GetUsedSlotsSince(instanceID string, ref time.Time) (count int, err error) {
	return 0, errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) ReserveSlot(instanceID string, participantID string) error {
	return errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) CancelSlotReservation(instanceID string, participantID string) error {
	return errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) ConfirmSlot(instanceID string, participantID string) error {
	return errors.New("unimplemented")
}

func (dbService *SelfSwabbingExtDBService) CleanUpExpiredSlotReservations(instanceID string) error {
	return errors.New("unimplemented")
}
