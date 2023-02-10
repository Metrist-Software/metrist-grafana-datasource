// Some helpers around the auto generated openapi.go structs for easy grafana data.Frame creation across auto created types
package internal

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type FrameData interface {
	GetTimestamp() (time.Time, error)
	GetGraphVals(timestamp time.Time) []any
	GetTableVals(timestamp time.Time) []any
	GetKey() string
	GetGraphFrameDefinition() data.Frame
	GetTableFrameDefinition() data.Frame

	getLabels() map[string]string
}

// Monitor Errors
func (errorCount *MonitorErrorCount) GetTimestamp() (time.Time, error) {
	return time.Parse(time.RFC3339, *errorCount.Timestamp)
}

func (errorCount *MonitorErrorCount) GetGraphVals(timestamp time.Time) []any {
	return []any{timestamp, int64(*errorCount.Count)}
}

func (errorCount *MonitorErrorCount) GetTableVals(timestamp time.Time) []any {
	return []any{timestamp, int64(*errorCount.Count), *errorCount.Instance, *errorCount.Check, *errorCount.MonitorLogicalName}
}

func (errorCount *MonitorErrorCount) GetKey() string {
	return fmt.Sprintf("%s-%s-%s", *errorCount.Instance, *errorCount.Check, *errorCount.MonitorLogicalName)
}

func (errorCount *MonitorErrorCount) GetGraphFrameDefinition() data.Frame {
	return data.Frame{
		Fields: []*data.Field{
			data.NewField("time", nil, make([]time.Time, 0)),
			data.NewField("count", errorCount.getLabels(), make([]int64, 0)),
		},
		Meta: &data.FrameMeta{
			Type:                   data.FrameTypeTimeSeriesMulti,
			PreferredVisualization: data.VisTypeGraph,
		},
	}
}

func (errorCount *MonitorErrorCount) GetTableFrameDefinition() data.Frame {
	return data.Frame{
		Fields: []*data.Field{
			data.NewField("time", nil, []time.Time{}),
			data.NewField("count", nil, []int64{}),
			data.NewField("instance", nil, []string{}),
			data.NewField("check", nil, []string{}),
			data.NewField("monitor", nil, []string{}),
		},
		Meta: &data.FrameMeta{
			Type:                   data.FrameTypeTimeSeriesWide,
			PreferredVisualization: data.VisTypeTable,
		},
	}
}

func (errorCount *MonitorErrorCount) getLabels() map[string]string {
	return map[string]string{"instance": *errorCount.Instance, "check": *errorCount.Check, "monitor": *errorCount.MonitorLogicalName}
}

// Monitor Telemetry
func (te *MonitorTelemetry) GetTimestamp() (time.Time, error) {
	return time.Parse(time.RFC3339, *te.Timestamp)
}

func (te *MonitorTelemetry) GetGraphVals(timestamp time.Time) []any {
	return []any{timestamp, *te.Value}
}

func (te *MonitorTelemetry) GetTableVals(timestamp time.Time) []any {
	return []any{timestamp, *te.Value, *te.Instance, *te.Check, *te.MonitorLogicalName}
}

func (te *MonitorTelemetry) GetKey() string {
	return fmt.Sprintf("%s-%s-%s", *te.Instance, *te.Check, *te.MonitorLogicalName)
}

func (te *MonitorTelemetry) GetGraphFrameDefinition() data.Frame {
	return data.Frame{
		Fields: []*data.Field{
			data.NewField("time", nil, make([]time.Time, 0)),
			data.NewField("response time (ms)", te.getLabels(), make([]float32, 0)),
		},
		Meta: &data.FrameMeta{
			Type:                   data.FrameTypeTimeSeriesMulti,
			PreferredVisualization: data.VisTypeGraph,
		},
	}
}

func (te *MonitorTelemetry) GetTableFrameDefinition() data.Frame {
	return data.Frame{
		Fields: []*data.Field{
			data.NewField("time", nil, []time.Time{}),
			data.NewField("response time (ms)", nil, []float32{}),
			data.NewField("instance", nil, []string{}),
			data.NewField("check", nil, []string{}),
			data.NewField("monitor", nil, []string{}),
		},
		Meta: &data.FrameMeta{
			Type:                   data.FrameTypeTimeSeriesWide,
			PreferredVisualization: data.VisTypeTable,
		},
	}
}

func (te *MonitorTelemetry) getLabels() map[string]string {
	return map[string]string{"instance": *te.Instance, "check": *te.Check, "monitor": *te.MonitorLogicalName}
}

// Status Page Changes
func (spc *StatusPageComponentChange) GetTimestamp() (time.Time, error) {
	return time.Parse(time.RFC3339, *spc.Timestamp)
}

func (spc *StatusPageComponentChange) GetGraphVals(timestamp time.Time) []any {
	return []any{timestamp, spcStatusToInt(*spc.Status)}
}

func (spc *StatusPageComponentChange) GetTableVals(timestamp time.Time) []any {
	return []any{timestamp, spcStatusToInt(*spc.Status), *spc.Component, *spc.MonitorLogicalName}
}

func (spc *StatusPageComponentChange) GetKey() string {
	return fmt.Sprintf("%s-%s", *spc.Component, *spc.MonitorLogicalName)
}

func (spc *StatusPageComponentChange) GetGraphFrameDefinition() data.Frame {
	return data.Frame{
		Fields: []*data.Field{
			data.NewField("time", nil, make([]time.Time, 0)),
			data.NewField("status", spc.getLabels(), make([]int8, 0)),
		},
		Meta: &data.FrameMeta{
			Type:                   data.FrameTypeTimeSeriesMulti,
			PreferredVisualization: data.VisTypeGraph,
		},
	}
}

func (spc *StatusPageComponentChange) GetTableFrameDefinition() data.Frame {
	return data.Frame{
		Fields: []*data.Field{
			data.NewField("time", nil, []time.Time{}),
			data.NewField("status", nil, []int8{}),
			data.NewField("component", nil, []string{}),
			data.NewField("monitor", nil, []string{}),
		},
		Meta: &data.FrameMeta{
			Type:                   data.FrameTypeTimeSeriesWide,
			PreferredVisualization: data.VisTypeTable,
		},
	}
}

func (spc *StatusPageComponentChange) getLabels() map[string]string {
	return map[string]string{"component": *spc.Component, "monitor": *spc.MonitorLogicalName}
}

// Map statuses to numeric values for Frames
func spcStatusToInt(status string) int8 {
	statuses := map[string]int8{
		"up":                   1,
		"operational":          1,
		"Good":                 1,
		"Information":          1,
		"NotApplicable":        1,
		"Advisory":             1,
		"Healthy":              1,
		"available":            1,
		"information":          1,
		"Degraded":             2,
		"Warning":              2,
		"degraded":             2,
		"disruption":           2,
		"down":                 3,
		"Disruption":           3,
		"Critical":             3,
		"outage":               3,
		"degraded_performance": 3,
		"major_outage":         3,
		"partial_outage":       3,
		"under_maintenance":    4,
	}
	result := statuses[status]
	return result
}
