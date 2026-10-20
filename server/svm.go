package server

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/rest"
	"github.com/netapp/ontap-mcp/tool"
)

func (a *App) CreateSVM(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVMCreate) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	svmCreate, err := newCreateSVM(parameters)
	if err != nil {
		return errorResult(err), nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.CreateSVM(ctx, svmCreate)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SVM created successfully"},
		},
	}, nil, nil
}

func (a *App) UpdateSVM(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVM) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	svmUpdate, err := newUpdateSVM(parameters)
	if err != nil {
		return errorResult(err), nil, err
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.UpdateSVM(ctx, svmUpdate, parameters.Name)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SVM updated successfully"},
		},
	}, nil, nil
}

func (a *App) DeleteSVM(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVM) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.Name == "" {
		return nil, nil, errors.New("SVM name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	err = client.DeleteSVM(ctx, parameters.Name)
	if err != nil {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SVM deleted successfully"},
		},
	}, nil, nil
}

func newCreateSVM(in tool.SVMCreate) (ontap.SVMCreate, error) {
	out := ontap.SVMCreate{}
	if in.Name == "" {
		return out, errors.New("SVM name is required")
	}
	out.Name = in.Name
	return out, nil
}

func newUpdateSVM(in tool.SVM) (ontap.SVM, error) {
	out := ontap.SVM{}

	if in.Name == "" {
		return out, errors.New("SVM name is required")
	}

	hasUpdate := false
	if in.NewName != "" {
		out.Name = in.NewName
		hasUpdate = true
	}
	if in.Comment != "" {
		out.Comment = &in.Comment
		hasUpdate = true
	}
	if in.State != "" {
		out.State = in.State
		hasUpdate = true
	}
	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: new_name, comment, or state")
	}

	return out, nil
}

func (a *App) ModifySVM(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVMModify) (*mcp.CallToolResult, any, error) {
	if !a.locks.TryLock(parameters.Cluster) {
		return errorResult(fmt.Errorf("another write operation is in progress on cluster %s, please try again", parameters.Cluster)), nil, nil
	}
	defer a.locks.Unlock(parameters.Cluster)

	if parameters.Name == "" {
		return nil, nil, errors.New("SVM name is required")
	}

	client, err := a.getClient(parameters.Cluster)
	if err != nil {
		return errorResult(err), nil, err
	}

	switch parameters.Operation {
	case "update":
		svmUpdate, err := updateSVMValidation(parameters.SVMUpdate)
		if err != nil {
			return errorResult(err), nil, err
		}

		err = client.UpdateSVM(ctx, svmUpdate, parameters.Name)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "SVM updated successfully"},
			},
		}, nil, nil
	case "delete":
		err = client.DeleteSVM(ctx, parameters.Name)
		if err != nil {
			return errorResult(err), nil, err
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "SVM deleted successfully"},
			},
		}, nil, nil
	default:
		return errorResult(fmt.Errorf("unsupported operation %q; supported values: update, delete", parameters.Operation)), nil, nil
	}
}

func updateSVMValidation(in tool.SVMUpdate) (ontap.SVM, error) {
	out := ontap.SVM{}

	hasUpdate := false
	if in.NewName != "" {
		out.Name = in.NewName
		hasUpdate = true
	}
	// A nil comment means no change; a pointer to "" clears the comment.
	if in.Comment != nil {
		out.Comment = in.Comment
		hasUpdate = true
	}
	if in.State != "" {
		out.State = in.State
		hasUpdate = true
	}
	if !hasUpdate {
		return out, errors.New("at least one updatable field must be provided: new_name, comment, or state")
	}

	return out, nil
}

func validateSVMPeerParams(sourceCluster, sourceSVM, destinationCluster, destinationSVM string, applications []string) error {
	var supportedApplication = []string{"snapmirror", "file_copy", "lun_copy", "flexcache"}

	if sourceCluster == "" {
		return errors.New("source cluster name is required")
	}
	if sourceSVM == "" {
		return errors.New("source SVM name is required")
	}
	if destinationCluster == "" {
		return errors.New("destination cluster name is required")
	}
	if destinationSVM == "" {
		return errors.New("destination SVM name is required")
	}
	if len(applications) > 0 {
		for i := range applications {
			applications[i] = strings.ToLower(applications[i])
			if !slices.Contains(supportedApplication, applications[i]) {
				return fmt.Errorf("application %q found, supported application are only %v", applications[i], supportedApplication)
			}
		}
	}
	return nil
}

