package rest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/netapp/ontap-mcp/ontap"
)

var ErrSVMPeerNotFound = errors.New("SVM peer relationship not found")

func (c *Client) CreateSVM(ctx context.Context, svm ontap.SVMCreate) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/svm/svms`, &statusCode, responseHeaders).
		BodyJSON(svm).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) UpdateSVM(ctx context.Context, svm ontap.SVM, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
		svmData    ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/svm/svms`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&svmData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if svmData.NumRecords == 0 {
		return fmt.Errorf("failed to get details of SVM %s because it does not exist", svmName)
	}
	if svmData.NumRecords != 1 {
		return fmt.Errorf("failed to get details of SVM %s because there are %d matching records",
			svmName, svmData.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/svm/svms/`+svmData.Records[0].UUID, &statusCode, responseHeaders).
		Patch().
		ToBytesBuffer(&buf).
		BodyJSON(svm)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) DeleteSVM(ctx context.Context, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
		svmData    ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/svm/svms`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&svmData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if svmData.NumRecords == 0 {
		return fmt.Errorf("failed to get details of SVM %s because it does not exist", svmName)
	}
	if svmData.NumRecords != 1 {
		return fmt.Errorf("failed to get details of SVM %s because there are %d matching records",
			svmName, svmData.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/svm/svms/`+svmData.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) FindSVMPeer(ctx context.Context, localSVM, remoteSVM, remoteCluster string) (ontap.SVMPeer, error) {
	var (
		statusCode  int
		svmPeerData ontap.SVMPeerCollection
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", localSVM)
	params.Set("peer.svm.name", remoteSVM)
	if remoteCluster != "" {
		params.Set("peer.cluster.name", remoteCluster)
	}
	params.Set("fields", "uuid,state,applications,svm.name,peer.svm.name,peer.cluster.name")

	builder := c.baseRequestBuilder(`/api/svm/peers`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&svmPeerData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return ontap.SVMPeer{}, err
	}

	if svmPeerData.NumRecords == 0 {
		return ontap.SVMPeer{}, fmt.Errorf("%w for local SVM %s, remote SVM %s, remote cluster %s", ErrSVMPeerNotFound, localSVM, remoteSVM, remoteCluster)
	}
	if svmPeerData.NumRecords != 1 {
		return ontap.SVMPeer{}, fmt.Errorf("failed to uniquely identify SVM peer for local SVM %s, remote SVM %s, remote cluster %s because there are %d matching records",
			localSVM, remoteSVM, remoteCluster, svmPeerData.NumRecords)
	}
	if svmPeerData.Records[0].UUID == "" {
		return ontap.SVMPeer{}, fmt.Errorf("empty UUID found for local SVM %s, remote SVM %s, remote cluster %s", localSVM, remoteSVM, remoteCluster)
	}

	return svmPeerData.Records[0], nil
}

func (c *Client) CreateSVMPeer(ctx context.Context, svmPeer ontap.SVMPeer) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/svm/peers`, &statusCode, responseHeaders).
		BodyJSON(svmPeer).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) UpdateSVMPeer(ctx context.Context, uuid string, svmPeer ontap.SVMPeer) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/svm/peers/`+url.PathEscape(uuid), &statusCode, responseHeaders).
		Patch().
		BodyJSON(svmPeer).
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

func (c *Client) DeleteSVMPeer(ctx context.Context, uuid string) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/svm/peers/`+url.PathEscape(uuid), &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, &buf)
}

// getSVMUUID looks up the UUID of an SVM by name.
func (c *Client) getSVMUUID(ctx context.Context, svmName string) (string, error) {
	var (
		statusCode int
		svmData    ontap.GetData
	)
	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", svmName)
	params.Set("fields", "uuid")

	builder := c.baseRequestBuilder(`/api/svm/svms`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&svmData)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return "", err
	}

	if svmData.NumRecords == 0 {
		return "", fmt.Errorf("failed to get details of SVM %s because it does not exist", svmName)
	}
	if svmData.NumRecords != 1 {
		return "", fmt.Errorf("failed to get details of SVM %s because there are %d matching records", svmName, svmData.NumRecords)
	}

	return svmData.Records[0].UUID, nil
}
