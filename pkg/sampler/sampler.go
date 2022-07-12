package sampler

import (
	"encoding/csv"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/coneno/logger"
)

func NewSampler(
	instanceID string,
	dbService SamplerDBService,
) *Sampler {
	return &Sampler{
		instanceID: instanceID,
		dbService:  dbService,
	}
}

func (s Sampler) openSlotTargetNow() int {
	if len(s.SlotCurve.OpenSlots) < 1 {
		logger.Debug.Println("slot curve is not available")
		return 0
	}

	currentT := time.Now().Unix() - s.SlotCurve.IntervalStart

	openSlots := s.SlotCurve.OpenSlots[0].Value
	for _, slotTarget := range s.SlotCurve.OpenSlots {
		if int64(slotTarget.T) > currentT {
			break
		}
		openSlots = slotTarget.Value
	}
	logger.Debug.Printf("Target slot count at %d: %d", currentT, openSlots)
	return openSlots
}

func (s Sampler) getUsedSlotsCountNow() int {
	count, err := s.dbService.GetUsedSlotsCountSince(s.instanceID, s.SlotCurve.IntervalStart)
	if err != nil {
		logger.Debug.Printf("error when fetching used slot count: %v", err)
		return 0
	}
	return int(count)
}

func (s Sampler) HasAvailableFreeSlots() bool {
	openSlotsTarget := s.openSlotTargetNow()
	usedSlots := s.getUsedSlotsCountNow()
	availableSlots := openSlotsTarget - usedSlots

	return availableSlots > 0
}

func (s *Sampler) LoadSlotCurveFromDB() {
	sc, err := s.dbService.LoadLatestSlotCurve(s.instanceID)
	if err != nil {
		logger.Debug.Printf("error when trying to load slot curve from DB: %v", err)
		return
	}
	s.SlotCurve = sc
}

func (s Sampler) SaveSlotCurveToDB() {
	err := s.dbService.SaveNewSlotCurve(s.instanceID, s.SlotCurve)
	if err != nil {
		logger.Error.Printf("unexpected error when saving slot curve to DB: %v", err)
	}
}

func (s *Sampler) InitFromSampleCSV(filePath string, target int, minVal int) {
	data := readCsvFile(filePath)[1:]

	rand.Seed(time.Now().UnixNano())

	n := target - minVal
	samples := make([]int, n)
	for i := 0; i < n; i++ {
		index := rand.Intn(len(data) - 1)
		value, err := strconv.Atoi(data[index][1])
		if err != nil {
			logger.Error.Fatal("wrong value: " + err.Error())
		}
		samples[i] = value * 60
	}
	sort.Ints(samples)

	openSlots := []OpenSlots{
		{T: 0, Value: minVal},
	}

	for _, s := range samples {
		lastSlotOpening := openSlots[len(openSlots)-1]
		if s == lastSlotOpening.T {
			openSlots[len(openSlots)-1].Value += 1
		} else {
			openSlots = append(openSlots,
				OpenSlots{T: s, Value: lastSlotOpening.Value + 1},
			)
		}
	}

	s.SlotCurve = SlotCurve{
		IntervalStart: getStartOfTheWeek(),
		OpenSlots:     openSlots,
	}
}

func (s Sampler) NeedsRefresh() bool {
	sY, sW := time.Unix(s.SlotCurve.IntervalStart, 0).ISOWeek()
	nY, nW := time.Now().ISOWeek()
	if nY != sY || sW != nW {
		return true
	}
	return false
}

func getStartOfTheWeek() int64 {
	t := time.Now()
	year, month, day := t.Date()
	t = time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	for t.Weekday() != time.Monday { // iterate back to Monday
		t = t.AddDate(0, 0, -1)
	}
	return t.Unix()
}

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		logger.Error.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		logger.Error.Fatal("Unable to parse file as CSV for "+filePath, err)
	}
	return records
}