func (a *App) lockSVMPeerClusters(sourceCluster, destinationCluster string) (func(), error) {
	clusters := []string{sourceCluster}
	if !strings.EqualFold(sourceCluster, destinationCluster) {
		clusters = append(clusters, destinationCluster)
	}
	sort.Slice(clusters, func(i, j int) bool {
		return strings.ToLower(clusters[i]) < strings.ToLower(clusters[j])
	})

	locked := make([]string, 0, len(clusters))
	for _, cluster := range clusters {
		if !a.locks.TryLock(cluster) {
			for _, val := range slices.Backward(locked) {
				a.locks.Unlock(val)
			}
			return nil, fmt.Errorf("another write operation is in progress on cluster %s, please try again", cluster)
		}
		locked = append(locked, cluster)
	}

	return func() {
		for _, val := range slices.Backward(locked) {
			a.locks.Unlock(val)
		}
	}, nil
}

func (a *App) CreateSVMPeer(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVMPeerCreate) (*mcp.CallToolResult, any, error) {
	parameters.SourceCluster = strings.TrimSpace(parameters.SourceCluster)
	parameters.SourceSVM = strings.TrimSpace(parameters.SourceSVM)
	parameters.DestinationCluster = strings.TrimSpace(parameters.DestinationCluster)
	parameters.DestinationSVM = strings.TrimSpace(parameters.DestinationSVM)
	if err := validateSVMPeerParams(parameters.SourceCluster, parameters.SourceSVM, parameters.DestinationCluster, parameters.DestinationSVM, parameters.Application); err != nil {
		return errorResult(err), nil, err
	}

	unlock, err := a.lockSVMPeerClusters(parameters.SourceCluster, parameters.DestinationCluster)
	if err != nil {
		return errorResult(err), nil, nil
	}
	defer unlock()

	sourceClient, err := a.getClient(parameters.SourceCluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	destinationClient := sourceClient
	if !strings.EqualFold(parameters.SourceCluster, parameters.DestinationCluster) {
		destinationClient, err = a.getClient(parameters.DestinationCluster)
		if err != nil {
			return errorResult(err), nil, err
		}
	}

	sourceInfo, err := sourceClient.GetClusterInfo(ctx)
	if err != nil {
		return errorResult(err), nil, err
	}
	destinationInfo := sourceInfo
	if destinationClient != sourceClient {
		destinationInfo, err = destinationClient.GetClusterInfo(ctx)
		if err != nil {
			return errorResult(err), nil, err
		}
	}

	applications := parameters.Application
	if len(applications) == 0 {
		applications = []string{"snapmirror"}
	}

	if parameters.AcceptOnly {
		err = acceptSVMPeer(ctx, destinationClient, parameters.DestinationSVM, parameters.SourceSVM, sourceInfo.Name, false)
	} else {
		lookupCluster := destinationInfo.Name
		payloadCluster := destinationInfo.Name
		if destinationClient == sourceClient {
			payloadCluster = ""
		}
		peer, findErr := sourceClient.FindSVMPeer(ctx, parameters.SourceSVM, parameters.DestinationSVM, lookupCluster)
		switch {
		case findErr == nil:
			for _, application := range peer.Applications {
				if !slices.Contains(applications, application) {
					applications = append(applications, application)
				}
			}
			err = sourceClient.UpdateSVMPeer(ctx, peer.UUID, ontap.SVMPeer{Applications: applications})
		case errors.Is(findErr, rest.ErrSVMPeerNotFound):
			err = sourceClient.CreateSVMPeer(ctx, ontap.SVMPeer{
				SVM:          ontap.NameAndUUID{Name: parameters.SourceSVM},
				Peer:         ontap.SVMPeerRemote{Cluster: ontap.NameAndUUID{Name: payloadCluster}, SVM: ontap.NameAndUUID{Name: parameters.DestinationSVM}},
				Applications: applications,
			})
		default:
			err = findErr
		}
		if err == nil && destinationClient != sourceClient {
			err = acceptSVMPeer(ctx, destinationClient, parameters.DestinationSVM, parameters.SourceSVM, sourceInfo.Name, true)
		}
	}
	if err != nil {
		return errorResult(err), nil, err
	}

	message := "SVM peer created or updated successfully"
	if parameters.AcceptOnly {
		message = "SVM peer accepted successfully"
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: message}}}, nil, nil
}

