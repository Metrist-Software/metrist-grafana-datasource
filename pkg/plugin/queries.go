package plugin

import (
	"context"
	"encoding/json"
	"fmt"
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

	// We are going to generate 2 frame sets. one for graph display and one for table display
	graphFrameMap := make(map[string]*data.Frame)
	tableFrameMap := make(map[string]*data.Frame)
	frames := make([]*data.Frame, 0)

	for _, monitorError := range responses {
		timestamp, err := time.Parse(time.RFC3339, *monitorError.Timestamp)
		if err != nil {
			log.DefaultLogger.Error("error while parsing monitor error time %w", err)
			continue
		}

		key := fmt.Sprintf("%s-%s-%s", *monitorError.Instance, *monitorError.Check, *monitorError.MonitorLogicalName)

		frameToAppendTo, ok := graphFrameMap[key]
		if !ok {
			labels := map[string]string{"instance": *monitorError.Instance, "check": *monitorError.Check, "monitor": *monitorError.MonitorLogicalName}

			frameToAppendTo = &data.Frame{
				Fields: []*data.Field{
					data.NewField("time", nil, make([]time.Time, 0)),
					data.NewField("count", labels, make([]int64, 0)),
				},
				Meta: &data.FrameMeta{
					Type: data.FrameTypeTimeSeriesMulti,
				},
			}

			graphFrameMap[key] = frameToAppendTo
		}

		frameToAppendTo.AppendRow(timestamp, int64(*monitorError.Count))

		frameToAppendTo, ok = tableFrameMap[key]
		if !ok {
			frameToAppendTo = &data.Frame{
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

			tableFrameMap[key] = frameToAppendTo
		}
		frameToAppendTo.AppendRow(timestamp, int64(*monitorError.Count), *monitorError.Instance, *monitorError.Check, *monitorError.MonitorLogicalName)
	}

	for _, frame := range graphFrameMap {
		frames = append(frames, frame)
	}

	// If this query is coming from CloudAlerting or Unified alerting do not include the table frames
	// The table frames are not FrameTypeTimeSeriesWide format which alerting won't accept
	// See https://github.com/grafana/grafana-plugin-sdk-go/blob/main/data/contract_docs/timeseries.md#time-series-multi-format-timeseriesmulti
	if !monitorTelemetryQuery.FromAlerting {
		for _, frame := range tableFrameMap {
			frames = append(frames, frame)
		}
	}

	return backend.DataResponse{Frames: frames}, nil
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

	// We are going to generate 2 frame sets. one for graph display and one for table display
	graphFrameMap := make(map[string]*data.Frame)
	tableFrameMap := make(map[string]*data.Frame)
	frames := make([]*data.Frame, 0)

	for _, te := range responses {
		timestamp, err := time.Parse(time.RFC3339, *te.Timestamp)
		if err != nil {
			log.DefaultLogger.Error("error while parsing telemetry time %w", err)
			continue
		}

		key := fmt.Sprintf("%s-%s-%s", *te.Instance, *te.Check, *te.MonitorLogicalName)

		frameToAppendTo, ok := graphFrameMap[key]
		if !ok {
			labels := map[string]string{"instance": *te.Instance, "check": *te.Check, "monitor": *te.MonitorLogicalName}

			frameToAppendTo = &data.Frame{
				Fields: []*data.Field{
					data.NewField("time", nil, make([]time.Time, 0)),
					data.NewField("response time (ms)", labels, make([]float32, 0)),
				},
				Meta: &data.FrameMeta{
					Type: data.FrameTypeTimeSeriesMulti,
				},
			}

			graphFrameMap[key] = frameToAppendTo
		}

		frameToAppendTo.AppendRow(timestamp, *te.Value)

		frameToAppendTo, ok = tableFrameMap[key]
		if !ok {
			frameToAppendTo = &data.Frame{
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

			tableFrameMap[key] = frameToAppendTo
		}
		frameToAppendTo.AppendRow(timestamp, *te.Value, *te.Instance, *te.Check, *te.MonitorLogicalName)
	}

	for _, frame := range graphFrameMap {
		frames = append(frames, frame)
	}

	// If this query is coming from CloudAlerting or Unified alerting do not include the table frames
	// The table frames are not FrameTypeTimeSeriesWide format which alerting won't accept
	// See https://github.com/grafana/grafana-plugin-sdk-go/blob/main/data/contract_docs/timeseries.md#time-series-multi-format-timeseriesmulti
	if !monitorTelemetryQuery.FromAlerting {
		for _, frame := range tableFrameMap {
			frames = append(frames, frame)
		}
	}

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

	// We are going to generate 2 frame sets. one for graph display and one for table display
	graphFrameMap := make(map[string]*data.Frame)
	tableFrameMap := make(map[string]*data.Frame)
	frames := make([]*data.Frame, 0)

	for _, te := range responses {
		timestamp, err := time.Parse(time.RFC3339, *te.Timestamp)
		if err != nil {
			log.DefaultLogger.Error("error while parsing status page changes time %w", err)
			continue
		}

		key := fmt.Sprintf("%s-%s", *te.Component, *te.MonitorLogicalName)

		frameToAppendTo, ok := graphFrameMap[key]
		if !ok {
			labels := map[string]string{"component": *te.Component, "monitor": *te.MonitorLogicalName}

			frameToAppendTo = &data.Frame{
				Fields: []*data.Field{
					data.NewField("time", nil, make([]time.Time, 0)),
					data.NewField("status", labels, make([]int8, 0)),
				},
				Meta: &data.FrameMeta{
					Type: data.FrameTypeTimeSeriesMulti,
				},
			}

			graphFrameMap[key] = frameToAppendTo
		}

		frameToAppendTo.AppendRow(timestamp, spcStatusToInt(*te.Status))

		frameToAppendTo, ok = tableFrameMap[key]
		if !ok {
			frameToAppendTo = &data.Frame{
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

			tableFrameMap[key] = frameToAppendTo
		}
		frameToAppendTo.AppendRow(timestamp, spcStatusToInt(*te.Status), *te.Component, *te.MonitorLogicalName)
	}

	for _, frame := range graphFrameMap {
		frames = append(frames, frame)
	}

	// If this query is coming from CloudAlerting or Unified alerting do not include the table frames
	// The table frames are not FrameTypeTimeSeriesWide format which alerting won't accept
	// See https://github.com/grafana/grafana-plugin-sdk-go/blob/main/data/contract_docs/timeseries.md#time-series-multi-format-timeseriesmulti
	if !monitorTelemetryQuery.FromAlerting {
		for _, frame := range tableFrameMap {
			frames = append(frames, frame)
		}
	}

	if err != nil {
		return backend.DataResponse{}, err
	}

	for _, frame := range frames {
		for idx, field := range frame.Fields {
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
	}

	return backend.DataResponse{Frames: frames}, nil
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

func spcStatusToInt(status string) int8 {
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
