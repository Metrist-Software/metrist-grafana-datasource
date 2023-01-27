package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"golang.org/x/sync/errgroup"
)

const (
	DataFrameMonitorErrors            = "errors"
	DataFrameMonitorTelemetry         = "telemetry"
	DataFrameMonitorStatusPageChanges = "status_page_changes"
	DataFrameMonitorStatus            = "status"
)

const (
	maxPageCount = 20
)

// QueryMonitorErrors queries `/monitor-telemetry`
func QueryMonitorErrors(ctx context.Context, query backend.DataQuery, client internal.ClientWithResponsesInterface) (backend.DataResponse, error) {
	var monitorTelemetryQuery monitorTelemetryQuery
	if err := json.Unmarshal(query.JSON, &monitorTelemetryQuery); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "json unmarshal: "+err.Error()), err
	}

	responses, err := fetchAllMonitorErrors(ctx, client, monitorTelemetryQuery, query.TimeRange)
	if err != nil {
		return backend.DataResponse{}, err
	}

	if len(responses) == 0 {
		return backend.DataResponse{}, nil
	}

	frame := &data.Frame{
		Name: DataFrameMonitorErrors,
		Fields: []*data.Field{
			data.NewField("time", nil, []time.Time{}),
			data.NewField("", nil, []int64{}),
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
		frame.AppendRow(timestamp, int64(*monitorError.Count), *monitorError.Instance, *monitorError.Check, *monitorError.MonitorLogicalName)
	}

	f, err := data.LongToWide(frame, nil)
	if err != nil {
		return backend.DataResponse{}, err
	}

	return backend.DataResponse{Frames: []*data.Frame{f}}, nil
}

func fetchAllMonitorErrors(ctx context.Context, client internal.ClientWithResponsesInterface, query monitorTelemetryQuery, tr backend.TimeRange) (internal.MonitorErrorCounts, error) {
	onlyShared := true
	from, to := tr.From.Format(time.RFC3339), tr.To.Format(time.RFC3339)

	params := []internal.BackendWebMonitorErrorControllerGetParams{{
		From: from,
		To:   &to,
		M:    &query.Monitors,
	}}

	if query.IncludeShared {
		params = append(params, internal.BackendWebMonitorErrorControllerGetParams{
			From:       from,
			To:         &to,
			M:          &query.Monitors,
			OnlyShared: &onlyShared,
		})
	}

	g, ctx := errgroup.WithContext(ctx)
	result := make([]internal.MonitorErrorCounts, maxPageCount)
	// Runs 2 go routines if shared is included
	// Each goroutine will page through the result
	for i, param := range params {
		param := param // https://golang.org/doc/faq#closures_and_goroutines
		i := i
		g.Go(func() error {
			var cursorAfter *string
			for pageCount := 0; pageCount < maxPageCount; pageCount++ {
				resp, err := client.BackendWebMonitorErrorControllerGetWithResponse(ctx, &param)
				if err != nil {
					return err
				}
				response := resp.JSON200
				result[i] = *response.Entries
				if cursorAfter = response.Metadata.CursorAfter; cursorAfter == nil {
					break
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	monitorErrors := make(internal.MonitorErrorCounts, 0)
	for _, v := range result {
		if v == nil {
			continue
		}
		monitorErrors = append(monitorErrors, v...)
	}
	sort.SliceStable(monitorErrors, func(i, j int) bool {
		return strToTime(*monitorErrors[i].Timestamp).Before(strToTime(*monitorErrors[j].Timestamp))
	})
	return monitorErrors, nil
}

// QueryMonitorTelemetry queries `/monitor-telemetry`
func QueryMonitorTelemetry(ctx context.Context, query backend.DataQuery, client internal.ClientWithResponsesInterface) (backend.DataResponse, error) {
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
		})

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
func QueryMonitorStatusPageChanges(ctx context.Context, query backend.DataQuery, client internal.ClientWithResponsesInterface) (backend.DataResponse, error) {
	var monitorTelemetryQuery monitorTelemetryQuery

	if err := json.Unmarshal(query.JSON, &monitorTelemetryQuery); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "json unmarshal: "+err.Error()), err
	}

	responses, err := fetchAllStatusPageMonitor(ctx, client, monitorTelemetryQuery, query.TimeRange)
	if err != nil {
		return backend.DataResponse{}, err
	}

	if len(responses) == 0 {
		return backend.DataResponse{}, nil
	}

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

func fetchAllStatusPageMonitor(ctx context.Context, client internal.ClientWithResponsesInterface, query monitorTelemetryQuery, tr backend.TimeRange) (internal.StatusPageChanges, error) {
	monitorStatuses := make(internal.StatusPageChanges, 0)
	var cursorAfter *string = nil
	from, to := tr.From.Format(time.RFC3339), tr.To.Format(time.RFC3339)
	for pageCount := 0; pageCount < maxPageCount; pageCount++ {
		resp, err := client.BackendWebStatusPageChangeControllerGetWithResponse(ctx,
			&internal.BackendWebStatusPageChangeControllerGetParams{
				From: from,
				To:   &to,
				M:    &query.Monitors,
			})

		if err != nil {
			return nil, err
		}

		response := resp.JSON200
		monitorStatuses = append(monitorStatuses, *response.Entries...)

		if cursorAfter = response.Metadata.CursorAfter; cursorAfter == nil {
			break
		}
	}
	return monitorStatuses, nil
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
