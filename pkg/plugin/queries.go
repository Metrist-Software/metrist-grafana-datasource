package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/metrist/metrist/pkg/internal"
)

const (
	DataFrameMonitorErrors            = "errors"
	DataFrameMonitorTelemetry         = "telemetry"
	DataFrameMonitorStatusPageChanges = "status_page_changes"
	DataFrameMonitorStatus            = "status"
	DataFrameMonitorList           = "monitor_list"
)

// QueryMonitorErrors queries `/monitor-telemetry`
func QueryMonitorErrors(ctx context.Context, query backend.DataQuery, client internal.ClientWithResponsesInterface, apiKey string) (backend.DataResponse, error) {
	from, to := query.TimeRange.From.Format(time.RFC3339), query.TimeRange.To.Format(time.RFC3339)
	var monitorTelemetryQuery monitorTelemetryQuery
	if err := json.Unmarshal(query.JSON, &monitorTelemetryQuery); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "json unmarshal: "+err.Error()), err
	}

	resp, err := client.BackendWebMonitorErrorControllerGetWithResponse(ctx,
		&internal.BackendWebMonitorErrorControllerGetParams{
			From:          from,
			To:            &to,
			M:             &monitorTelemetryQuery.Monitors,
			IncludeShared: &monitorTelemetryQuery.IncludeShared,
		},
		withAPIKey(apiKey))

	if err != nil {
		return backend.DataResponse{}, err
	}

	if len(*resp.JSON200) == 0 {
		return backend.DataResponse{}, nil
	}

	responses := *resp.JSON200
	frame := &data.Frame{
		Name: DataFrameMonitorErrors,
		Fields: []*data.Field{
			data.NewField("time", nil, []time.Time{}),
			data.NewField("", nil, []int8{}),
			data.NewField("instance", nil, []string{}),
			data.NewField("check", nil, []string{}),
			data.NewField("monitor", nil, []string{}),
		},
	}

	var value int8 = 1
	for _, monitorError := range responses {
		timestamp, err := time.Parse(time.RFC3339, *monitorError.Timestamp)
		if err != nil {
			log.DefaultLogger.Error("error while parsing monitor error time %w", err)
			continue
		}
		frame.AppendRow(timestamp, value, *monitorError.Instance, *monitorError.Check, *monitorError.MonitorLogicalName)
	}

	f, err := data.LongToWide(frame, nil)
	if err != nil {
		return backend.DataResponse{}, err
	}

	return backend.DataResponse{Frames: []*data.Frame{f}}, nil
}

// QueryMonitorTelemetry queries `/monitor-telemetry`
func QueryMonitorTelemetry(ctx context.Context, query backend.DataQuery, client internal.ClientWithResponsesInterface, apiKey string) (backend.DataResponse, error) {
	from, to := query.TimeRange.From.Format(time.RFC3339), query.TimeRange.To.Format(time.RFC3339)
	var monitorTelemetryQuery monitorTelemetryQuery

	if err := json.Unmarshal(query.JSON, &monitorTelemetryQuery); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "json unmarshal: "+err.Error()), err
	}

	resp, err := client.BackendWebMonitorTelemetryControllerGetWithResponse(ctx,
		&internal.BackendWebMonitorTelemetryControllerGetParams{
			From:          from,
			To:            &to,
			M:             &monitorTelemetryQuery.Monitors,
			IncludeShared: &monitorTelemetryQuery.IncludeShared,
		},
		withAPIKey(apiKey))

	if err != nil {
		return backend.DataResponse{}, err
	}

	if len(*resp.JSON200) == 0 {
		return backend.DataResponse{}, nil
	}

	responses := *resp.JSON200
	frame := &data.Frame{
		Name: DataFrameMonitorTelemetry,
		Fields: []*data.Field{
			data.NewField("time", nil, []time.Time{}),
			data.NewField("", nil, []float32{}),
			data.NewField("instance", nil, []string{}),
			data.NewField("check", nil, []string{}),
			data.NewField("monitor", nil, []string{}),
		},
	}

	for _, te := range responses {
		timestamp, err := time.Parse(time.RFC3339, *te.Timestamp)
		if err != nil {
			log.DefaultLogger.Error("error while parsing telemetry time %w", err)
			continue
		}
		frame.AppendRow(timestamp, *te.Value, *te.Instance, *te.Check, *te.MonitorLogicalName)
	}

	f, err := data.LongToWide(frame, nil)
	if err != nil {
		return backend.DataResponse{}, err
	}
	return backend.DataResponse{Frames: []*data.Frame{f}}, nil

}

