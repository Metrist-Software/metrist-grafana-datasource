package plugin

import (
	"context"

	"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
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
	checksResponse      internal.BackendWebMonitorCheckControllerGetResponse
	instancesResponse   internal.BackendWebMonitorInstanceControllerGetResponse
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

func (m *stubClient) BackendWebMonitorCheckControllerGetWithResponse(ctx context.Context,
	params *internal.BackendWebMonitorCheckControllerGetParams,
	reqEditors ...internal.RequestEditorFn) (*internal.BackendWebMonitorCheckControllerGetResponse, error) {
	return &m.checksResponse, m.err
}

func (m *stubClient) BackendWebMonitorInstanceControllerGetWithResponse(ctx context.Context,
	params *internal.BackendWebMonitorInstanceControllerGetParams,
	reqEditors ...internal.RequestEditorFn) (*internal.BackendWebMonitorInstanceControllerGetResponse, error) {
	return &m.instancesResponse, m.err
}
