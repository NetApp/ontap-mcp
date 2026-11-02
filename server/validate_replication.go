package server

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/netapp/ontap-mcp/ontap"
	"github.com/netapp/ontap-mcp/rest"
	"github.com/netapp/ontap-mcp/tool"
)

type ReplicationCheck struct {
	Name       string `json:"name"`
	Scope      string `json:"scope"`
	Passed     bool   `json:"passed"`
	Verifiable bool   `json:"verifiable"`
	Message    string `json:"message"`
}

type ValidateReplicationResponse struct {
	Message              string             `json:"message"`
	CanReplicate         bool               `json:"can_replicate"`
	SourceVersion        string             `json:"source_version,omitempty"`
	DestinationVersion   string             `json:"destination_version,omitempty"`
	Checks               []ReplicationCheck `json:"checks"`
	MissingPrerequisites []string           `json:"missing_prerequisites"`
}

type validateReplicationArgs struct {
	sourceCluster      string
	destinationCluster string
	sourceSVM          string
	destinationSVM     string
}

func newValidateReplication(p tool.ValidateReplicationParams) (validateReplicationArgs, error) {
	args := validateReplicationArgs{
		sourceCluster:      strings.TrimSpace(p.SourceCluster),
		destinationCluster: strings.TrimSpace(p.DestinationCluster),
		sourceSVM:          strings.TrimSpace(p.SourceSVM),
		destinationSVM:     strings.TrimSpace(p.DestinationSVM),
	}
	if args.sourceCluster == "" {
		return validateReplicationArgs{}, errors.New("source_cluster is required")
	}
	if args.destinationCluster == "" {
		return validateReplicationArgs{}, errors.New("destination_cluster is required")
	}
	if strings.EqualFold(args.sourceCluster, args.destinationCluster) {
		return validateReplicationArgs{}, errors.New("source_cluster and destination_cluster must be different")
	}
	if (args.sourceSVM == "") != (args.destinationSVM == "") {
		return validateReplicationArgs{}, errors.New("source_svm and destination_svm must be provided together")
	}
	return args, nil
}