// QueryMonitorStatusPageChanges queries `/status-page-changes`
func QueryMonitorStatusPageChanges(ctx context.Context, query backend.DataQuery, client internal.ClientWithResponsesInterface, apiKey string) (backend.DataResponse, error) {
	from, to := query.TimeRange.From.Format(time.RFC3339), query.TimeRange.To.Format(time.RFC3339)
	var monitorTelemetryQuery monitorTelemetryQuery

	if err := json.Unmarshal(query.JSON, &monitorTelemetryQuery); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "json unmarshal: "+err.Error()), err
	}

	resp, err := client.BackendWebStatusPageChangeControllerGetWithResponse(ctx,
		&internal.BackendWebStatusPageChangeControllerGetParams{
			From: from,
			To:   &to,
			M:    &monitorTelemetryQuery.Monitors,
		}, withAPIKey(apiKey))

	if err != nil {
		return backend.DataResponse{}, err
	}

	if len(*resp.JSON200) == 0 {
		return backend.DataResponse{}, nil
	}

	responses := *resp.JSON200
	frame := &data.Frame{
		Name: DataFrameMonitorStatusPageChanges,
		Fields: []*data.Field{
			data.NewField("time", nil, []time.Time{}),
			data.NewField("", nil, []int8{}),
			data.NewField("component", nil, []string{}),
			data.NewField("monitor", nil, []string{}),
		},
	}

	for _, te := range responses {
		timestamp, err := time.Parse(time.RFC3339, *te.Timestamp)
		if err != nil {
			log.DefaultLogger.Error("error while parsing status page changes time %w", err)
			continue
		}
		frame.AppendRow(timestamp, spcStatusToFloat(*te.Status), *te.Component, *te.MonitorLogicalName)
	}

	longFrame, err := data.LongToWide(frame, nil)

	if err != nil {
		return backend.DataResponse{}, err
	}

	for idx, field := range longFrame.Fields {
		if idx == 0 {
			continue
		}
		field.SetConfig(&data.FieldConfig{
			Mappings: data.ValueMappings{
				data.ValueMapper{"0": data.ValueMappingResult{Text: "(0) up", Color: "green"}},
				data.ValueMapper{"1": data.ValueMappingResult{Text: "(1) degraded", Color: "yellow"}},
				data.ValueMapper{"2": data.ValueMappingResult{Text: "(2) error", Color: "red"}},
			},
		})
	}

	return backend.DataResponse{Frames: []*data.Frame{longFrame}}, nil
}

//Query Monitor List

func QueryMonitorList(ctx context.Context, query backend.DataQuery, client internal.ClientWithResponsesInterface, apiKey string) (backend.DataResponse, error) {

	resp, err := client.BackendWebMonitorListControllerGetWithResponse(ctx,
		withAPIKey(apiKey))

	if err != nil {
		return backend.DataResponse{}, err
	}

	if len(*resp.JSON200) == 0 {
		return backend.DataResponse{}, nil
	}

	responses := *resp.JSON200
	print(responses)

	frame := &data.Frame{
		Name: DataFrameMonitorList,
		Fields: []*data.Field{
			data.NewField("monitor", nil, []string{}),
		},
	}

	longFrame, err := data.LongToWide(frame, nil)



	return backend.DataResponse{Frames: []*data.Frame{longFrame}}, nil
}

func spcStatusToFloat(status string) int8 {
	statuses := map[string]int8{
		"up":          0,
		"operational": 0,
		"degraded":    1,
		"down":        2,
	}
	result := statuses[status]
	return result
}

func withAPIKey(apiKey string) internal.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Add("Authorization", apiKey)
		return nil
	}
}
