package rest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/netapp/ontap-mcp/config"
)

type replicationRoundTripFunc func(*http.Request) (*http.Response, error)

func (f replicationRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestReplicationQueries(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantQuery map[string]string
		response  string
		invoke    func(context.Context, *Client) error
	}{
		{
			name: "intercluster lifs",
			path: "/api/network/ip/interfaces",
			wantQuery: map[string]string{
				"services": "intercluster_core",
				"fields":   "name,enabled,state,ip.address,location.node.name",
			},
			response: `{"records":[{"name":"icl1","enabled":true,"state":"up","ip":{"address":"192.0.2.1"},"location":{"node":{"name":"node1"}}}],"num_records":1}`,
			invoke: func(ctx context.Context, client *Client) error {
				records, err := client.GetInterclusterLIFs(ctx)
				if err == nil && (len(records) != 1 || records[0].Name != "icl1" || records[0].IP.Address != "192.0.2.1") {
					t.Fatalf("unexpected LIF records: %#v", records)
				}
				return err
			},
		},
		{
			name: "cluster peers",
			path: "/api/cluster/peers",
			wantQuery: map[string]string{
				"name":   "cluster-b",
				"fields": "name,uuid,status",
			},
			response: `{"records":[{"name":"cluster-b","uuid":"peer-1","status":{"state":"available"}}],"num_records":1}`,
			invoke: func(ctx context.Context, client *Client) error {
				records, err := client.GetClusterPeers(ctx, "cluster-b")
				if err == nil && (len(records) != 1 || records[0].Status.State != "available") {
					t.Fatalf("unexpected cluster peer records: %#v", records)
				}
				return err
			},
		},
		{
			name: "svm peers",
			path: "/api/svm/peers",
			wantQuery: map[string]string{
				"svm.name":          "svm-a",
				"peer.cluster.name": "cluster-b",
				"peer.svm.name":     "svm-b",
				"fields":            "uuid,state,svm.name,peer.cluster.name,peer.svm.name",
			},
			response: `{"records":[{"uuid":"svm-peer-1","state":"peered","svm":{"name":"svm-a"},"peer":{"cluster":{"name":"cluster-b"},"svm":{"name":"svm-b"}}}],"num_records":1}`,
			invoke: func(ctx context.Context, client *Client) error {
				records, err := client.GetSVMPeers(ctx, "svm-a", "cluster-b", "svm-b")
				if err == nil && (len(records) != 1 || records[0].State != "peered" || records[0].Peer.SVM.Name != "svm-b") {
					t.Fatalf("unexpected SVM peer records: %#v", records)
				}
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := replicationRoundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", request.Method)
				}
				if request.URL.Path != test.path {
					t.Fatalf("path = %s, want %s", request.URL.Path, test.path)
				}
				for key, want := range test.wantQuery {
					if got := request.URL.Query().Get(key); got != want {
						t.Errorf("query %s = %q, want %q", key, got, want)
					}
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(test.response)),
				}, nil
			})
			client := NewWithClient(&config.Poller{Addr: "cluster.test"}, &http.Client{Transport: transport})
			if err := test.invoke(context.Background(), client); err != nil {
				t.Fatalf("query returned error: %v", err)
			}
		})
	}
}
