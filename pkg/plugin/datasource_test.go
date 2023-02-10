package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal"
	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
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
					JSON200: &internal.MonitorTelemetryResponse{internal.MonitorTelemetry{
						Check:              ptr("Check"),
						Instance:           ptr("us-east-1"),
						MonitorLogicalName: ptr("awslambda"),
						Timestamp:          ptr("2022-12-07T18:28:06.485416Z"),
						Value:              &value,
					}},
				},
			},
			want: data.Frames{{
				Fields: []*data.Field{
					data.NewField("time", nil, []time.Time{strToTime("2022-12-07T18:28:06.485416Z")}),
					data.NewField("response time (ms)", data.Labels{"instance": "us-east-1", "check": "Check", "monitor": "awslambda"}, []float32{value}),
				},
				Meta: &data.FrameMeta{Type: data.FrameTypeTimeSeriesMulti, PreferredVisualization: data.VisTypeGraph},
			},
				{
					Fields: []*data.Field{
						data.NewField("time", nil, []time.Time{strToTime("2022-12-07T18:28:06.485416Z")}),
						data.NewField("response time (ms)", nil, []float32{100}),
						data.NewField("instance", nil, []string{"us-east-1"}),
						data.NewField("check", nil, []string{"Check"}),
						data.NewField("monitor", nil, []string{"awslambda"}),
					},
					Meta: &data.FrameMeta{Type: data.FrameTypeTimeSeriesWide, PreferredVisualization: data.VisTypeTable},
				},
			},
		},
		{
			name: "Returns an empty frame if no response",
			client: stubClient{
				telemetryResponse: internal.BackendWebMonitorTelemetryControllerGetResponse{
					JSON200: &internal.MonitorTelemetryResponse{},
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
		page *internal.StatusPageChangesResponse
		name string
		want data.Frames
	}{
		{
			name: "Returns a dataframe if client returns telemetry",
			page: &internal.StatusPageChangesResponse{
				Metadata: &internal.PagingMetadata{},
				Entries: &[]internal.StatusPageComponentChange{{
					Component:          ptr("component1"),
					MonitorLogicalName: ptr("monitor"),
					Status:             ptr("up"),
					Timestamp:          ptr("2022-12-07T18:28:06.485416Z"),
				}},
			},
			want: data.Frames{{
				Fields: []*data.Field{
					data.NewField("time", nil, []time.Time{strToTime("2022-12-07T18:28:06.485416Z")}),
					data.NewField("status", data.Labels{"component": "component1", "monitor": "monitor"}, []int8{0}),
				},
				Meta: &data.FrameMeta{Type: data.FrameTypeTimeSeriesMulti, PreferredVisualization: data.VisTypeGraph},
			},
				{
					Fields: []*data.Field{
						data.NewField("time", nil, []time.Time{strToTime("2022-12-07T18:28:06.485416Z")}),
						data.NewField("status", nil, []int8{0}),
						data.NewField("component", nil, []string{"component1"}),
						data.NewField("monitor", nil, []string{"monitor"}),
					},
					Meta: &data.FrameMeta{Type: data.FrameTypeTimeSeriesWide, PreferredVisualization: data.VisTypeTable},
				},
			},
		},
		{
			name: "Returns an empty frame if no response",
			page: &internal.StatusPageChangesResponse{
				Metadata: &internal.PagingMetadata{},
				Entries:  &[]internal.StatusPageComponentChange{},
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
	query := []byte(`{"monitors": ["awslambda"], "includeShared": false, "queryType": "GetMonitorErrors"}`)
	tests := []struct {
		page *internal.MonitorErrorResponse
		name string
		want data.Frames
	}{
		{
			name: "Returns a dataframe if client returns telemetry",
			page: &internal.MonitorErrorResponse{
				Entries: &[]internal.MonitorErrorCount{{
					Check:              ptr("check"),
					Count:              ptr(1),
					Instance:           ptr("us-east-1"),
					MonitorLogicalName: ptr("monitor"),
					Timestamp:          ptr("2022-12-07T18:28:06.485416Z"),
				}},
				Metadata: &internal.PagingMetadata{},
			},
			want: data.Frames{{
				Fields: []*data.Field{
					data.NewField("time", nil, []time.Time{strToTime("2022-12-07T18:28:06.485416Z")}),
					data.NewField("count", data.Labels{"check": "check", "monitor": "monitor", "instance": "us-east-1"}, []int64{1}),
				},
				Meta: &data.FrameMeta{Type: data.FrameTypeTimeSeriesMulti, PreferredVisualization: data.VisTypeGraph},
			},
				{
					Fields: []*data.Field{
						data.NewField("time", nil, []time.Time{strToTime("2022-12-07T18:28:06.485416Z")}),
						data.NewField("count", nil, []int64{1}),
						data.NewField("instance", nil, []string{"us-east-1"}),
						data.NewField("check", nil, []string{"check"}),
						data.NewField("monitor", nil, []string{"monitor"}),
					},
					Meta: &data.FrameMeta{Type: data.FrameTypeTimeSeriesWide, PreferredVisualization: data.VisTypeTable},
				},
			},
		},
		{
			name: "Returns an empty frame if no response",
			page: &internal.MonitorErrorResponse{
				Entries:  &[]internal.MonitorErrorCount{},
				Metadata: &internal.PagingMetadata{}},
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
