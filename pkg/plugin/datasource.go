package plugin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
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
	errTimerangeLimitExceeded = errors.New("time range cannot exceed 90 days")
)

const (
	durationThreeMonths = 3 * 30 * 24 * time.Hour
)

// NewDatasource creates a new datasource instance.
func NewDatasource(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	logRequestMeta := func(ctx context.Context, req *http.Request) error {
		log.DefaultLogger.Debug("request url: %s, header %s", req.URL.String(), req.Header)
		return nil
	}

	opts, err := settings.HTTPClientOptions()
	if err != nil {
		return nil, fmt.Errorf("http client options: %w", err)
	}

	opts.ConfigureTLSConfig = func(opts httpclient.Options, tlsConfig *tls.Config) {
		if internal.Environment == "local" {
			// We skip TLS verification if running against local as self signed certificates may be being used
			tlsConfig.InsecureSkipVerify = true
		}
	}

	cl, err := httpclient.New(opts)
	if err != nil {
		return nil, fmt.Errorf("httpclient new: %w", err)
	}

	apiKey, ok := settings.DecryptedSecureJSONData["apiKey"]
	if !ok || apiKey == "" {
		return nil, errMissingApiKey
	}

	openApiClient, err := internal.NewClientWithResponses(internal.Endpoint(), internal.WithHTTPClient(cl), internal.WithRequestEditorFn(withAPIKey(apiKey)), internal.WithRequestEditorFn(logRequestMeta))
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

	switch qm.QueryType {
	case "GetMonitorErrors":
		return QueryMonitorErrors(ctx, query, d.openApiClient)
	case "GetMonitorTelemetry":
		return QueryMonitorTelemetry(ctx, query, d.openApiClient)
	case "GetMonitorStatusPageChanges":
		return QueryMonitorStatusPageChanges(ctx, query, d.openApiClient)
	default:
		return backend.DataResponse{}, nil
	}
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	resp, err := d.openApiClient.BackendWebVerifyAuthControllerGetWithResponse(ctx)
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
	out, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	log.DefaultLogger.Debug("test123 =====================================")
	log.DefaultLogger.Debug(string(out))
	log.DefaultLogger.Debug(req.URL)

	// Parameters from getResource come in as query string parameters in the URL property
	var queryStringValues url.Values
	var er error
	if strings.Index(req.URL, "?") > 0 {
		queryStringValues, er = url.ParseQuery(strings.Split(req.URL, "?")[1])
	}

	if er != nil {
		return er
	}

	switch req.Path {
	case "Monitors":
		response, err := ResourceMonitorList(ctx, d.openApiClient)
		if err != nil {
			log.DefaultLogger.Error("resource monitor list error: %w", err)
			return sender.Send(&backend.CallResourceResponse{
				Status: http.StatusInternalServerError,
				Body:   []byte(fmt.Sprintf(`{"message": "%s"}`, "internal server error")),
			})
		}
		return sender.Send(&response)
	case "Checks":
		log.DefaultLogger.Debug("GETTING CHECKS")
		response, err := ResourceCheckList(ctx, d.openApiClient, strings.Split(queryStringValues["monitors"][0], ","), queryStringValues["includeShared"][0] == "true")
		if err != nil {
			log.DefaultLogger.Error("checks list error: %w", err)
			return sender.Send(&backend.CallResourceResponse{
				Status: http.StatusInternalServerError,
				Body:   []byte(fmt.Sprintf(`{"message": "%s"}`, "internal server error")),
			})
		}
		return sender.Send(&response)
	case "Instances":
		response, err := ResourceInstanceList(ctx, d.openApiClient, strings.Split(queryStringValues["monitors"][0], ","), queryStringValues["includeShared"][0] == "true")
		if err != nil {
			log.DefaultLogger.Error("instances list error: %w", err)
			return sender.Send(&backend.CallResourceResponse{
				Status: http.StatusInternalServerError,
				Body:   []byte(fmt.Sprintf(`{"message": "%s"}`, "internal server error")),
			})
		}
		return sender.Send(&response)
	case "BuildHash":
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusOK,
			Body:   []byte(fmt.Sprintf(`{"hash": "%s"}`, internal.BuildHash)),
		})
	default:
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusNotFound,
		})
	}
}

func ensureTimeRangeWithinLimits(duration time.Duration) error {
	if duration.Truncate(time.Hour) > durationThreeMonths {
		return errTimerangeLimitExceeded
	}

	return nil
}
