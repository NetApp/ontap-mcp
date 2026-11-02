package rest

import (
	"context"
	"encoding/json"
	"net/url"
)

type InterclusterLIF struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	State   string `json:"state"`
	IP      struct {
		Address string `json:"address"`
	} `json:"ip"`
	Location struct {
		Node struct {
			Name string `json:"name"`
		} `json:"node"`
	} `json:"location"`
}

type ClusterPeer struct {
	Name   string `json:"name"`
	UUID   string `json:"uuid"`
	Status struct {
		State   string `json:"state"`
		Message string `json:"message"`
	} `json:"status"`
}

type SVMPeer struct {
	UUID  string `json:"uuid"`
	State string `json:"state"`
	SVM   struct {
		Name string `json:"name"`
	} `json:"svm"`
	Peer struct {
		Cluster struct {
			Name string `json:"name"`
		} `json:"cluster"`
		SVM struct {
			Name string `json:"name"`
		} `json:"svm"`
	} `json:"peer"`
}

func (c *Client) GetInterclusterLIFs(ctx context.Context) ([]InterclusterLIF, error) {
	params := url.Values{}
	params.Set("services", "intercluster_core")
	params.Set("fields", "name,enabled,state,ip.address,location.node.name")

	var result struct {
		Records []InterclusterLIF `json:"records"`
	}
	raw, err := c.GenericGet(ctx, "/network/ip/interfaces", params, 0)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Records, nil
}

func (c *Client) GetClusterPeers(ctx context.Context, peerCluster string) ([]ClusterPeer, error) {
	params := url.Values{}
	params.Set("name", peerCluster)
	params.Set("fields", "name,uuid,status")

	var result struct {
		Records []ClusterPeer `json:"records"`
	}
	raw, err := c.GenericGet(ctx, "/cluster/peers", params, 0)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Records, nil
}

func (c *Client) GetSVMPeers(ctx context.Context, localSVM, peerCluster, peerSVM string) ([]SVMPeer, error) {
	params := url.Values{}
	params.Set("svm.name", localSVM)
	params.Set("peer.cluster.name", peerCluster)
	params.Set("peer.svm.name", peerSVM)
	params.Set("fields", "uuid,state,svm.name,peer.cluster.name,peer.svm.name")

	var result struct {
		Records []SVMPeer `json:"records"`
	}
	raw, err := c.GenericGet(ctx, "/svm/peers", params, 0)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Records, nil
}