func acceptSVMPeer(ctx context.Context, client *rest.Client, localSVM, remoteSVM, remoteCluster string, wait bool) error {
	peer, err := client.FindSVMPeer(ctx, localSVM, remoteSVM, remoteCluster)
	// polling until peer state would be pending or peered or deadlined exceed.
	if wait && (errors.Is(err, rest.ErrSVMPeerNotFound) || (peer.State != "" && peer.State != "pending" && peer.State != "peered")) {
		waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for errors.Is(err, rest.ErrSVMPeerNotFound) || (peer.State != "" && peer.State != "pending" && peer.State != "peered") {
			select {
			case <-waitCtx.Done():
				return err
			case <-ticker.C:
				peer, err = client.FindSVMPeer(waitCtx, localSVM, remoteSVM, remoteCluster)
			}
		}
	}
	if err != nil {
		return err
	}

	switch peer.State {
	case "pending":
		return client.UpdateSVMPeer(ctx, peer.UUID, ontap.SVMPeer{State: "peered"})
	case "peered":
		return client.UpdateSVMPeer(ctx, peer.UUID, ontap.SVMPeer{})
	default:
		return fmt.Errorf("timed out after 30s waiting for SVM peer proposal, found peer state %s", peer.State)
	}
}

func (a *App) DeleteSVMPeer(ctx context.Context, _ *mcp.CallToolRequest, parameters tool.SVMPeerDelete) (*mcp.CallToolResult, any, error) {
	parameters.SourceCluster = strings.TrimSpace(parameters.SourceCluster)
	parameters.SourceSVM = strings.TrimSpace(parameters.SourceSVM)
	parameters.DestinationCluster = strings.TrimSpace(parameters.DestinationCluster)
	parameters.DestinationSVM = strings.TrimSpace(parameters.DestinationSVM)
	if err := validateSVMPeerParams(parameters.SourceCluster, parameters.SourceSVM, parameters.DestinationCluster, parameters.DestinationSVM, []string{}); err != nil {
		return errorResult(err), nil, err
	}

	unlock, err := a.lockSVMPeerClusters(parameters.SourceCluster, parameters.DestinationCluster)
	if err != nil {
		return errorResult(err), nil, nil
	}
	defer unlock()

	sourceClient, err := a.getClient(parameters.SourceCluster)
	if err != nil {
		return errorResult(err), nil, err
	}
	destinationClient := sourceClient
	if !strings.EqualFold(parameters.SourceCluster, parameters.DestinationCluster) {
		destinationClient, err = a.getClient(parameters.DestinationCluster)
		if err != nil {
			return errorResult(err), nil, err
		}
	}

	sourceInfo, err := sourceClient.GetClusterInfo(ctx)
	if err != nil {
		return errorResult(err), nil, err
	}
	destinationInfo := sourceInfo
	if destinationClient != sourceClient {
		destinationInfo, err = destinationClient.GetClusterInfo(ctx)
		if err != nil {
			return errorResult(err), nil, err
		}
	}

	remoteCluster := destinationInfo.Name
	if destinationClient == sourceClient {
		remoteCluster = ""
	}
	sourcePeer, sourceErr := sourceClient.FindSVMPeer(ctx, parameters.SourceSVM, parameters.DestinationSVM, remoteCluster)
	if sourceErr != nil && !errors.Is(sourceErr, rest.ErrSVMPeerNotFound) {
		return errorResult(sourceErr), nil, sourceErr
	}
	destinationPeerDeleted := false
	if destinationClient == sourceClient {
		if sourceErr != nil {
			return errorResult(sourceErr), nil, sourceErr
		}
		err = sourceClient.DeleteSVMPeer(ctx, sourcePeer.UUID)
	} else {
		found := sourceErr == nil
		var deleteErrors []error
		if sourceErr == nil {
			if deleteErr := sourceClient.DeleteSVMPeer(ctx, sourcePeer.UUID); deleteErr != nil {
				deleteErrors = append(deleteErrors, deleteErr)
			}
		}

		// only delete from destination when source peer has not found or delete failed in source
		if sourceErr != nil || len(deleteErrors) > 0 {
			destinationPeer, destinationErr := destinationClient.FindSVMPeer(ctx, parameters.DestinationSVM, parameters.SourceSVM, sourceInfo.Name)
			if destinationErr == nil {
				found = true
				if deleteErr := destinationClient.DeleteSVMPeer(ctx, destinationPeer.UUID); deleteErr != nil {
					deleteErrors = append(deleteErrors, deleteErr)
				} else {
					destinationPeerDeleted = true
				}
			} else if !errors.Is(destinationErr, rest.ErrSVMPeerNotFound) {
				deleteErrors = append(deleteErrors, destinationErr)
			}
			if !found && len(deleteErrors) == 0 {
				err = fmt.Errorf("%w between %s and %s", rest.ErrSVMPeerNotFound, parameters.SourceSVM, parameters.DestinationSVM)
			} else {
				err = errors.Join(deleteErrors...)
			}
		}
	}

	if err != nil && !destinationPeerDeleted {
		return errorResult(err), nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "SVM peer deleted successfully"},
		},
	}, nil, nil
}
