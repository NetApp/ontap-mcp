package rest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
)

func (c *Client) GetCIFSShare(ctx context.Context) ([]string, error) {
	var (
		cifsShare  ontap.GetData
		buf        bytes.Buffer
		statusCode int
		cifsShares []string
	)
	responseHeaders := http.Header{}

	params := url.Values{}

	builder := c.baseRequestBuilder(`/api/protocols/cifs/shares`, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
		ToJSON(&cifsShare).
		Params(params)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return []string{}, err
	}

	if cifsShare.NumRecords == 0 {
		return []string{}, errors.New("no cifs share found in the cluster")
	}

	for _, cifsShareData := range cifsShare.Records {
		cifsShares = append(cifsShares, cifsShareData.Name)
	}

	return cifsShares, nil
}

func (c *Client) CreateCIFSShare(ctx context.Context, cifsShare ontap.CIFSShare) error {
	var (
		buf        bytes.Buffer
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/cifs/shares`, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
		BodyJSON(cifsShare)

	err := c.buildAndExecuteRequest(ctx, builder)

	if statusCode == http.StatusCreated || statusCode == http.StatusAccepted {
		return nil
	}
	return err
}

func (c *Client) UpdateCIFSShare(ctx context.Context, svmName, cifsShareName string, cifsShare ontap.CIFSShare) error {
	var (
		statusCode int
		cShare     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", cifsShareName)
	params.Set("svm", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/cifs/shares`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&cShare)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if cShare.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of cifs share %s because it does not exist", cifsShareName)
	}

	builder = c.baseRequestBuilder(`/api/protocols/cifs/shares/`+cShare.Records[0].Svm.UUID+`/`+cifsShareName, &statusCode, responseHeaders).
		BodyJSON(cifsShare).
		ToJSON(&cShare).
		Patch()

	err = c.buildAndExecuteRequest(ctx, builder)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}

func (c *Client) DeleteCIFSShare(ctx context.Context, cifsShare ontap.CIFSShare) error {
	var (
		statusCode int
		cShare     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("name", cifsShare.Name)
	params.Set("svm", cifsShare.SVM.Name)

	builder := c.baseRequestBuilder(`/api/protocols/cifs/shares`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&cShare)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if cShare.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of cifs share %s because it does not exist", cifsShare.Name)
	}

	builder = c.baseRequestBuilder(`/api/protocols/cifs/shares/`+cShare.Records[0].Svm.UUID+`/`+cifsShare.Name, &statusCode, responseHeaders).
		Delete()

	err = c.buildAndExecuteRequest(ctx, builder)

	if statusCode == http.StatusOK {
		return nil
	}
	return err
}
