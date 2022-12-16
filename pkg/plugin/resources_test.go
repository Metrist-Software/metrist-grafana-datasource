package plugin

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/metrist/metrist/pkg/internal"
)

func TestResourceMonitorList(t *testing.T) {
	type args struct {
		client internal.ClientWithResponsesInterface
		apiKey string
	}
	tests := []struct {
		name    string
		args    args
		want    backend.CallResourceResponse
		wantErr bool
	}{
		{
			name: "serializes list of monitors properly properly",
			args: args{
				apiKey: "OK",
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
				apiKey: "OK",
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
			got, err := ResourceMonitorList(context.Background(), tt.args.client, tt.args.apiKey)
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
