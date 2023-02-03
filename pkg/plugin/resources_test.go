package plugin

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type args struct {
	client internal.ClientWithResponsesInterface
}

type test struct {
	name    string
	args    args
	want    backend.CallResourceResponse
	wantErr bool
}

func TestResourceMonitorList(t *testing.T) {
	tests := []test{
		{
			name: "serializes list of monitors properly properly",
			args: args{
				client: &stubClient{monitorListResponse: internal.BackendWebMonitorListControllerGetResponse{
					JSON200: &internal.MonitorList{{LogicalName: ptr("AWS Lambda"), Name: ptr("awslambda")}},
				}},
			},
			want: backend.CallResourceResponse{
				Status: http.StatusOK,
				Body:   []byte("[{\"label\":\"awslambda\",\"value\":\"AWS Lambda\"}]"),
			},
			wantErr: false,
		},
		{
			name: "handles empty monitor list",
			args: args{
				client: &stubClient{monitorListResponse: internal.BackendWebMonitorListControllerGetResponse{
					JSON200: &internal.MonitorList{},
				}},
			},
			want: backend.CallResourceResponse{
				Status: http.StatusOK,
				Body:   []byte("[]"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResourceMonitorList(context.Background(), tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResourceMonitorList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResourceMonitorList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceChecksList(t *testing.T) {
	tests := []test{
		{
			name: "serializes list of checks properly properly with proper combining of monitor names",
			args: args{
				client: &stubClient{checksResponse: internal.BackendWebMonitorCheckControllerGetResponse{
					JSON200: &internal.MonitorChecks{
						{
							Checks: &[]internal.MonitorCheck{
								{
									LogicalName: ptr("check1"),
									Name:        ptr("Check One"),
								},
							},
							MonitorLogicalName: ptr("mon_one"),
						},
						{
							Checks: &[]internal.MonitorCheck{
								{
									LogicalName: ptr("check3"),
									Name:        ptr("Check Three"),
								},
							},
							MonitorLogicalName: ptr("mon_two"),
						},
					},
				}},
			},
			want: backend.CallResourceResponse{
				Status: http.StatusOK,
				Body:   []byte(`[{"label":"mon_one:Check One","value":"check1"},{"label":"mon_two:Check Three","value":"check3"}]`),
			},
			wantErr: false,
		},
		{
			name: "handles empty checks list",
			args: args{
				client: &stubClient{checksResponse: internal.BackendWebMonitorCheckControllerGetResponse{
					JSON200: &internal.MonitorChecks{},
				}},
			},
			want: backend.CallResourceResponse{
				Status: http.StatusOK,
				Body:   []byte("[]"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResourceCheckList(context.Background(), tt.args.client, []string{"testsignal"}, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResourceChecks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResourceChecks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInstancesList(t *testing.T) {
	tests := []test{
		{
			name: "serializes list of instances properly removing duplicates",
			args: args{
				client: &stubClient{instancesResponse: internal.BackendWebMonitorInstanceControllerGetResponse{
					JSON200: &internal.MonitorInstances{
						{
							Instances: &[]string{
								"instance1",
							},
							MonitorLogicalName: ptr("mon_one"),
						},
						{
							Instances: &[]string{
								"instance1",
								"instance2",
							},
							MonitorLogicalName: ptr("mon_two"),
						},
					},
				}},
			},
			want: backend.CallResourceResponse{
				Status: http.StatusOK,
				Body:   []byte(`[{"label":"instance1","value":"instance1"},{"label":"instance2","value":"instance2"}]`),
			},
			wantErr: false,
		},
		{
			name: "handles empty instances list",
			args: args{
				client: &stubClient{instancesResponse: internal.BackendWebMonitorInstanceControllerGetResponse{
					JSON200: &internal.MonitorInstances{},
				}},
			},
			want: backend.CallResourceResponse{
				Status: http.StatusOK,
				Body:   []byte("[]"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResourceInstanceList(context.Background(), tt.args.client, []string{"testsignal"}, true)
			println(string(got.Body))
			if (err != nil) != tt.wantErr {
				t.Errorf("ResourceInstances() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResourceInstances() = %v, want %v", got, tt.want)
			}
		})
	}
}
