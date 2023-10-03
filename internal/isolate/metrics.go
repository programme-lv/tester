package isolate

import (
	"fmt"
	"log/slog"
	"strings"
)

type Metrics struct {
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

func parseMetaFile(metaFileBytes []byte) (*Metrics, error) {
	metrics := &Metrics{}
	for _, line := range strings.Split(string(metaFileBytes), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			slog.Info("invalid meta file line", slog.String("line", line))
			return nil, fmt.Errorf("invalid meta file line: %s", line)
		}

		key, value := parts[0], parts[1]
		if err := parseLine(key, value, metrics); err != nil {
			return nil, err
		}
	}
	return metrics, nil
}

func parseLine(key, value string, metrics *Metrics) error {
	switch key {
	case "time":
		return sscanfErr(fmt.Sscanf(value, "%f", &metrics.TimeSec))
	case "time-wall":
		return sscanfErr(fmt.Sscanf(value, "%f", &metrics.TimeWallSec))
	case "max-rss":
		return sscanfErr(fmt.Sscanf(value, "%d", &metrics.MaxRssKb))
	case "csw-voluntary":
		return sscanfErr(fmt.Sscanf(value, "%d", &metrics.CswVoluntary))
	case "csw-forced":
		return sscanfErr(fmt.Sscanf(value, "%d", &metrics.CswForced))
	case "cg-mem":
		return sscanfErr(fmt.Sscanf(value, "%d", &metrics.CgMemKb))
	case "exitcode":
		return sscanfErr(fmt.Sscanf(value, "%d", &metrics.ExitCode))
	case "status":
		metrics.Status = value
	case "message":
		metrics.Message = value
	case "":
		// ignore
	default:
		slog.Info("unknown meta file line", slog.String("line", key+":"+value))
	}
	return nil
}

func sscanfErr(_ int, err error) error {
	return err
}

/*
time:0.112
time-wall:0.103
max-rss:18984
csw-voluntary:1513
csw-forced:23
cg-mem:38248
exitcode:0
*/

/*
time:0.002
time-wall:0.045
max-rss:2624
csw-voluntary:6
csw-forced:2
cg-mem:38248
exitcode:2
status:RE
message:Exited with error status 2
*/

/*
time:0.115
time-wall:0.125
max-rss:18444
csw-voluntary:1597
csw-forced:28
cg-mem:38248
status:TO
message:Time limit exceeded
*/
