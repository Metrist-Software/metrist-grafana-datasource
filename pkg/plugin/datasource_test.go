package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/metrist/metrist/pkg/internal"
)

func TestQueryMonitorTelemetry(t *testing.T) {
	var value float32 = 100
	timeRange := backend.TimeRange{
		To:   time.Now(),
		From: time.Now().Add(time.Hour * time.Duration(-100)),
	}
	query := []byte(`{"monitors": ["awslambda"], "includeShared": true, "queryType": "GetMonitorTelemetry"}`)
	tests := []struct {
		client stubClient
		name   string
		want   data.Frames
	}{
		{
			name: "Returns a dataframe if client returns telemetry",
			client: stubClient{
				telemetryResponse: internal.BackendWebMonitorTelemetryControllerGetResponse{
					JSON200: &internal.MonitorTelemetry{{
						Check:              ptr("Check"),
						Instance:           ptr("us-east-1"),
						MonitorLogicalName: ptr("awslambda"),
						Timestamp:          ptr("2022-12-07T18:28:06.485416Z"),
						Value:              &value,
					}},
				},
			},
			want: data.Frames{{
				Name: DataFrameMonitorTelemetry,
				Fields: []*data.Field{
					data.NewField("time", nil, []time.Time{strToTime("2022-12-07T18:28:06.485416Z")}),
					data.NewField("", data.Labels{"instance": "us-east-1", "check": "Check", "monitor": "awslambda"}, []float32{value}),
				},
				Meta: &data.FrameMeta{Type: data.FrameTypeTimeSeriesWide},
			}},
		},
		{
			name: "Returns an empty frame if no response",
			client: stubClient{
				telemetryResponse: internal.BackendWebMonitorTelemetryControllerGetResponse{
					JSON200: &internal.MonitorTelemetry{},
				},
			},
			want: data.Frames{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ds := Datasource{openApiClient: &test.client}
			resp, err := ds.QueryData(
				context.Background(),
				&backend.QueryDataRequest{
					PluginContext: testPluginContext,
					Queries:       []backend.DataQuery{{RefID: "A", JSON: query, TimeRange: timeRange}},
				},
			)
			if err != nil {
				t.Error(err)
			}
			if len(resp.Responses) != 1 {
				t.Fatal("QueryData must return a response")
			}
			if diff := cmp.Diff(test.want, resp.Responses["A"].Frames, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		})

	}
}

func TestQueryMonitorStatusPageChanges(t *testing.T) {
	timeRange := backend.TimeRange{
		To:   time.Now(),
		From: time.Now().Add(time.Hour * time.Duration(-100)),
	}
	query := []byte(`{"monitors": ["awslambda"], "includeShared": true, "queryType": "GetMonitorStatusPageChanges"}`)
	tests := []struct {
		page *internal.StatusPageChangesPage
		name string
		want data.Frames
	}{
		{
			name: "Returns a dataframe if client returns telemetry",
			page: &internal.StatusPageChangesPage{
				Metadata: &internal.PageMetadata{},
				Entries: &internal.StatusPageChanges{{
					Component:          ptr("component1"),
					MonitorLogicalName: ptr("monitor"),
					Status:             ptr("up"),
					Timestamp:          ptr("2022-12-07T18:28:06.485416Z"),
				}},
			},
			want: data.Frames{{
				Name: DataFrameMonitorStatusPageChanges,
				Fields: []*data.Field{
					data.NewField("time", nil, []time.Time{strToTime("2022-12-07T18:28:06.485416Z")}),
					data.NewField("", data.Labels{"component": "component1", "monitor": "monitor"}, []int8{0}),
				},
				Meta: &data.FrameMeta{Type: data.FrameTypeTimeSeriesWide},
			}},
		},
		{
			name: "Returns an empty frame if no response",
			page: &internal.StatusPageChangesPage{
				Metadata: &internal.PageMetadata{},
				Entries:  &internal.StatusPageChanges{},
			},
			want: data.Frames{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ds := Datasource{openApiClient: &stubClient{
				statusPageResponse: internal.BackendWebStatusPageChangeControllerGetResponse{
					JSON200: test.page,
				},
			}}
			resp, err := ds.QueryData(
				context.Background(),
				&backend.QueryDataRequest{
					PluginContext: testPluginContext,
					Queries:       []backend.DataQuery{{RefID: "A", JSON: query, TimeRange: timeRange}},
				},
			)
			if err != nil {
				t.Error(err)
			}
			if len(resp.Responses) != 1 {
				t.Fatal("QueryData must return a response")
			}
			// We dont care about the field config when testing
			for _, frame := range resp.Responses["A"].Frames {
				for _, field := range frame.Fields {
					field.Config = nil
				}
			}
			if diff := cmp.Diff(test.want, resp.Responses["A"].Frames, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		})

	}
}

func TestQueryMonitorErrors(t *testing.T) {
	timeRange := backend.TimeRange{
		To:   time.Now(),
		From: time.Now().Add(time.Hour * time.Duration(-100)),
	}
	query := []byte(`{"monitors": ["awslambda"], "includeShared": true, "queryType": "GetMonitorErrors"}`)
	tests := []struct {
		page *internal.MonitorErrorsPage
		name string
		want data.Frames
	}{
		{
			name: "Returns a dataframe if client returns telemetry",
			page: &internal.MonitorErrorsPage{
				Entries: &internal.MonitorErrorCounts{{
					Check:              ptr("check"),
					Count:              ptr(1),
					Instance:           ptr("us-east-1"),
					MonitorLogicalName: ptr("monitor"),
					Timestamp:          ptr("2022-12-07T18:28:06.485416Z"),
				}},
				Metadata: &internal.PageMetadata{},
			},
			want: data.Frames{{
				Name: DataFrameMonitorErrors,
				Fields: []*data.Field{
					data.NewField("time", nil, []time.Time{strToTime("2022-12-07T18:28:06.485416Z")}),
					data.NewField("", data.Labels{"check": "check", "monitor": "monitor", "instance": "us-east-1"}, []int64{1}),
				},
				Meta: &data.FrameMeta{Type: data.FrameTypeTimeSeriesWide},
			}},
		},
		{
			name: "Returns an empty frame if no response",
			page: &internal.MonitorErrorsPage{
				Entries:  &internal.MonitorErrorCounts{},
				Metadata: &internal.PageMetadata{}},
			want: data.Frames{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ds := Datasource{openApiClient: &stubClient{errorResponse: internal.BackendWebMonitorErrorControllerGetResponse{
				JSON200: test.page,
			}}}
			resp, err := ds.QueryData(
				context.Background(),
				&backend.QueryDataRequest{
					PluginContext: testPluginContext,
					Queries:       []backend.DataQuery{{RefID: "A", JSON: query, TimeRange: timeRange}},
				},
			)
			if err != nil {
				t.Error(err)
			}
			if len(resp.Responses) != 1 {
				t.Fatal("QueryData must return a response")
			}
			if diff := cmp.Diff(test.want, resp.Responses["A"].Frames, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		})

	}
}

func ptr[T any](v T) *T {
	return &v
}
