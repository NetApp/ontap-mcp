package rest

import (
	"context"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) CreateNVMeService(ctx context.Context, nvmeService ontap.NVMeService) error {
	var (
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/nvme/services`, &statusCode, responseHeaders).
		BodyJSON(nvmeService)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateNVMeService(ctx context.Context, svmName string, nvmeService ontap.NVMeService) error {
	var (
		statusCode int
		nvmeSr     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/nvme/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeSr)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme service in svm %s because it does not exist", svmName)
	}

	builder = c.baseRequestBuilder(`/api/protocols/nvme/services/`+nvmeSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		BodyJSON(nvmeService).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNVMeService(ctx context.Context, svmName string) error {
	var (
		statusCode int
		nvmeSr     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)

	builder := c.baseRequestBuilder(`/api/protocols/nvme/services`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeSr)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeSr.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme service in svm %s because it does not exist", svmName)
	}

	builder = c.baseRequestBuilder(`/api/protocols/nvme/services/`+nvmeSr.Records[0].Svm.UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) CreateNVMeSubsystem(ctx context.Context, nvmeSubsystem ontap.NVMeSubsystem) error {
	var (
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/nvme/subsystems`, &statusCode, responseHeaders).
		BodyJSON(nvmeSubsystem)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateNVMeSubsystem(ctx context.Context, svmName string, name string, osType string, nvmeSubsystem ontap.NVMeSubsystem) error {
	var (
		statusCode int
		nvmeSs     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", name)
	params.Set("os_type", osType)

	builder := c.baseRequestBuilder(`/api/protocols/nvme/subsystems`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeSs)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeSs.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme subsystem of name %s in svm %s because it does not exist", name, svmName)
	}

	if nvmeSs.NumRecords != 1 {
		return fmt.Errorf("failed to update NVMe subsystem %s in svm=%s because there are %d matching records",
			name, svmName, nvmeSs.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/protocols/nvme/subsystems/`+nvmeSs.Records[0].UUID, &statusCode, responseHeaders).
		BodyJSON(nvmeSubsystem).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNVMeSubsystem(ctx context.Context, svmName string, name string, osType string, allowDeleteWhileMapped bool, allowDeleteWithHosts bool) error {
	var (
		statusCode int
		nvmeSs     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", name)
	params.Set("os_type", osType)

	builder := c.baseRequestBuilder(`/api/protocols/nvme/subsystems`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeSs)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeSs.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme subsystem of name %s in svm %s because it does not exist", name, svmName)
	}

	if nvmeSs.NumRecords != 1 {
		return fmt.Errorf("failed to delete NVMe subsystem %s in svm=%s because there are %d matching records",
			name, svmName, nvmeSs.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/protocols/nvme/subsystems/`+nvmeSs.Records[0].UUID+`?allow_delete_while_mapped=`+strconv.FormatBool(allowDeleteWhileMapped)+`&allow_delete_with_hosts=`+strconv.FormatBool(allowDeleteWithHosts), &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) AddNVMeSubsystemHost(ctx context.Context, svmName string, name string, osType string, nvmeSubsystemHost ontap.NVMeSubsystemHost) error {
	var (
		statusCode int
		nvmeSs     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", name)
	params.Set("os_type", osType)

	builder := c.baseRequestBuilder(`/api/protocols/nvme/subsystems`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeSs)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeSs.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme subsystem of name %s in svm %s because it does not exist", name, svmName)
	}

	if nvmeSs.NumRecords != 1 {
		return fmt.Errorf("failed to get NVMe subsystem %s in svm=%s because there are %d matching records",
			name, svmName, nvmeSs.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/nvme/subsystems/`+nvmeSs.Records[0].UUID+`/hosts`, &statusCode, responseHeaders).
		BodyJSON(nvmeSubsystemHost)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) RemoveNVMeSubsystemHost(ctx context.Context, svmName string, name string, osType string, nqn string) error {
	var (
		statusCode int
		nvmeSs     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", name)
	params.Set("os_type", osType)

	builder := c.baseRequestBuilder(`/api/protocols/nvme/subsystems`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeSs)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeSs.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme subsystem of name %s in svm %s because it does not exist", name, svmName)
	}

	if nvmeSs.NumRecords != 1 {
		return fmt.Errorf("failed to get NVMe subsystem %s in svm=%s because there are %d matching records",
			name, svmName, nvmeSs.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/protocols/nvme/subsystems/`+nvmeSs.Records[0].UUID+`/hosts/`+url.PathEscape(nqn), &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) CreateNVMeNamespace(ctx context.Context, nvmeNamespace ontap.NVMeNamespace) error {
	var (
		statusCode int
	)
	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/storage/namespaces`, &statusCode, responseHeaders).
		BodyJSON(nvmeNamespace)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) UpdateNVMeNamespace(ctx context.Context, svmName string, name string, osType string, nvmeNamespace ontap.NVMeNamespace) error {
	var (
		statusCode int
		nvmeNs     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", name)
	params.Set("os_type", osType)

	builder := c.baseRequestBuilder(`/api/storage/namespaces`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeNs)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeNs.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme namespace of name %s in svm %s because it does not exist", name, svmName)
	}

	if nvmeNs.NumRecords != 1 {
		return fmt.Errorf("failed to update NVMe namespace %s in svm=%s because there are %d matching records",
			name, svmName, nvmeNs.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/storage/namespaces/`+nvmeNs.Records[0].UUID, &statusCode, responseHeaders).
		BodyJSON(nvmeNamespace).
		Patch()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNVMeNamespace(ctx context.Context, svmName string, name string, osType string, allowDeleteWhileMapped bool) error {
	var (
		statusCode int
		nvmeNs     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", name)
	params.Set("os_type", osType)

	builder := c.baseRequestBuilder(`/api/storage/namespaces`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeNs)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if nvmeNs.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme namespace of name %s in svm %s because it does not exist", name, svmName)
	}

	if nvmeNs.NumRecords != 1 {
		return fmt.Errorf("failed to delete NVMe namespace %s in svm=%s because there are %d matching records",
			name, svmName, nvmeNs.NumRecords)
	}

	builder = c.baseRequestBuilder(`/api/storage/namespaces/`+nvmeNs.Records[0].UUID+`?allow_delete_while_mapped=`+strconv.FormatBool(allowDeleteWhileMapped), &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) CreateNVMeSubsystemMap(ctx context.Context, nvmeSubsystemMap ontap.NVMeSubsystemMap) error {
	var (
		statusCode int
	)

	responseHeaders := http.Header{}

	builder := c.baseRequestBuilder(`/api/protocols/nvme/subsystem-maps`, &statusCode, responseHeaders).
		BodyJSON(nvmeSubsystemMap)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}

func (c *Client) DeleteNVMeSubsystemMap(ctx context.Context, svmName string, subsystemName string, namespaceName string) error {
	var (
		statusCode int
		nvmeSs     ontap.GetData
		nvmeNs     ontap.GetData
	)

	responseHeaders := http.Header{}

	params := url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", subsystemName)

	builder := c.baseRequestBuilder(`/api/protocols/nvme/subsystems`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeSs)

	if err := c.buildAndExecuteRequest(ctx, builder); err != nil {
		return err
	}

	if nvmeSs.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme subsystem of name %s in svm %s because it does not exist", subsystemName, svmName)
	}

	if nvmeSs.NumRecords != 1 {
		return fmt.Errorf("failed to get NVMe subsystem %s in svm=%s because there are %d matching records",
			subsystemName, svmName, nvmeSs.NumRecords)
	}

	responseHeaders = http.Header{}
	params = url.Values{}
	params.Set("svm.name", svmName)
	params.Set("name", namespaceName)

	builder2 := c.baseRequestBuilder(`/api/storage/namespaces`, &statusCode, responseHeaders).
		Params(params).
		ToJSON(&nvmeNs)

	if err := c.buildAndExecuteRequest(ctx, builder2); err != nil {
		return err
	}

	if nvmeNs.NumRecords == 0 {
		return fmt.Errorf("failed to get detail of nvme namespace of name %s in svm %s because it does not exist", namespaceName, svmName)
	}

	if nvmeNs.NumRecords != 1 {
		return fmt.Errorf("failed to get NVMe namespace %s in svm=%s because there are %d matching records",
			namespaceName, svmName, nvmeNs.NumRecords)
	}

	builder3 := c.baseRequestBuilder(`/api/protocols/nvme/subsystem-maps/`+nvmeSs.Records[0].UUID+`/`+nvmeNs.Records[0].UUID, &statusCode, responseHeaders).
		Delete()

	if err := c.buildAndExecuteRequest(ctx, builder3); err != nil {
		return err
	}

	return c.checkStatus(statusCode)
}
