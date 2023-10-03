package isolate

type IsolateMetrics struct {
	TimeSec      float64
	TimeWallSec  float64
	MaxRssKb     int64
	CswVoluntary int64
	CswForced    int64
	CgMemKb      int64
	ExitCode     int64
	Status       string
	Message      string
}
