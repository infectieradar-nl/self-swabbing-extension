package sampler

import (
	"errors"
	"testing"
	"time"
)

type selectionDB struct {
	used         int64
	reservations map[string]string
	reserveErr   error
}

func (db *selectionDB) LoadLatestSlotCurve(string) (SlotCurve, error) { return SlotCurve{}, nil }
func (db *selectionDB) SaveNewSlotCurve(string, SlotCurve) error      { return nil }
func (db *selectionDB) GetUsedSlotsCountSince(string, int64) (int64, error) {
	return db.used, nil
}
func (db *selectionDB) ReserveSlot(_ string, participantID string, source string) error {
	if db.reserveErr != nil {
		return db.reserveErr
	}
	if _, exists := db.reservations[participantID]; !exists {
		db.reservations[participantID] = source
		db.used++
	}
	return nil
}

func selectionSampler(db *selectionDB) *Sampler {
	s := NewSampler("default", db)
	s.SlotCurve = SlotCurve{
		IntervalStart: time.Now().Unix() - 60,
		OpenSlots:     []OpenSlots{{T: 0, Value: 1}},
	}
	return s
}

func TestSelectParticipant(t *testing.T) {
	for _, tc := range []struct {
		name     string
		used     int64
		pienter  bool
		selected bool
		source   string
	}{
		{"ordinary with capacity", 0, false, true, SelectionSourceRandom},
		{"Pienter with capacity stays random", 0, true, true, SelectionSourceRandom},
		{"ordinary at capacity", 1, false, false, ""},
		{"Pienter at capacity", 1, true, true, SelectionSourcePienter},
		{"Pienter above capacity", 2, true, true, SelectionSourcePienter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &selectionDB{used: tc.used, reservations: map[string]string{}}
			selected, err := selectionSampler(db).SelectParticipant("participant", tc.pienter)
			if err != nil || selected != tc.selected {
				t.Fatalf("selected=%v, error=%v; want selected=%v", selected, err, tc.selected)
			}
			if source := db.reservations["participant"]; source != tc.source {
				t.Fatalf("source=%q; want %q", source, tc.source)
			}
			wantUsed := tc.used
			if tc.selected {
				wantUsed++
			}
			if db.used != wantUsed {
				t.Fatalf("used=%d; want %d", db.used, wantUsed)
			}
		})
	}
}

func TestPienterReservationConsumesOrdinaryCapacity(t *testing.T) {
	db := &selectionDB{used: 1, reservations: map[string]string{}}
	s := selectionSampler(db)
	if selected, err := s.SelectParticipant("pienter", true); !selected || err != nil {
		t.Fatalf("Pienter selected=%v, error=%v", selected, err)
	}
	if selected, err := s.SelectParticipant("ordinary", false); selected || err != nil {
		t.Fatalf("ordinary selected=%v, error=%v", selected, err)
	}
	infos := s.GetSamplerInfos()
	if infos.UsedSlots != 2 || infos.AvailableSlots != -1 {
		t.Fatalf("unexpected shared budget: %+v", infos)
	}
	// As the curve opens more places, the Pienter reservation is still counted.
	s.SlotCurve.OpenSlots[0].Value = 2
	if selected, err := s.SelectParticipant("ordinary", false); selected || err != nil {
		t.Fatalf("Pienter reservation was not counted: selected=%v, error=%v", selected, err)
	}
	s.SlotCurve.OpenSlots[0].Value = 3
	if selected, err := s.SelectParticipant("ordinary", false); !selected || err != nil {
		t.Fatalf("new capacity unavailable: selected=%v, error=%v", selected, err)
	}
}

func TestSelectParticipantRequiresSuccessfulReservation(t *testing.T) {
	for _, pienter := range []bool{false, true} {
		db := &selectionDB{reservations: map[string]string{}, reserveErr: errors.New("database unavailable")}
		if pienter {
			db.used = 1
		}
		selected, err := selectionSampler(db).SelectParticipant("participant", pienter)
		if selected || !errors.Is(err, db.reserveErr) || len(db.reservations) != 0 {
			t.Fatalf("Pienter=%v: selected=%v, error=%v, reservations=%v", pienter, selected, err, db.reservations)
		}
	}
}