func (a *App) ValidateReplication(ctx context.Context, _ *mcp.CallToolRequest, p tool.ValidateReplicationParams) (*mcp.CallToolResult, ValidateReplicationResponse, error) {
	empty := ValidateReplicationResponse{}
	args, err := newValidateReplication(p)
	if err != nil {
		return errorResult(err), empty, nil
	}

	sourceCanonical, ok := a.resolveCluster(args.sourceCluster)
	if !ok {
		err := fmt.Errorf("cluster %s not found", args.sourceCluster)
		return errorResult(err), empty, err
	}
	destinationCanonical, ok := a.resolveCluster(args.destinationCluster)
	if !ok {
		err := fmt.Errorf("cluster %s not found", args.destinationCluster)
		return errorResult(err), empty, err
	}
	if sourceCanonical == destinationCanonical {
		err := errors.New("source_cluster and destination_cluster must be different")
		return errorResult(err), empty, err
	}

	lockedClusters := []string{sourceCanonical, destinationCanonical}
	sort.Strings(lockedClusters)
	sourceClient, err := a.getClient(sourceCanonical)
	if err != nil {
		return errorResult(err), empty, err
	}
	destinationClient, err := a.getClient(destinationCanonical)
	if err != nil {
		return errorResult(err), empty, err
	}

	for _, cluster := range lockedClusters {
		a.locks.RLock(cluster)
	}
	defer func() {
		for _, lockedCluster := range slices.Backward(lockedClusters) {
			a.locks.RUnlock(lockedCluster)
		}
	}()

	response := ValidateReplicationResponse{
		Checks:               make([]ReplicationCheck, 0, 9),
		MissingPrerequisites: []string{},
	}
	addCheck := func(check ReplicationCheck) {
		response.Checks = append(response.Checks, check)
		if !check.Passed {
			response.MissingPrerequisites = append(response.MissingPrerequisites, check.Message)
		}
	}

	sourceLIFs, sourceLIFErr := sourceClient.GetInterclusterLIFs(ctx)
	addCheck(interclusterLIFCheck("source", sourceCanonical, sourceLIFs, sourceLIFErr))
	destinationLIFs, destinationLIFErr := destinationClient.GetInterclusterLIFs(ctx)
	addCheck(interclusterLIFCheck("destination", destinationCanonical, destinationLIFs, destinationLIFErr))

	sourceRemote, sourceInfoErr := sourceClient.GetClusterInfo(ctx)
	addCheck(clusterInfoCheck("source", sourceCanonical, sourceRemote, sourceInfoErr))
	destinationRemote, destinationInfoErr := destinationClient.GetClusterInfo(ctx)
	addCheck(clusterInfoCheck("destination", destinationCanonical, destinationRemote, destinationInfoErr))

	sourcePeerName := destinationRemote.Name
	if sourcePeerName == "" {
		sourcePeerName = destinationCanonical
	}
	destinationPeerName := sourceRemote.Name
	if destinationPeerName == "" {
		destinationPeerName = sourceCanonical
	}
	sourcePeers, sourcePeerErr := sourceClient.GetClusterPeers(ctx, sourcePeerName)
	addCheck(clusterPeerCheck("source", sourceCanonical, sourcePeerName, sourcePeers, sourcePeerErr))
	destinationPeers, destinationPeerErr := destinationClient.GetClusterPeers(ctx, destinationPeerName)
	addCheck(clusterPeerCheck("destination", destinationCanonical, destinationPeerName, destinationPeers, destinationPeerErr))

	if args.sourceSVM != "" {
		sourceSVMPeers, sourceSVMPeerErr := sourceClient.GetSVMPeers(ctx, args.sourceSVM, sourcePeerName, args.destinationSVM)
		addCheck(svmPeerCheck("source", sourceCanonical, args.sourceSVM, args.destinationSVM, sourceSVMPeers, sourceSVMPeerErr))
		destinationSVMPeers, destinationSVMPeerErr := destinationClient.GetSVMPeers(ctx, args.destinationSVM, destinationPeerName, args.sourceSVM)
		addCheck(svmPeerCheck("destination", destinationCanonical, args.destinationSVM, args.sourceSVM, destinationSVMPeers, destinationSVMPeerErr))
	}

	if sourceInfoErr != nil || destinationInfoErr != nil {
		addCheck(ReplicationCheck{
			Name:       "version_compatibility",
			Scope:      "source_to_destination",
			Verifiable: false,
			Message:    "asynchronous XDP version compatibility could not be verified because cluster version information is unavailable",
		})
	} else {
		compatibility := checkXDPCompatibility(sourceRemote, destinationRemote)
		response.SourceVersion = compatibility.SourceRelease
		response.DestinationVersion = compatibility.DestinationRelease
		addCheck(ReplicationCheck{
			Name:       "version_compatibility",
			Scope:      "source_to_destination",
			Passed:     compatibility.Compatible,
			Verifiable: compatibility.Known,
			Message:    compatibility.Reason,
		})
	}

	response.CanReplicate = len(response.MissingPrerequisites) == 0
	if response.CanReplicate {
		response.Message = "All prerequisites are satisfied; intercluster replication is possible."
	} else {
		response.Message = "Intercluster replication prerequisites are not satisfied: " + strings.Join(response.MissingPrerequisites, "; ")
	}
	return nil, response, nil
}

func clusterInfoCheck(scope, cluster string, remote ontap.Remote, err error) ReplicationCheck {
	check := ReplicationCheck{Name: "cluster_info", Scope: scope, Verifiable: err == nil}
	if err != nil {
		check.Message = fmt.Sprintf("could not query cluster information for %s cluster %s: %v", scope, cluster, err)
		return check
	}
	check.Passed = true
	check.Message = fmt.Sprintf("%s cluster identity is %s", scope, remote.Name)
	return check
}

