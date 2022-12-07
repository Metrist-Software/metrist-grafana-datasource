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
)

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
			data.NewField("errorString", nil, []string{}),
			data.NewField("instance", nil, []string{}),
			data.NewField("check", nil, []string{}),
			data.NewField("monitor", nil, []string{}),
		},
	}

	for _, monitorError := range responses {
		timestamp, err := time.Parse(time.RFC3339, *monitorError.Timestamp)
		if err != nil {
			log.DefaultLogger.Error("error while parsing monitor error time %w", err)
			continue
		}
		frame.AppendRow(timestamp, *monitorError.ErrorString, *monitorError.Instance, *monitorError.Check, *monitorError.MonitorLogicalName)
	}

	return backend.DataResponse{Frames: []*data.Frame{frame}}, nil
}

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

	f, _ := data.LongToWide(frame, nil)
	return backend.DataResponse{Frames: []*data.Frame{f}}, nil

}

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
			data.NewField("status", nil, []string{}),
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
		frame.AppendRow(timestamp, *te.Status, *te.Component, *te.MonitorLogicalName)
	}

	return backend.DataResponse{Frames: []*data.Frame{frame}}, nil
}

func QueryMonitorStatus(ctx context.Context, query backend.DataQuery, client internal.ClientWithResponsesInterface, apiKey string) (backend.DataResponse, error) {
	var monitorTelemetryQuery monitorTelemetryQuery

	if err := json.Unmarshal(query.JSON, &monitorTelemetryQuery); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "json unmarshal: "+err.Error()), err
	}

	resp, err := client.BackendWebMonitorStatusControllerGetWithResponse(ctx,
		&internal.BackendWebMonitorStatusControllerGetParams{
			M: monitorTelemetryQuery.Monitors,
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
			data.NewField("state", nil, []string{}),
			data.NewField("monitor", nil, []string{}),
		},
	}

	for _, te := range responses {
		timestamp, err := time.Parse(time.RFC3339, *te.LastChecked)
		if err != nil {
			log.DefaultLogger.Error("error while parsing status page changes time %w", err)
			continue
		}
		frame.AppendRow(timestamp, *te.State, *te.MonitorLogicalName)
	}

	return backend.DataResponse{Frames: []*data.Frame{frame}}, nil
}

func withAPIKey(apiKey string) internal.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Add("Authorization", apiKey)
		return nil
	}
}
