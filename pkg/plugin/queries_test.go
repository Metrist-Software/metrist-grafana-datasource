package plugin

import (
	"testing"
	"time"
)

func TestEnsureTelemetryRequestWithinLast90Days(t *testing.T) {
	if err := ensureTelemetryRequestWithinLast90Days(time.Now().AddDate(0, 0, -89)); err != nil {
		t.Errorf("TestEnsureTelemetryRequestWithinLast90Days() returned an error when it was not expected")
	}

	if err := ensureTelemetryRequestWithinLast90Days(time.Now().AddDate(0, 0, -91)); err == nil {
		t.Errorf("TestEnsureTelemetryRequestWithinLast90Days() did not return an error when it was expected")
	}
}
