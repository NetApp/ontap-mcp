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

func (c *Client) GetVolume(ctx context.Context, volume ontap.Volume) ([]string, error) {
	var (
		vol     ontap.GetData
		volumes []string
	)
	responseHeaders := http.Header{}

	// If we only have the volume name we need to find the volume's UUID

	params := url.Values{}
	svmName := volume.SVM.Name
	if svmName != "" {
		params.Set("svm", svmName)
	}

	builder := c.baseRequestBuilder(`/api/storage/volumes`, nil, responseHeaders).
		Params(params).
		ToJSON(&vol)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return []string{}, err
	}

	if vol.NumRecords == 0 {
		if svmName != "" {
			return []string{}, fmt.Errorf("no volumes found on svm: %s", svmName)
		}
		return []string{}, errors.New("no volumes found in the cluster")
	}

	for _, v := range vol.Records {
		volumes = append(volumes, v.Name)
	}

	return volumes, nil
}

func (c *Client) CreateVolume(ctx context.Context, volume ontap.Volume) error {
	var (
		buf        bytes.Buffer
		statusCode int
		oc         ontap.OnlyCount
	)
	responseHeaders := http.Header{}

	// If an export policy is included, check if it exists. If it does not, create it
	if volume.Nas.ExportPolicy.Name != "" {
		params := url.Values{}
		params.Set("return_records", "false")
		params.Set("fields", "name")
		params.Set("name", volume.Nas.ExportPolicy.Name)
		params.Set("svm.name", volume.SVM.Name)

		builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, &statusCode, responseHeaders).
			ToBytesBuffer(&buf).
			ToJSON(&oc).
			Params(params)

		err := c.buildAndExecuteRequest(ctx, builder)

		if err != nil {
			return err
		}

		if oc.NumRecords == 0 {
			// This is OK, create it
			err := c.CreateExportPolicy(ctx, volume)
			if err != nil {
				return err
			}
		} else if oc.NumRecords != 1 {
			return fmt.Errorf("failed to create volume=%s on svm=%s with export policy=%s because there are %d matching export policies",
				volume.Name, volume.SVM.Name, volume.Nas.ExportPolicy.Name, oc.NumRecords)
		}
	}

	builder := c.baseRequestBuilder(`/api/storage/volumes`, &statusCode, responseHeaders).
		BodyJSON(volume).
		ToBytesBuffer(&buf)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) UpdateVolume(ctx context.Context, volume ontap.Volume, oldVolumeName string, svmName string) error {
	var (
		buf        bytes.Buffer
		statusCode int
		vol        ontap.GetData
	)
	responseHeaders := http.Header{}

	// If we only have the volume name we need to find the volume's UUID

	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", oldVolumeName)
	params.Set("svm", svmName)

	builder := c.baseRequestBuilder(`/api/storage/volumes`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&vol)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if vol.NumRecords == 0 {
		return fmt.Errorf("failed to update volume=%s on svm=%s because it does not exist", oldVolumeName, svmName)
	}
	if vol.NumRecords != 1 {
		return fmt.Errorf("failed to update volume=%s on svm=%s because there are %d matching records",
			oldVolumeName, svmName, vol.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/storage/volumes/`+vol.Records[0].UUID, &statusCode, responseHeaders).
		Patch().
		ToBytesBuffer(&buf).
		BodyJSON(volume)

	err = c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return fmt.Errorf("error during update volume request: %w", err)
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) DeleteVolume(ctx context.Context, volume ontap.Volume) error {
	var (
		buf        bytes.Buffer
		statusCode int
		vol        ontap.GetData
	)
	responseHeaders := http.Header{}

	// If we only have the volume name we need to find the volume's UUID

	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", volume.Name)
	params.Set("svm", volume.SVM.Name)

	builder := c.baseRequestBuilder(`/api/storage/volumes`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&vol)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if vol.NumRecords == 0 {
		return fmt.Errorf("failed to delete volume=%s on svm=%s because it does not exist", volume.Name, volume.SVM.Name)
	}
	if vol.NumRecords != 1 {
		return fmt.Errorf("failed to delete volume=%s on svm=%s because there are %d matching records",
			volume.Name, volume.SVM.Name, vol.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/storage/volumes/`+vol.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) CreateExportPolicy(ctx context.Context, volume ontap.Volume) error {
	var statusCode int
	newExportPolicy := ontap.NameAndSVM{
		Name: volume.Nas.ExportPolicy.Name,
		Svm:  volume.SVM,
	}

	builder := c.baseRequestBuilder(`/api/protocols/nfs/export-policies`, &statusCode, nil).
		BodyJSON(newExportPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)
	if err != nil {
		return err
	}

	if statusCode != http.StatusCreated && statusCode != http.StatusAccepted {
		return fmt.Errorf(`unexpected status code: %d`, statusCode)
	}

	return nil
}
