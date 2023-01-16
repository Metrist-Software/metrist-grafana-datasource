package plugin

import "time"

func strToTime(str string) time.Time {
	time, _ := time.Parse(time.RFC3339, str)
	return time
}
