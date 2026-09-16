package rest

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) CreateClusterPeer(ctx context.Context, destinationClient *Client, sourceCluster, destinationCluster string) error {
	passphrase := rand.Text()

	// Step1: fetch inter cluster LIFs of source cluster
	sourceLIFs, err := c.fetchInterClusterLIFs(ctx, sourceCluster)
	if err != nil {
		return err
	}

	// Step2: fetch inter cluster LIFs of destination cluster
	destinationLIFs, err := destinationClient.fetchInterClusterLIFs(ctx, destinationCluster)
	if err != nil {
		return err
	}

	// Step3: Generate passphrase in source cluster with destination LIFs
	cp := ontap.ClusterPeer{RemotePeer: ontap.RemotePeer{IPaddresses: destinationLIFs}, Authentication: ontap.Authentication{Passphrase: passphrase}}
	if err := c.managePassphrase(ctx, cp); err != nil {
		return err
	}

	// Step4: Approve passphrase in destination cluster with source LIFs
	cp = ontap.ClusterPeer{RemotePeer: ontap.RemotePeer{IPaddresses: sourceLIFs}, Authentication: ontap.Authentication{Passphrase: passphrase}}
	if err := destinationClient.managePassphrase(ctx, cp); err != nil {
		fmt.Println("failed to create cluster peer relationships in destination cluster, rolling back in source cluster")
		destinationClusterName, err1 := destinationClient.fetchClusterName(ctx)
		if err1 != nil {
			return err1
		}
		if err2 := c.removeClusterPeer(ctx, destinationClusterName); err2 != nil {
			return err2
		}
		return err
	}
	return nil
}

func (c *Client) fetchInterClusterLIFs(ctx context.Context, cluster string) ([]string, error) {
	var (
		statusCode int
		icls       []string
		icl        ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("fields", "ip.address")
	params.Set("service_policy.name", "default-intercluster")
	params.Set("services", "intercluster_core")

	builder := c.baseRequestBuilder(`/api/network/ip/interfaces`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&icl)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return []string{}, err
	}

	if icl.NumRecords == 0 {
		return []string{}, fmt.Errorf("failed to find intercluster LIFS for cluster=%s because it does not exist", cluster)
	}

	for _, lifData := range icl.Records {
		icls = append(icls, lifData.IP.Address)
	}

	return icls, nil
}

func (c *Client) managePassphrase(ctx context.Context, clusterPeer ontap.ClusterPeer) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}
	builder := c.baseRequestBuilder(`/api/cluster/peers`, &statusCode, responseHeaders).
		BodyJSON(clusterPeer).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		if !strings.Contains(err.Error(), "4653075") {
			return err
		}
		fmt.Println("cluster peer relationship already exists")
		return nil
	}
	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) DeleteClusterPeer(ctx context.Context, destinationClient *Client) error {
	// Step1: fetch source and destination cluster names
	sourceClusterName, err := c.fetchClusterName(ctx)
	if err != nil {
		return err
	}
	destinationClusterName, err := destinationClient.fetchClusterName(ctx)
	if err != nil {
		return err
	}

	// Step2: delete cluster peer from source cluster
	destinationUUID, err := c.fetchClusterPeerUUID(ctx, destinationClusterName)
	if err != nil {
		return err
	}
	err1 := c.removeClusterPeer(ctx, destinationUUID)

	// Step3: delete cluster peer from destination cluster
	sourceUUID, err := c.fetchClusterPeerUUID(ctx, sourceClusterName)
	if err != nil {
		return err
	}
	err2 := destinationClient.removeClusterPeer(ctx, sourceUUID)
	if err1 != nil || err2 != nil {
		return errors.Join(err1, err2)
	}
	return nil
}

func (c *Client) removeClusterPeer(ctx context.Context, cpUUID string) error {
	var (
		statusCode int
		buf        bytes.Buffer
	)
	responseHeaders := http.Header{}
	builder := c.baseRequestBuilder(`/api/cluster/peers/`+url.PathEscape(cpUUID), &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) fetchClusterName(ctx context.Context) (string, error) {
	var (
		statusCode int
		cl         ontap.Cluster
	)
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "name")

	builder := c.baseRequestBuilder(`/api/cluster`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&cl)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}

	return cl.Name, nil
}

func (c *Client) fetchClusterPeerUUID(ctx context.Context, remoteClusterName string) (string, error) {
	var (
		statusCode int
		cp         ontap.GetData
	)
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("remote.name", remoteClusterName)

	builder := c.baseRequestBuilder(`/api/cluster/peers`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&cp)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}

	if cp.NumRecords == 0 {
		return "", fmt.Errorf("failed to find cluster peer relationships for remote cluster=%s because it does not exist", remoteClusterName)
	}
	if cp.NumRecords != 1 {
		return "", fmt.Errorf("failed to find cluster peer relationships for remote cluster=%s because there are %d matching records", remoteClusterName, cp.NumRecords)
	}

	return cp.Records[0].UUID, nil
}
