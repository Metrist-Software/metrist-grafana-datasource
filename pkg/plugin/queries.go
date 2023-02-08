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

func fetchAllMonitorErrors(ctx context.Context, client internal.ClientWithResponsesInterface, query monitorTelemetryQuery, tr backend.TimeRange) ([]internal.MonitorErrorCount, error) {
	onlyShared := true

	params := []internal.BackendWebMonitorErrorControllerGetParams{{
		From: tr.From,
		To:   tr.To,
		M:    query.Monitors,
		C:    query.Checks,
		I:    query.Instances,
	}}

	if query.IncludeShared {
		params = append(params, internal.BackendWebMonitorErrorControllerGetParams{
			From:       tr.From,
			To:         tr.To,
			M:          query.Monitors,
			OnlyShared: &onlyShared,
			C:          query.Checks,
			I:          query.Instances,
		})
	}

	g, ctx := errgroup.WithContext(ctx)
	result := make([][]internal.MonitorErrorCount, len(params))
	// Runs 2 go routines if shared is included
	// Each goroutine will page through the result
	for i, param := range params {
		param := param // https://golang.org/doc/faq#closures_and_goroutines
		i := i
		g.Go(func() error {
			currentParam := internal.BackendWebMonitorErrorControllerGetParams{
				From:       param.From,
				To:         param.To,
				M:          param.M,
				OnlyShared: param.OnlyShared,
				C:          nilIfEmpty(param.C),
				I:          nilIfEmpty(param.I),
			}

			for pageCount := 0; pageCount < maxPageCount; pageCount++ {
				resp, err := client.BackendWebMonitorErrorControllerGetWithResponse(ctx, &currentParam)
				if err != nil {
					return err
				}

				response := resp.JSON200
				if response == nil {
					log.DefaultLogger.Warn("non 200 status code encountered. status %v, body %v", resp.HTTPResponse.Status, resp.Body)
					return nil
				}

				result[i] = append(result[i], *response.Entries...)
				if currentParam.CursorAfter = response.Metadata.CursorAfter; currentParam.CursorAfter == nil {
					break
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	monitorErrors := make([]internal.MonitorErrorCount, 0)
	for _, v := range result {
		if len(v) == 0 {
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
	if err := ensureTelemetryRequestWithinLast90Days(query.TimeRange.From); err != nil {
		log.DefaultLogger.Error("telemetry requested for greater than 90 days error: %w", err)
		return backend.ErrDataResponse(backend.StatusBadRequest, err.Error()), err
	}

	var monitorTelemetryQuery monitorTelemetryQuery

	if err := json.Unmarshal(query.JSON, &monitorTelemetryQuery); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "json unmarshal: "+err.Error()), err
	}

	params := internal.BackendWebMonitorTelemetryControllerGetParams{
		From:          query.TimeRange.From,
		To:            query.TimeRange.To,
		M:             monitorTelemetryQuery.Monitors,
		IncludeShared: &monitorTelemetryQuery.IncludeShared,
		C:             nilIfEmpty(monitorTelemetryQuery.Checks),
		I:             nilIfEmpty(monitorTelemetryQuery.Instances),
	}

	resp, err := client.BackendWebMonitorTelemetryControllerGetWithResponse(ctx, &params)

	if err != nil {
		return backend.DataResponse{}, err
	}

	if len(*resp.JSON200) == 0 {
		return backend.DataResponse{}, nil
	}

	responses := *resp.JSON200

	type Key struct {
		instance, check, monitor string
	}

	type Record struct {
		time  time.Time
		value float32
	}

	m := map[Key][]Record{}
	for _, r := range responses {
		timestamp, err := time.Parse(time.RFC3339, *r.Timestamp)
		if err != nil {
			log.DefaultLogger.Error("error while parsing telemetry time %w", err)
			continue
		}
		key := Key{*r.Instance, *r.Check, *r.MonitorLogicalName}
		m[key] = append(m[key], Record{timestamp, *r.Value})
	}

	var frames = make([]*data.Frame, 0)
	for key, values := range m {
		graphFrame := data.NewFrame("",
			data.NewField("Time", nil, make([]time.Time, len(values))),
			data.NewField("Value",
				map[string]string{"check": key.check, "instance": key.instance, "monitor": key.monitor},
				make([]float32, len(values)),
			),
		)
		for pIdx, record := range values {
			graphFrame.Set(0, pIdx, record.time)
			graphFrame.Set(1, pIdx, record.value)
		}
		graphFrame.Meta = &data.FrameMeta{
			PreferredVisualization: data.VisTypeGraph,
		}
		frames = append(frames, graphFrame)
	}

	tableFrame := &data.Frame{
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
		tableFrame.AppendRow(timestamp, *te.Value, *te.Instance, *te.Check, *te.MonitorLogicalName)
	}
	tableFrame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeTable,
	}

	frames = append(frames, tableFrame)

	return backend.DataResponse{Frames: frames}, nil

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

func fetchAllStatusPageMonitor(ctx context.Context, client internal.ClientWithResponsesInterface, query monitorTelemetryQuery, tr backend.TimeRange) ([]internal.StatusPageComponentChange, error) {
	monitorStatuses := make([]internal.StatusPageComponentChange, 0)
	params := internal.BackendWebStatusPageChangeControllerGetParams{
		From: tr.From,
		To:   &tr.To,
		M:    query.Monitors,
	}
	for pageCount := 0; pageCount < maxPageCount; pageCount++ {
		resp, err := client.BackendWebStatusPageChangeControllerGetWithResponse(ctx, &params)
		if err != nil {
			return nil, err
		}

		response := resp.JSON200
		monitorStatuses = append(monitorStatuses, *response.Entries...)

		if params.CursorAfter = response.Metadata.CursorAfter; params.CursorAfter == nil {
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

func nilIfEmpty(slice *[]string) *[]string {
	if slice == nil || len(*slice) == 0 {
		return nil
	} else {
		return slice
	}
}

func ensureTelemetryRequestWithinLast90Days(fromDate time.Time) error {
	currentTime := time.Now().In(fromDate.Location())
	threeMonthsAgo := currentTime.Add(-durationThreeMonths)

	if time.Time.Before(fromDate, threeMonthsAgo) {
		return errTelemetryRequestedOutsideBounds
	}

	return nil
}
