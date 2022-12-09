package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/metrist/metrist/pkg/internal"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces- only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

var (
	errRemoteRequest  = errors.New("remote request error")
	errRemoteResponse = errors.New("remote response error")
	errMissingApiKey  = errors.New("missing api key")
)

// NewDatasource creates a new datasource instance.
func NewDatasource(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	opts, err := settings.HTTPClientOptions()
	if err != nil {
		return nil, fmt.Errorf("http client options: %w", err)
	}
	cl, err := httpclient.New(opts)
	if err != nil {
		return nil, fmt.Errorf("httpclient new: %w", err)
	}
	openApiClient, err := internal.NewClientWithResponses(internal.Endpoint(), internal.WithHTTPClient(cl))
	if err != nil {
		return nil, fmt.Errorf("internal new client: %w", err)
	}
	return &Datasource{
		settings:      settings,
		httpClient:    cl,
		openApiClient: openApiClient,
	}, nil
}

type Datasource struct {
	settings      backend.DataSourceInstanceSettings
	httpClient    *http.Client
	openApiClient internal.ClientWithResponsesInterface
}

func (d *Datasource) Dispose() {
	d.httpClient.CloseIdleConnections()
}

// QueryData go through each query and routes them to the appropriate query handler
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Debug("QueryData called", "numQueries", len(req.Queries))

	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		res, err := d.query(ctx, req.PluginContext, q)

		if err != nil {
			log.DefaultLogger.Error("error %v", err)
		}

		switch {
		case err == nil:
			// Do nothing
		case errors.Is(err, context.DeadlineExceeded):
			res = backend.ErrDataResponse(backend.StatusTimeout, "gateway timeout")
		case errors.Is(err, errRemoteRequest):
			res = backend.ErrDataResponse(backend.StatusBadGateway, "bad gateway request")
		case errors.Is(err, errRemoteResponse):
			res = backend.ErrDataResponse(backend.StatusValidationFailed, "bad gateway response")
		default:
			res = backend.ErrDataResponse(backend.StatusInternal, err.Error())
		}

		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *Datasource) query(ctx context.Context, pCtx backend.PluginContext, query backend.DataQuery) (backend.DataResponse, error) {
	var qm queryModel
	if err := json.Unmarshal(query.JSON, &qm); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, "json unmarshal: "+err.Error()), err
	}

	apiKey, ok := pCtx.DataSourceInstanceSettings.DecryptedSecureJSONData["apiKey"]
	if !ok {
		return backend.DataResponse{}, errMissingApiKey
	}

	switch qm.QueryType {
	case "GetMonitorErrors":
		return QueryMonitorErrors(ctx, query, d.openApiClient, apiKey)
	case "GetMonitorTelemetry":
		return QueryMonitorTelemetry(ctx, query, d.openApiClient, apiKey)
	case "GetMonitorStatusPageChanges":
		return QueryMonitorStatusPageChanges(ctx, query, d.openApiClient, apiKey)
	default:
		return backend.DataResponse{}, nil
	}
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Debug("CheckHealth called")

	var status = backend.HealthStatusOk
	var message = "Data source is working"

	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}
