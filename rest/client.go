package rest

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // using sha1 for a hash, not a security risk
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/netapp/ontap-mcp/version"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/carlmjohnson/requests"
	"github.com/netapp/ontap-mcp/config"
	"github.com/netapp/ontap-mcp/ontap"
)

type Client struct {
	poller     *config.Poller
	httpClient *http.Client
	credCache  credentialsCache
	remote     ontap.Remote
}

// credentials holds authentication information
type credentials struct {
	Username  string
	Password  string
	AuthToken string
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

func (c *Client) handleJob(ctx context.Context, statusCode int, buf bytes.Buffer) error {
	if statusCode == http.StatusCreated || statusCode == http.StatusAccepted {
		var pj ontap.PostJob
		err := json.Unmarshal(buf.Bytes(), &pj)
		if err != nil {
			return err
		}

		err = c.waitForJob(ctx, `/api/cluster/jobs/`+pj.Job.UUID, 3*time.Minute)
		if err != nil {
			return err
		}
	}

	return nil
}

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

//nolint:unparam
func (c *Client) waitForJob(ctx context.Context, jobLocation string, duration time.Duration) error {
	var jr ontap.JobResponse

	// Poll every pollInterval seconds, up to duration
	const pollInterval = 2 * time.Second
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	builder := c.baseRequestBuilder(jobLocation, nil, nil).
		ToJSON(&jr)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	// If the job state is success or failure, return
	// otherwise keep trying
	// queued, running, paused
	handleJob := func(jobResponse ontap.JobResponse) (bool, error) {
		switch jobResponse.State {
		case "success":
			return true, nil
		case "failure":
			if jobResponse.Error != nil {
				return true, fmt.Errorf("job failed code=%s msg=%s", jobResponse.Error.Code, jobResponse.Error.Message)
			}
			return true, fmt.Errorf("job failed code=%d msg=%s", jobResponse.Code, jobResponse.Message)
		}
		return false, nil
	}

	done, err := handleJob(jr)
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	for {
		select {
		case <-ticker.C:
			err = c.buildAndExecuteRequest(ctx, builder)
			if err != nil {
				return err
			}
			done2, err2 := handleJob(jr)
			if err2 != nil {
				return err2
			}
			if done2 {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func ontapValidator(response *http.Response) error {
	if response.StatusCode >= http.StatusBadRequest {
		var ontapErr ontap.ClusterError
		err := requests.ToJSON(&ontapErr)(response)
		if err != nil {
			return err
		}
		ontapErr.StatusCode = response.StatusCode
		return ontapErr
	}
	return nil
}

func (c *Client) newClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.poller.UseInsecureTLS, //nolint:gosec
		},
	}
	aClient := &http.Client{
		Transport: transport,
		Timeout:   2 * time.Minute,
	}

	return aClient
}

func New(p *config.Poller) *Client {
	return &Client{
		poller:     p,
		httpClient: nil,
	}
}

// NewWithClient creates a new Client with a custom HTTP client for testing
func NewWithClient(p *config.Poller, aClient *http.Client) *Client {
	return &Client{
		poller:     p,
		httpClient: aClient,
	}
}

// getHTTPClient returns the custom client if set, otherwise creates a new default client
func (c *Client) getHTTPClient() *http.Client {
	var err error

	if c.httpClient == nil {
		client := c.newClient()
		c.httpClient = client
		c.remote, err = c.GetClusterInfo()
		if err == nil {
			err = c.sendMcpVersion()
			if err != nil {
				slog.Error("failed to send mcp version", slog.Any("error", err))
			}
		}
	}

	return c.httpClient
}

func (c *Client) GetClusterInfo() (ontap.Remote, error) {
	var cluster ontap.Cluster

	builder := c.baseRequestBuilder("/api/cluster?fields=*", nil, nil).
		ToJSON(&cluster)

	ctx := context.Background()
	err := c.buildAndExecuteRequest(ctx, builder)
	if err != nil {
		return ontap.Remote{}, err
	}

	r := ontap.Remote{
		Name:            cluster.Name,
		UUID:            cluster.UUID,
		Version:         cluster.Version,
		IsSanOptimized:  cluster.SanOptimized,
		IsDisaggregated: cluster.Disaggregated,
		IsClustered:     true,
		HasREST:         true,
		Model:           ontap.CDOT,
	}

	if r.IsDisaggregated && r.IsSanOptimized {
		r.Model = ontap.ASAr2
	}

	return r, nil
}

// baseRequestBuilder creates a request builder with common configuration:
// - Base URL with poller address
// - HTTP client
// - Response headers copying
// - Status code validator
// - ONTAP error validator
func (c *Client) baseRequestBuilder(endpoint string, statusCode *int, responseHeaders http.Header) *requests.Builder {
	aClient := c.getHTTPClient()
	builder := requests.
		URL(`https://` + c.poller.Addr + endpoint).
		Client(aClient)

	if responseHeaders != nil {
		builder = builder.CopyHeaders(responseHeaders)
	}

	return builder.
		AddValidator(func(response *http.Response) error {
			if statusCode != nil {
				*statusCode = response.StatusCode
			}
			return nil
		}).
		UserAgent("ontap-mcp/" + version.Info()).
		AddValidator(ontapValidator)
}

// buildAndExecuteRequest is a helper method that handles the common pattern of:
// 1. Getting authentication credentials
// 2. Building a request with authentication
// 3. Executing the request
func (c *Client) buildAndExecuteRequest(ctx context.Context, builder *requests.Builder) error {
	creds, err := c.getAuth(ctx)
	if err != nil {
		return err
	}

	if creds.AuthToken != "" {
		builder = builder.Bearer(creds.AuthToken)
	} else {
		builder = builder.BasicAuth(creds.Username, creds.Password)
	}

	return builder.Fetch(ctx)
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

// getAuth returns the appropriate authentication credentials
// Priority: credentials_script > credentials_file > inline config
func (c *Client) getAuth(ctx context.Context) (credentials, error) {
	// If credentials_script is configured, use it
	if c.poller.CredentialsScript.Path != "" {
		return c.getScriptCredentials(ctx)
	}

	// If credentials_file is configured, use it
	if c.poller.CredentialsFile != "" {
		return c.getFileCredentials()
	}

	// Use inline config credentials
	return credentials{
		Username:  c.poller.Username,
		Password:  c.poller.Password,
		AuthToken: "",
	}, nil
}

// getFileCredentials fetches credentials from the configured file
func (c *Client) getFileCredentials() (credentials, error) {
	username, password, err := loadCredentialsFile(c.poller.CredentialsFile, c.poller.Name)
	if err != nil {
		return credentials{}, err
	}

	// If username is not in the file, use the one from the main config
	if username == "" && c.poller.Username != "" {
		username = c.poller.Username
	}

	return credentials{
		Username:  username,
		Password:  password,
		AuthToken: "",
	}, nil
}

// getScriptCredentials fetches credentials from the configured script
func (c *Client) getScriptCredentials(ctx context.Context) (credentials, error) {
	schedule := parseSchedule(c.poller.CredentialsScript.Schedule)

	// Check if we need to refresh credentials
	if !c.credCache.shouldRefreshCredentials(schedule) {
		return c.credCache.getCredentials(), nil
	}

	// Execute the script to get new credentials
	response, err := executeCredentialsScript(ctx, c.poller)
	if err != nil {
		return credentials{}, err
	}

	// Determine username: use script response if provided, otherwise use config
	username := response.Username
	if username == "" && c.poller.Username != "" {
		username = c.poller.Username
	}

	// Update cache
	c.credCache.updateCache(username, response.Password, response.AuthToken)

	return credentials{
		Username:  username,
		Password:  response.Password,
		AuthToken: response.AuthToken,
	}, nil
}

func (c *Client) sendMcpVersion() error {
	if !c.remote.HasREST {
		return nil
	}

	// If the cluster is running ONTAP 9.11.1 or later,
	// send an ontapmcpTag to the cluster to indicate that the ONTAP MCP is running.
	// Otherwise, do nothing

	if c.remote.Version.Generation < 9 || c.remote.Version.Major < 11 || c.remote.Version.Minor < 1 {
		return nil
	}

	// Send the ontapMcpTag to the ONTAP cluster including the OS name, sha1(hostname), and ONTAP-MCP version
	osName := runtime.GOOS
	hostname, _ := os.Hostname()
	sha1Hostname := sha1Sum(hostname)

	fields := []string{osName, sha1Hostname, version.VERSION}

	u := `/api/cluster?ignore_unknown_fields=true&fields=` + "ontapMcpTag," + strings.Join(fields, ",")

	var statusCode int
	builder := c.baseRequestBuilder(u, &statusCode, nil)
	err := c.buildAndExecuteRequest(context.Background(), builder)

	if err != nil {
		return err
	}

	return nil
}

func sha1Sum(s string) string {
	hash := sha1.New() //nolint:gosec // using sha1 for a hash, not a security risk
	hash.Write([]byte(s))
	return hex.EncodeToString(hash.Sum(nil))
}
