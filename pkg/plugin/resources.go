package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"golang.org/x/exp/slices"

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

func ResourceCheckList(ctx context.Context, client internal.ClientWithResponsesInterface, monitors []string, includeShared bool) (backend.CallResourceResponse, error) {
	params := internal.BackendWebMonitorCheckControllerGetParams{M: monitors, IncludeShared: &includeShared}

	resp, err := client.BackendWebMonitorCheckControllerGetWithResponse(ctx, &params)
	if err != nil {
		return backend.CallResourceResponse{}, err
	}

	checkList := *resp.JSON200
	options := make(selectOptions, 0)

	for _, item := range checkList {
		for _, check := range *item.Checks {
			options = append(options, selectOption{
				Label: fmt.Sprintf("%s:%s", *item.MonitorLogicalName, *check.Name),
				Value: *check.LogicalName,
			})
		}
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	optionsJson, err := json.Marshal(options)
	if err != nil {
		return backend.CallResourceResponse{}, err
	}

	return backend.CallResourceResponse{
		Status: http.StatusOK,
		Body:   optionsJson,
	}, nil
}

func ResourceInstanceList(ctx context.Context, client internal.ClientWithResponsesInterface, monitors []string, includeShared bool) (backend.CallResourceResponse, error) {
	params := internal.BackendWebMonitorInstanceControllerGetParams{M: monitors, IncludeShared: &includeShared}

	resp, err := client.BackendWebMonitorInstanceControllerGetWithResponse(ctx, &params)
	if err != nil {
		return backend.CallResourceResponse{}, err
	}

	instanceList := *resp.JSON200

	all_instances := make([]string, 0)
	for _, item := range instanceList {
		all_instances = append(all_instances, *item.Instances...)
	}
	all_instances = uniqStrings(all_instances)

	slices.Sort(all_instances)

	options := make(selectOptions, 0)
	for _, instance := range all_instances {
		options = append(options, selectOption{
			Label: instance,
			Value: instance,
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
