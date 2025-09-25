package isolate

import (
	"fmt"
	"log"
	"log/slog"
	"strings"
)

type Metrics struct {
	CpuMillis    int64
	WallMillis   int64
	MaxRssKb     int64
	CswVoluntary int64
	CswForced    int64
	CgMemKb      int64
	ExitCode     int64
	Status       *string
	Message      *string
	Killed       int64
	ExitSig      *int64
	FullReport   string
}

func parseMetaFile(metaFileBytes []byte) (*Metrics, error) {
	metrics := &Metrics{}
	metrics.FullReport = string(metaFileBytes)
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
			log.Println("Error parsing meta file: ", string(metaFileBytes))
			return nil, err
		}
	}
	return metrics, nil
}

func parseLine(key, value string, metrics *Metrics) error {
	switch key {
	case "time":
		var timeSec float64
		if err := sscanfErr(fmt.Sscanf(value, "%f", &timeSec)); err != nil {
			return err
		}
		metrics.CpuMillis = int64(timeSec * 1000.0)
	case "time-wall":
		var timeWallSec float64
		if err := sscanfErr(fmt.Sscanf(value, "%f", &timeWallSec)); err != nil {
			return err
		}
		metrics.WallMillis = int64(timeWallSec * 1000.0)
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
		metrics.Status = &value
	case "killed":
		return sscanfErr(fmt.Sscanf(value, "%d", &metrics.Killed))
	case "message":
		metrics.Message = &value
	case "exitsig":
		var exitsig int64
		if err := sscanfErr(fmt.Sscanf(value, "%d", &exitsig)); err != nil {
			return err
		}
		metrics.ExitSig = &exitsig
	case "":
		// ignore
	default:
		slog.Info("unknown meta file line", slog.String("line", key+":"+value))
		return fmt.Errorf("unknown meta file line: %s", key+":"+value)
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
