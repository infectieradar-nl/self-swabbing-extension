package db

import (
	"testing"

	"github.com/infectieradar-nl/self-swabbing-extension/pkg/sampler"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

const testSlotNamespace = "test_default_self-swabbing-ext.used-slots"

func slotDB(mt *mtest.T) *SelfSwabbingExtDBService {
	return &SelfSwabbingExtDBService{DBClient: mt.Client, timeout: 5, DBNamePrefix: "test_"}
}

func TestReserveSlotStoresSelectionSource(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	for _, source := range []string{sampler.SelectionSourceRandom, sampler.SelectionSourcePienter} {
		mt.Run(source, func(mt *mtest.T) {
			mt.AddMockResponses(
				mtest.CreateCursorResponse(0, testSlotNamespace, mtest.FirstBatch),
				mtest.CreateSuccessResponse(),
			)
			if err := slotDB(mt).ReserveSlot("default", "participant", source); err != nil {
				mt.Fatal(err)
			}
			for _, event := range mt.GetAllStartedEvents() {
				if event.CommandName != "insert" {
					continue
				}
				documents, err := event.Command.Lookup("documents").Array().Values()
				if err != nil || len(documents) != 1 {
					mt.Fatalf("unexpected insert: %s", event.Command)
				}
				document := documents[0].Document()
				if document.Lookup("selectionSource").StringValue() != source ||
					document.Lookup("status").StringValue() != USED_SLOT_STATUS_RESERVED ||
					document.Lookup("participantID").StringValue() != "participant" {
					mt.Fatalf("unexpected reservation: %s", document)
				}
				return
			}
			mt.Fatal("no reservation inserted")
		})
	}
}

func TestReserveSlotPreservesExistingReservation(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	for _, source := range []string{sampler.SelectionSourceRandom, sampler.SelectionSourcePienter, ""} {
		name := source
		if name == "" {
			name = "legacy"
		}
		mt.Run(name, func(mt *mtest.T) {
			document := bson.D{{Key: "participantID", Value: "participant"}, {Key: "status", Value: USED_SLOT_STATUS_RESERVED}}
			if source != "" {
				document = append(document, bson.E{Key: "selectionSource", Value: source})
			}
			mt.AddMockResponses(mtest.CreateCursorResponse(0, testSlotNamespace, mtest.FirstBatch, document))
			if err := slotDB(mt).ReserveSlot("default", "participant", sampler.SelectionSourcePienter); err != nil {
				mt.Fatal(err)
			}
			for _, event := range mt.GetAllStartedEvents() {
				if event.CommandName != "find" {
					mt.Fatalf("existing reservation was changed by %s", event.CommandName)
				}
			}
		})
	}
}

func TestReserveSlotReturnsDatabaseErrors(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("lookup error does not insert", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{Code: 123, Message: "lookup failed"}))
		if err := slotDB(mt).ReserveSlot("default", "participant", sampler.SelectionSourcePienter); err == nil {
			mt.Fatal("expected lookup error")
		}
		for _, event := range mt.GetAllStartedEvents() {
			if event.CommandName == "insert" {
				mt.Fatal("inserted after a failed lookup")
			}
		}
	})
	mt.Run("insert error", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, testSlotNamespace, mtest.FirstBatch),
			mtest.CreateWriteErrorsResponse(mtest.WriteError{Index: 0, Code: 11000, Message: "insert failed"}),
		)
		if err := slotDB(mt).ReserveSlot("default", "participant", sampler.SelectionSourcePienter); err == nil {
			mt.Fatal("expected insert error")
		}
	})
}

func TestPienterReservationUsesExistingConfirmationAndCancellation(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("confirm preserves selection source", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "value", Value: bson.D{
			{Key: "participantID", Value: "participant"},
			{Key: "status", Value: USED_SLOT_STATUS_CONFIRMED},
			{Key: "selectionSource", Value: sampler.SelectionSourcePienter},
		}}))
		if err := slotDB(mt).ConfirmSlot("default", "participant"); err != nil {
			mt.Fatal(err)
		}
		event := mt.GetStartedEvent()
		if event == nil || event.CommandName != "findAndModify" {
			mt.Fatal("no confirmation command")
		}
		update := event.Command.Lookup("update").Document().Lookup("$set").Document()
		var fields bson.M
		if err := bson.Unmarshal(update, &fields); err != nil || len(fields) != 1 || fields["status"] != USED_SLOT_STATUS_CONFIRMED {
			mt.Fatalf("confirmation changed more than status: %s", update)
		}
	})
	mt.Run("cancel releases reservation", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "value", Value: bson.D{
			{Key: "participantID", Value: "participant"},
			{Key: "status", Value: USED_SLOT_STATUS_RESERVED},
			{Key: "selectionSource", Value: sampler.SelectionSourcePienter},
		}}))
		if err := slotDB(mt).CancelSlotReservation("default", "participant"); err != nil {
			mt.Fatal(err)
		}
		event := mt.GetStartedEvent()
		if event == nil || event.CommandName != "findAndModify" || !event.Command.Lookup("remove").Boolean() {
			mt.Fatal("reservation was not removed")
		}
	})
}

func TestUsedSlotCountIncludesEverySelectionSource(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("shared count", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(0, testSlotNamespace, mtest.FirstBatch, bson.D{{Key: "n", Value: int32(2)}}))
		count, err := slotDB(mt).GetUsedSlotsCountSince("default", 100)
		if err != nil || count != 2 {
			mt.Fatalf("count=%d, error=%v", count, err)
		}
		event := mt.GetStartedEvent()
		stages, err := event.Command.Lookup("pipeline").Array().Values()
		if err != nil || len(stages) == 0 {
			mt.Fatal("missing count pipeline")
		}
		match := stages[0].Document().Lookup("$match").Document()
		var fields bson.M
		if err := bson.Unmarshal(match, &fields); err != nil || len(fields) != 1 || fields["time"] == nil {
			mt.Fatalf("count unexpectedly filters reservations: %s", match)
		}
	})
}
