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

type frameType int64

const (
	GraphFrameType frameType = 0
	TableFrameType frameType = 1
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

func buildFrames(responses []internal.FrameData, frameType frameType, frames []*data.Frame) []*data.Frame {
	frameMap := make(map[string]*data.Frame)

	var frameToAppendTo *data.Frame
	for _, frameDataItem := range responses {
		timestamp, err := frameDataItem.GetTimestamp()
		if err != nil {
			log.DefaultLogger.Error("error while parsing time %w", err)
			continue
		}
		frameDefinition := getFrameDefinitionFunction(frameType, frameDataItem)()
		// For table Wide frames, we always want to append to the one single frame in order
		if frameType == TableFrameType && frameToAppendTo == nil {
			frameToAppendTo = &frameDefinition
			frameMap["fixed-table"] = frameToAppendTo
		} else if frameType == GraphFrameType {
			key := frameDataItem.GetKey()

			var ok bool
			frameToAppendTo, ok = frameMap[key]
			if !ok {
				frameToAppendTo = &frameDefinition
				frameMap[key] = frameToAppendTo
			}
		}

		vals := getValDefinitionFunction(frameType, frameDataItem)(timestamp)
		frameToAppendTo.AppendRow(vals...)
	}
	for _, frame := range frameMap {
		frames = append(frames, frame)
	}

	return frames
}

func getFrameDefinitionFunction(frameType frameType, frameData internal.FrameData) func() data.Frame {
	switch frameType {
	case GraphFrameType:
		return frameData.GetGraphFrameDefinition
	case TableFrameType:
		return frameData.GetTableFrameDefinition
	}

	return nil
}

func getValDefinitionFunction(frameType frameType, frameData internal.FrameData) func(time.Time) []any {
	switch frameType {
	case GraphFrameType:
		return frameData.GetGraphVals
	case TableFrameType:
		return frameData.GetTableVals
	}

	return nil
}

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

	// Have to coerce these into actual internal.FrameData as you can't pass responses to []any
	coercedCounts := make([]internal.FrameData, len(responses))
	for i := range responses {
		coercedCounts[i] = &responses[i]
	}

	frames := make([]*data.Frame, 0)
	frames = buildFrames(coercedCounts, GraphFrameType, frames)
	if !monitorTelemetryQuery.FromAlerting {
		frames = buildFrames(coercedCounts, TableFrameType, frames)
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

	// Have to coerce these into actual internal.FrameData as you can't pass responses to []any
	coercedTelemetry := make([]internal.FrameData, len(responses))
	for i := range responses {
		coercedTelemetry[i] = &responses[i]
	}

	frames := make([]*data.Frame, 0)
	frames = buildFrames(coercedTelemetry, GraphFrameType, frames)
	if !monitorTelemetryQuery.FromAlerting {
		frames = buildFrames(coercedTelemetry, TableFrameType, frames)
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

	// Have to coerce these into actual internal.FrameData as you can't pass responses to []any
	coercedStatusPageChanges := make([]internal.FrameData, len(responses))
	for i := range responses {
		coercedStatusPageChanges[i] = &responses[i]
	}

	frames := make([]*data.Frame, 0)
	frames = buildFrames(coercedStatusPageChanges, GraphFrameType, frames)
	if !monitorTelemetryQuery.FromAlerting {
		frames = buildFrames(coercedStatusPageChanges, TableFrameType, frames)
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
					data.ValueMapper{"3": data.ValueMappingResult{Text: "(3) maintenance", Color: "blue"}},
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
