package types

type DBConfig struct {
	URI             string
	DBNamePrefix    string
	Timeout         int
	MaxPoolSize     uint64
	IdleConnTimeout int
}

type SamplerConfig struct {
	SampleFilePath   string
	TargetSamples    int // maximum sample count target
	OpenSlotsAtStart int // number of slots open at start of the sample interval
}