func interclusterLIFCheck(scope, cluster string, lifs []rest.InterclusterLIF, err error) ReplicationCheck {
	check := ReplicationCheck{Name: "intercluster_lifs", Scope: scope, Verifiable: err == nil}
	if err != nil {
		check.Message = fmt.Sprintf("could not query intercluster LIFs on %s cluster %s: %v", scope, cluster, err)
		return check
	}
	if len(lifs) == 0 {
		check.Message = fmt.Sprintf("%s cluster %s has no intercluster LIFs", scope, cluster)
		return check
	}
	var unavailable []string
	for _, lif := range lifs {
		if !lif.Enabled || !strings.EqualFold(lif.State, "up") {
			unavailable = append(unavailable, fmt.Sprintf("%s (enabled=%t, state=%s)", lif.Name, lif.Enabled, lif.State))
		}
	}
	if len(unavailable) > 0 {
		check.Message = fmt.Sprintf("%s cluster %s has intercluster LIFs that are not enabled and up: %s", scope, cluster, strings.Join(unavailable, ", "))
		return check
	}
	check.Passed = true
	check.Message = fmt.Sprintf("%s cluster %s has %d enabled and up intercluster LIF(s)", scope, cluster, len(lifs))
	return check
}

func clusterPeerCheck(scope, cluster, peerName string, peers []rest.ClusterPeer, err error) ReplicationCheck {
	check := ReplicationCheck{Name: "cluster_peer", Scope: scope, Verifiable: err == nil}
	if err != nil {
		check.Message = fmt.Sprintf("could not query cluster peer %s from %s cluster %s: %v", peerName, scope, cluster, err)
		return check
	}
	if len(peers) == 0 {
		check.Message = fmt.Sprintf("%s cluster %s has no cluster peer relationship with %s", scope, cluster, peerName)
		return check
	}
	if len(peers) != 1 {
		check.Message = fmt.Sprintf("%s cluster %s returned %d cluster peer relationships for %s", scope, cluster, len(peers), peerName)
		return check
	}
	if !strings.EqualFold(peers[0].Status.State, "available") {
		check.Message = fmt.Sprintf("cluster peer %s on %s cluster %s is not available (state=%s)", peerName, scope, cluster, peers[0].Status.State)
		return check
	}
	check.Passed = true
	check.Message = fmt.Sprintf("cluster peer %s is available on %s cluster %s", peerName, scope, cluster)
	return check
}

func svmPeerCheck(scope, cluster, localSVM, peerSVM string, peers []rest.SVMPeer, err error) ReplicationCheck {
	check := ReplicationCheck{Name: "svm_peer", Scope: scope, Verifiable: err == nil}
	if err != nil {
		check.Message = fmt.Sprintf("could not query SVM peer %s to %s on %s cluster %s: %v", localSVM, peerSVM, scope, cluster, err)
		return check
	}
	if len(peers) == 0 {
		check.Message = fmt.Sprintf("%s cluster %s has no SVM peer relationship from %s to %s", scope, cluster, localSVM, peerSVM)
		return check
	}
	if len(peers) != 1 {
		check.Message = fmt.Sprintf("%s cluster %s returned %d SVM peer relationships from %s to %s", scope, cluster, len(peers), localSVM, peerSVM)
		return check
	}
	if !strings.EqualFold(peers[0].State, "peered") {
		check.Message = fmt.Sprintf("SVM peer %s to %s on %s cluster %s is not peered (state=%s)", localSVM, peerSVM, scope, cluster, peers[0].State)
		return check
	}
	check.Passed = true
	check.Message = fmt.Sprintf("SVM peer %s to %s is peered on %s cluster %s", localSVM, peerSVM, scope, cluster)
	return check
}
