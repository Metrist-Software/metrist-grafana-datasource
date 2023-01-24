package plugin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// ResourceMonitorList returns a list of monitors which is can be used by a select box
func ResourceMonitorList(ctx context.Context, client internal.ClientWithResponsesInterface) (backend.CallResourceResponse, error) {
	resp, err := client.BackendWebMonitorListControllerGetWithResponse(ctx)
	if err != nil {
		return backend.CallResourceResponse{}, err
	}

	monitorList := *resp.JSON200
	options := make(selectOptions, 0)

	for _, monitor := range monitorList {
		options = append(options, selectOption{
			Label: *monitor.Name,
			Value: *monitor.LogicalName,
		})
	}

	optionsJson, err := json.Marshal(options)
	if err != nil {
		return backend.CallResourceResponse{}, err
	}

	return backend.CallResourceResponse{
		Status: http.StatusOK,
		Body:   optionsJson,
	}, nil
}
