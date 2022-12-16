package plugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/metrist/metrist/pkg/internal"
)

var (
	testPluginContext = backend.PluginContext{
		DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{
			DecryptedSecureJSONData: map[string]string{
				"apiKey": "test",
			},
		},
	}
)

type stubClient struct {
	internal.ClientWithResponsesInterface
	err                 error
	telemetryResponse   internal.BackendWebMonitorTelemetryControllerGetResponse
	statusPageResponse  internal.BackendWebStatusPageChangeControllerGetResponse
	errorResponse       internal.BackendWebMonitorErrorControllerGetResponse
	monitorListResponse internal.BackendWebMonitorListControllerGetResponse
}

func (m *stubClient) BackendWebMonitorTelemetryControllerGetWithResponse(ctx context.Context,
	params *internal.BackendWebMonitorTelemetryControllerGetParams,
	reqEditors ...internal.RequestEditorFn) (*internal.BackendWebMonitorTelemetryControllerGetResponse, error) {
	return &m.telemetryResponse, m.err
}

func (m *stubClient) BackendWebStatusPageChangeControllerGetWithResponse(ctx context.Context,
	params *internal.BackendWebStatusPageChangeControllerGetParams,
	reqEditors ...internal.RequestEditorFn) (*internal.BackendWebStatusPageChangeControllerGetResponse, error) {
	return &m.statusPageResponse, m.err
}

func (m *stubClient) BackendWebMonitorErrorControllerGetWithResponse(ctx context.Context,
	params *internal.BackendWebMonitorErrorControllerGetParams,
	reqEditors ...internal.RequestEditorFn) (*internal.BackendWebMonitorErrorControllerGetResponse, error) {
	return &m.errorResponse, m.err
}

func (m *stubClient) BackendWebMonitorListControllerGetWithResponse(ctx context.Context,
	reqEditors ...internal.RequestEditorFn) (*internal.BackendWebMonitorListControllerGetResponse, error) {
	return &m.monitorListResponse, m.err
}
