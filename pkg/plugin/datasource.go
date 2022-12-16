package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/metrist/metrist/pkg/internal"
)

var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ backend.CallResourceHandler   = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

var (
	errRemoteRequest          = errors.New("remote request error")
	errRemoteResponse         = errors.New("remote response error")
	errMissingApiKey          = errors.New("missing api key")
	errTimerangeLimitExceeded = errors.New("time range cannot exceed 3 months long")
)

const (
	durationThreeMonths = 3 * 30 * 24 * time.Hour
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
		if err := ensureTimeRangeWithinLimits(q.TimeRange.Duration()); err != nil {
			log.DefaultLogger.Error("time range error: %w", err)
			response.Responses[q.RefID] = backend.ErrDataResponse(backend.StatusBadRequest, err.Error())
			continue
		}

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

	apiKey, err := requireApiKey(pCtx)
	if err != nil {
		return backend.DataResponse{}, err
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
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	apiKey, err := requireApiKey(req.PluginContext)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}

	resp, err := d.openApiClient.BackendWebVerifyAuthControllerGetWithResponse(ctx, withAPIKey(apiKey))
	if err != nil {
		log.DefaultLogger.Debug("verify auth controller error: %w", err)
		return nil, err
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "Data source is working!",
		}, nil
	case http.StatusUnauthorized:
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Unauthorized: Invalid API Key",
		}, nil
	default:
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: resp.Status(),
		}, nil
	}
}

// CallResource implements backend.CallResourceHandler
func (d *Datasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	apiKey, err := requireApiKey(req.PluginContext)
	if err != nil {
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusUnauthorized,
			Body:   []byte(fmt.Sprintf(`{"message": "%s"}`, err.Error())),
		})
	}

	switch req.Path {
	case "Monitors":
		response, err := ResourceMonitorList(ctx, d.openApiClient, apiKey)
		if err != nil {
			log.DefaultLogger.Error("resource monitor list error: %w", err)
			return sender.Send(&backend.CallResourceResponse{
				Status: http.StatusInternalServerError,
				Body:   []byte(fmt.Sprintf(`{"message": "%s"}`, "internal server error")),
			})
		}
		return sender.Send(&response)
	default:
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusNotFound,
		})
	}
}

func requireApiKey(ctx backend.PluginContext) (string, error) {
	apiKey, ok := ctx.DataSourceInstanceSettings.DecryptedSecureJSONData["apiKey"]
	log.DefaultLogger.Debug("api key %v", apiKey)
	if !ok || apiKey == "" {
		return "", errMissingApiKey
	}
	return apiKey, nil
}

func ensureTimeRangeWithinLimits(duration time.Duration) error {
	if duration > durationThreeMonths {
		return errTimerangeLimitExceeded
	}

	return nil
}
