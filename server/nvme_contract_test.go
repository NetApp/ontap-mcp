package server

import (
	"maps"
	"net/http"
	"strings"
	"testing"
)

// TestNVMeContract mirrors integration/test/nvme_test.go: NVMe subsystems (with hosts),
// namespaces, and subsystem maps, plus the NVMe service tools. The integration suite runs in
// "multiplex" tool mode, so updates and deletes go through modify_nvme_*.
func TestNVMeContract(t *testing.T) {
	const (
		nsName     = "/vol/docns/ns1"
		host1      = "nqn.1992-01.example.com:host1"
		host2      = "nqn.1992-01.example.com:host2"
		host3      = "nqn.1992-01.example.com:host3"
		noneFound  = `{"num_records":0,"records":[]}`
		noRecords  = `{"num_records":1,"records":[]}` // count disagrees with the records array
		accepted   = `{"job":{"uuid":"job-1"}}`
		jobPath    = "/api/cluster/jobs/job-1"
		subsysPath = "/api/protocols/nvme/subsystems"
		nsPath     = "/api/storage/namespaces"
		mapPath    = "/api/protocols/nvme/subsystem-maps"
		svcPath    = "/api/protocols/nvme/services"
	)

	jobDone := exchange{method: http.MethodGet, path: jobPath, resp: `{"uuid":"job-1","state":"success"}`}

	// lookup is the name-to-UUID GET that precedes every subsystem or namespace update and
	// delete. An empty uuid means the object does not exist.
	lookup := func(collection, svm, name, uuid string) exchange {
		e := exchange{method: http.MethodGet, path: collection + "?svm.name=" + svm + "&name=" + name, resp: noneFound}
		if uuid != "" {
			e.resp = `{"num_records":1,"records":[{"uuid":"` + uuid + `","name":"` + name + `"}]}`
		}
		return e
	}
	lookupSubsys := func(svm, name, uuid string) exchange { return lookup(subsysPath, svm, name, uuid) }
	lookupNS := func(svm, uuid string) exchange { return lookup(nsPath, svm, nsName, uuid) }

	// lookupService finds the NVMe service by SVM; its UUID is the SVM's UUID.
	lookupService := func(svm, svmUUID string) exchange {
		e := exchange{method: http.MethodGet, path: svcPath + "?svm.name=" + svm, resp: noneFound}
		if svmUUID != "" {
			e.resp = `{"num_records":1,"records":[{"svm":{"name":"` + svm + `","uuid":"` + svmUUID + `"}}]}`
		}
		return e
	}

	ontapErr := func(message, code string) string {
		return `{"error":{"message":"` + message + `","code":"` + code + `"}}`
	}

	sys2 := func(extra map[string]any) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "svm_name": "marketing", "name": "sys2"}
		maps.Copy(args, extra)
		return args
	}
	ns := func(extra map[string]any) map[string]any {
		args := map[string]any{"cluster_name": contractCluster, "svm_name": "nvmevs1", "name": nsName}
		maps.Copy(args, extra)
		return args
	}
	subsysMap := map[string]any{"cluster_name": contractCluster, "svm_name": "nvmevs1", "subsystem_name": "sys1", "namespace_name": nsName}
	subsysMapBody := `{"svm":{"name":"nvmevs1"},"subsystem":{"name":"sys1"},"namespace":{"name":"/vol/docns/ns1"}}`

	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		ontap    []exchange
		wantText string
		wantErr  bool
	}{
		// Subsystem create: "Create NVMe subsystem" steps.
		{
			name: "create subsystem with hosts",
			tool: "create_nvme_subsystem",
			args: sys2(map[string]any{"os_type": "linux", "hosts_nqns": []string{host1, host2}}),
			ontap: []exchange{
				{method: http.MethodPost, path: subsysPath, status: http.StatusCreated,
					body: `{"svm":{"name":"marketing"},"name":"sys2","os_type":"linux","hosts":[{"nqn":"` + host1 + `"},{"nqn":"` + host2 + `"}]}`},
			},
			wantText: "NVMe Subsystem created successfully",
		},
		{
			name: "create subsystem without hosts",
			tool: "create_nvme_subsystem",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "nvmevs1", "name": "sys1", "os_type": "linux"},
			ontap: []exchange{
				{method: http.MethodPost, path: subsysPath, status: http.StatusCreated,
					body: `{"svm":{"name":"nvmevs1"},"name":"sys1","os_type":"linux"}`},
			},
			wantText: "NVMe Subsystem created successfully",
		},
		{
			name: "create subsystem surfaces ONTAP error",
			tool: "create_nvme_subsystem",
			args: sys2(map[string]any{"os_type": "linux"}),
			ontap: []exchange{
				{method: http.MethodPost, path: subsysPath, status: http.StatusConflict,
					body: `{"svm":{"name":"marketing"},"name":"sys2","os_type":"linux"}`,
					resp: ontapErr("Subsystem sys2 already exists.", "72090019")},
			},
			wantText: "message: Subsystem sys2 already exists. code: 72090019",
			wantErr:  true,
		},
		{
			name:     "create subsystem without os_type sends nothing",
			tool:     "create_nvme_subsystem",
			args:     sys2(nil),
			wantText: "OS type is required",
			wantErr:  true,
		},

		// Subsystem update: "Update NVMe subsystem".
		{
			name: "modify subsystem sets comment",
			tool: "modify_nvme_subsystem",
			args: sys2(map[string]any{"operation": "update", "nvme_subsystem_update": map[string]any{"comment": "comment about the"}}),
			ontap: []exchange{
				lookupSubsys("marketing", "sys2", "ss-uuid"),
				{method: http.MethodPatch, path: subsysPath + "/ss-uuid", body: `{"comment":"comment about the"}`},
			},
			wantText: "NVMe Subsystem updated successfully",
		},
		{
			name:     "modify subsystem update without fields sends nothing",
			tool:     "modify_nvme_subsystem",
			args:     sys2(map[string]any{"operation": "update"}),
			wantText: "no update fields provided; specify at least one of: comment",
			wantErr:  true,
		},
		{
			name:     "modify subsystem update of missing subsystem",
			tool:     "modify_nvme_subsystem",
			args:     sys2(map[string]any{"operation": "update", "nvme_subsystem_update": map[string]any{"comment": "c"}}),
			ontap:    []exchange{lookupSubsys("marketing", "sys2", "")},
			wantText: "failed to get detail of nvme subsystem of name sys2 in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify subsystem update with empty records is treated as missing",
			tool: "modify_nvme_subsystem",
			args: sys2(map[string]any{"operation": "update", "nvme_subsystem_update": map[string]any{"comment": "c"}}),
			ontap: []exchange{
				{method: http.MethodGet, path: subsysPath + "?svm.name=marketing&name=sys2", resp: noRecords},
			},
			wantText: "failed to get detail of nvme subsystem of name sys2 in svm marketing because it does not exist",
			wantErr:  true,
		},

		// Subsystem delete: "Clean NVMe subsystem" steps.
		{
			name: "modify subsystem delete with allow flags",
			tool: "modify_nvme_subsystem",
			args: sys2(map[string]any{"operation": "delete", "allow_delete_while_mapped": true, "allow_delete_with_hosts": true}),
			ontap: []exchange{
				lookupSubsys("marketing", "sys2", "ss-uuid"),
				{method: http.MethodDelete, path: subsysPath + "/ss-uuid?allow_delete_while_mapped=true&allow_delete_with_hosts=true"},
			},
			wantText: "NVMe Subsystem deleted successfully",
		},
		{
			name: "modify subsystem delete defaults allow flags to false",
			tool: "modify_nvme_subsystem",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "nvmevs1", "name": "sys1"},
			ontap: []exchange{
				lookupSubsys("nvmevs1", "sys1", "ss-uuid"),
				{method: http.MethodDelete, path: subsysPath + "/ss-uuid?allow_delete_while_mapped=false&allow_delete_with_hosts=false"},
			},
			wantText: "NVMe Subsystem deleted successfully",
		},
		{
			// The integration suite's "Clean NVMe subsystem" steps depend on this message.
			name:     "modify subsystem delete of missing subsystem",
			tool:     "modify_nvme_subsystem",
			args:     sys2(map[string]any{"operation": "delete", "allow_delete_while_mapped": true, "allow_delete_with_hosts": true}),
			ontap:    []exchange{lookupSubsys("marketing", "sys2", "")},
			wantText: "failed to get detail of nvme subsystem of name sys2 in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify subsystem delete with empty records is treated as missing",
			tool: "modify_nvme_subsystem",
			args: sys2(map[string]any{"operation": "delete"}),
			ontap: []exchange{
				{method: http.MethodGet, path: subsysPath + "?svm.name=marketing&name=sys2", resp: noRecords},
			},
			wantText: "failed to get detail of nvme subsystem of name sys2 in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify subsystem delete refuses ambiguous match",
			tool: "modify_nvme_subsystem",
			args: sys2(map[string]any{"operation": "delete"}),
			ontap: []exchange{
				{method: http.MethodGet, path: subsysPath + "?svm.name=marketing&name=sys2",
					resp: `{"num_records":2,"records":[{"uuid":"ss-1","name":"sys2"},{"uuid":"ss-2","name":"sys2"}]}`},
			},
			wantText: "failed to delete NVMe subsystem sys2 in svm=marketing because there are 2 matching records",
			wantErr:  true,
		},
		{
			name: "modify subsystem delete surfaces ONTAP error",
			tool: "modify_nvme_subsystem",
			args: sys2(map[string]any{"operation": "delete"}),
			ontap: []exchange{
				lookupSubsys("marketing", "sys2", "ss-uuid"),
				{method: http.MethodDelete, path: subsysPath + "/ss-uuid?allow_delete_while_mapped=false&allow_delete_with_hosts=false",
					status: http.StatusBadRequest, resp: ontapErr("The subsystem has hosts configured.", "72090036")},
			},
			wantText: "message: The subsystem has hosts configured. code: 72090036",
			wantErr:  true,
		},
		{
			name:     "modify subsystem rejects unknown operation",
			tool:     "modify_nvme_subsystem",
			args:     sys2(map[string]any{"operation": "rename"}),
			wantText: `unsupported operation "rename"`,
			wantErr:  true,
		},

		// Subsystem hosts: "Add host in NVMe subsystem" and "Remove host in NVMe subsystem".
		{
			name: "add one host",
			tool: "add_nvme_subsystem_host",
			args: sys2(map[string]any{"nqn": host3}),
			ontap: []exchange{
				lookupSubsys("marketing", "sys2", "ss-uuid"),
				{method: http.MethodPost, path: subsysPath + "/ss-uuid/hosts", status: http.StatusCreated, body: `{"nqn":"` + host3 + `"}`},
			},
			wantText: "NVMe Subsystem Host added successfully",
		},
		{
			name: "add several hosts",
			tool: "add_nvme_subsystem_host",
			args: sys2(map[string]any{"records_nqns": []string{host1, host2}}),
			ontap: []exchange{
				lookupSubsys("marketing", "sys2", "ss-uuid"),
				{method: http.MethodPost, path: subsysPath + "/ss-uuid/hosts", status: http.StatusCreated,
					body: `{"records":[{"nqn":"` + host1 + `"},{"nqn":"` + host2 + `"}]}`},
			},
			wantText: "NVMe Subsystem Host added successfully",
		},
		{
			name:     "add host rejects nqn and records together",
			tool:     "add_nvme_subsystem_host",
			args:     sys2(map[string]any{"nqn": host3, "records_nqns": []string{host1}}),
			wantText: "specify either a single NVMe subsystem host NQN or an array of NQNs, but not both",
			wantErr:  true,
		},
		{
			name:     "add host requires an nqn",
			tool:     "add_nvme_subsystem_host",
			args:     sys2(nil),
			wantText: "either NVMe subsystem host NQN OR one or more host NQNs (records) must be provided",
			wantErr:  true,
		},
		{
			name:     "add host to missing subsystem",
			tool:     "add_nvme_subsystem_host",
			args:     sys2(map[string]any{"nqn": host3}),
			ontap:    []exchange{lookupSubsys("marketing", "sys2", "")},
			wantText: "failed to get detail of nvme subsystem of name sys2 in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "add host with empty subsystem records is treated as missing",
			tool: "add_nvme_subsystem_host",
			args: sys2(map[string]any{"nqn": host3}),
			ontap: []exchange{
				{method: http.MethodGet, path: subsysPath + "?svm.name=marketing&name=sys2", resp: noRecords},
			},
			wantText: "failed to get detail of nvme subsystem of name sys2 in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "add host surfaces ONTAP error",
			tool: "add_nvme_subsystem_host",
			args: sys2(map[string]any{"nqn": host3}),
			ontap: []exchange{
				lookupSubsys("marketing", "sys2", "ss-uuid"),
				{method: http.MethodPost, path: subsysPath + "/ss-uuid/hosts", status: http.StatusConflict, body: `{"nqn":"` + host3 + `"}`,
					resp: ontapErr("Host already added to subsystem.", "72089705")},
			},
			wantText: "message: Host already added to subsystem. code: 72089705",
			wantErr:  true,
		},
		{
			name: "remove host",
			tool: "remove_nvme_subsystem_host",
			args: sys2(map[string]any{"nqn": host3}),
			ontap: []exchange{
				lookupSubsys("marketing", "sys2", "ss-uuid"),
				{method: http.MethodDelete, path: subsysPath + "/ss-uuid/hosts/" + host3},
			},
			wantText: "NVMe Subsystem Host removed successfully",
		},
		{
			name: "remove host surfaces ONTAP error",
			tool: "remove_nvme_subsystem_host",
			args: sys2(map[string]any{"nqn": host3}),
			ontap: []exchange{
				lookupSubsys("marketing", "sys2", "ss-uuid"),
				{method: http.MethodDelete, path: subsysPath + "/ss-uuid/hosts/" + host3, status: http.StatusNotFound,
					resp: ontapErr("Host not found in subsystem.", "72090021")},
			},
			wantText: "message: Host not found in subsystem. code: 72090021",
			wantErr:  true,
		},
		{
			name:     "remove host from missing subsystem",
			tool:     "remove_nvme_subsystem_host",
			args:     sys2(map[string]any{"nqn": host3}),
			ontap:    []exchange{lookupSubsys("marketing", "sys2", "")},
			wantText: "failed to get detail of nvme subsystem of name sys2 in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name: "remove host with empty subsystem records is treated as missing",
			tool: "remove_nvme_subsystem_host",
			args: sys2(map[string]any{"nqn": host3}),
			ontap: []exchange{
				{method: http.MethodGet, path: subsysPath + "?svm.name=marketing&name=sys2", resp: noRecords},
			},
			wantText: "failed to get detail of nvme subsystem of name sys2 in svm marketing because it does not exist",
			wantErr:  true,
		},
		{
			name:     "remove host requires an nqn",
			tool:     "remove_nvme_subsystem_host",
			args:     sys2(nil),
			wantText: "NQN is required",
			wantErr:  true,
		},

		// Namespace create: "Create NVMe namespace".
		{
			name: "create namespace waits for job",
			tool: "create_nvme_namespace",
			args: ns(map[string]any{"os_type": "linux", "space.size": "20mb"}),
			ontap: []exchange{
				{method: http.MethodPost, path: nsPath, status: http.StatusAccepted, resp: accepted,
					body: `{"svm":{"name":"nvmevs1"},"name":"/vol/docns/ns1","os_type":"linux","space":{"size":"20mb"}}`},
				jobDone,
			},
			wantText: "NVMe Namespace created successfully",
		},
		{
			name: "create namespace completes synchronously",
			tool: "create_nvme_namespace",
			args: ns(map[string]any{"os_type": "linux", "space.size": "20mb"}),
			ontap: []exchange{
				{method: http.MethodPost, path: nsPath, status: http.StatusCreated,
					body: `{"svm":{"name":"nvmevs1"},"name":"/vol/docns/ns1","os_type":"linux","space":{"size":"20mb"}}`},
			},
			wantText: "NVMe Namespace created successfully",
		},
		{
			name: "create namespace surfaces async job failure",
			tool: "create_nvme_namespace",
			args: ns(map[string]any{"os_type": "linux", "space.size": "20mb"}),
			ontap: []exchange{
				{method: http.MethodPost, path: nsPath, status: http.StatusAccepted, resp: accepted,
					body: `{"svm":{"name":"nvmevs1"},"name":"/vol/docns/ns1","os_type":"linux","space":{"size":"20mb"}}`},
				{method: http.MethodGet, path: jobPath,
					resp: `{"uuid":"job-1","state":"failure","error":{"message":"Volume docns does not exist","code":"72089674"}}`},
			},
			wantText: "job failed code=72089674 msg=Volume docns does not exist",
			wantErr:  true,
		},
		{
			name: "create namespace surfaces ONTAP error",
			tool: "create_nvme_namespace",
			args: ns(map[string]any{"os_type": "linux", "space.size": "20mb"}),
			ontap: []exchange{
				{method: http.MethodPost, path: nsPath, status: http.StatusConflict,
					body: `{"svm":{"name":"nvmevs1"},"name":"/vol/docns/ns1","os_type":"linux","space":{"size":"20mb"}}`,
					resp: ontapErr("Namespace already exists.", "72089636")},
			},
			wantText: "message: Namespace already exists. code: 72089636",
			wantErr:  true,
		},
		{
			name:     "create namespace without size sends nothing",
			tool:     "create_nvme_namespace",
			args:     ns(map[string]any{"os_type": "linux"}),
			wantText: "size of namespace is required",
			wantErr:  true,
		},

		// Namespace update: "Update NVMe namespace".
		{
			name: "modify namespace resizes by UUID",
			tool: "modify_nvme_namespace",
			args: ns(map[string]any{"operation": "update", "nvme_namespace_update": map[string]any{"space.size": "40mb"}}),
			ontap: []exchange{
				lookupNS("nvmevs1", "ns-uuid"),
				{method: http.MethodPatch, path: nsPath + "/ns-uuid", body: `{"space":{"size":"40mb"}}`},
			},
			wantText: "NVMe Namespace updated successfully",
		},
		{
			name:     "modify namespace update without size sends nothing",
			tool:     "modify_nvme_namespace",
			args:     ns(map[string]any{"operation": "update"}),
			wantText: "at least one supported update field must be provided; size is supported for update",
			wantErr:  true,
		},
		{
			name:     "modify namespace update of missing namespace",
			tool:     "modify_nvme_namespace",
			args:     ns(map[string]any{"operation": "update", "nvme_namespace_update": map[string]any{"space.size": "40mb"}}),
			ontap:    []exchange{lookupNS("nvmevs1", "")},
			wantText: "failed to get detail of nvme namespace of name /vol/docns/ns1 in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify namespace update with empty records is treated as missing",
			tool: "modify_nvme_namespace",
			args: ns(map[string]any{"operation": "update", "nvme_namespace_update": map[string]any{"space.size": "40mb"}}),
			ontap: []exchange{
				{method: http.MethodGet, path: nsPath + "?svm.name=nvmevs1&name=" + nsName, resp: noRecords},
			},
			wantText: "failed to get detail of nvme namespace of name /vol/docns/ns1 in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify namespace update surfaces ONTAP error",
			tool: "modify_nvme_namespace",
			args: ns(map[string]any{"operation": "update", "nvme_namespace_update": map[string]any{"space.size": "1mb"}}),
			ontap: []exchange{
				lookupNS("nvmevs1", "ns-uuid"),
				{method: http.MethodPatch, path: nsPath + "/ns-uuid", body: `{"space":{"size":"1mb"}}`, status: http.StatusBadRequest,
					resp: ontapErr("Reducing the size of a namespace is not supported.", "72090015")},
			},
			wantText: "message: Reducing the size of a namespace is not supported. code: 72090015",
			wantErr:  true,
		},

		// Namespace delete: "Clean NVMe namespace" steps.
		{
			name: "modify namespace delete",
			tool: "modify_nvme_namespace",
			args: ns(map[string]any{"operation": "delete"}),
			ontap: []exchange{
				lookupNS("nvmevs1", "ns-uuid"),
				{method: http.MethodDelete, path: nsPath + "/ns-uuid?allow_delete_while_mapped=false"},
			},
			wantText: "NVMe Namespace deleted successfully",
		},
		{
			name: "modify namespace delete while mapped",
			tool: "modify_nvme_namespace",
			args: ns(map[string]any{"operation": "delete", "allow_delete_while_mapped": true}),
			ontap: []exchange{
				lookupNS("nvmevs1", "ns-uuid"),
				{method: http.MethodDelete, path: nsPath + "/ns-uuid?allow_delete_while_mapped=true"},
			},
			wantText: "NVMe Namespace deleted successfully",
		},
		{
			// The integration suite's "Clean NVMe namespace" steps depend on this message.
			name:     "modify namespace delete of missing namespace",
			tool:     "modify_nvme_namespace",
			args:     ns(map[string]any{"operation": "delete"}),
			ontap:    []exchange{lookupNS("nvmevs1", "")},
			wantText: "failed to get detail of nvme namespace of name /vol/docns/ns1 in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify namespace delete with empty records is treated as missing",
			tool: "modify_nvme_namespace",
			args: ns(map[string]any{"operation": "delete"}),
			ontap: []exchange{
				{method: http.MethodGet, path: nsPath + "?svm.name=nvmevs1&name=" + nsName, resp: noRecords},
			},
			wantText: "failed to get detail of nvme namespace of name /vol/docns/ns1 in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify namespace delete surfaces ONTAP error",
			tool: "modify_nvme_namespace",
			args: ns(map[string]any{"operation": "delete"}),
			ontap: []exchange{
				lookupNS("nvmevs1", "ns-uuid"),
				{method: http.MethodDelete, path: nsPath + "/ns-uuid?allow_delete_while_mapped=false", status: http.StatusBadRequest,
					resp: ontapErr("The namespace is mapped.", "72090007")},
			},
			wantText: "message: The namespace is mapped. code: 72090007",
			wantErr:  true,
		},
		{
			name:     "modify namespace rejects unknown operation",
			tool:     "modify_nvme_namespace",
			args:     ns(map[string]any{"operation": "move"}),
			wantText: `unsupported operation "move"`,
			wantErr:  true,
		},

		// Subsystem map create: "Create NVMe subsystem map".
		{
			name: "create subsystem map",
			tool: "create_nvme_subsystem_map",
			args: subsysMap,
			ontap: []exchange{
				{method: http.MethodPost, path: mapPath, status: http.StatusCreated, body: subsysMapBody},
			},
			wantText: "NVMe Subsystem Map created successfully",
		},
		{
			name: "create subsystem map waits for job",
			tool: "create_nvme_subsystem_map",
			args: subsysMap,
			ontap: []exchange{
				{method: http.MethodPost, path: mapPath, status: http.StatusAccepted, resp: accepted, body: subsysMapBody},
				jobDone,
			},
			wantText: "NVMe Subsystem Map created successfully",
		},
		{
			name: "create subsystem map surfaces ONTAP error",
			tool: "create_nvme_subsystem_map",
			args: subsysMap,
			ontap: []exchange{
				{method: http.MethodPost, path: mapPath, status: http.StatusBadRequest, body: subsysMapBody,
					resp: ontapErr("The namespace is already mapped.", "72089771")},
			},
			wantText: "message: The namespace is already mapped. code: 72089771",
			wantErr:  true,
		},
		{
			name:     "create subsystem map requires namespace",
			tool:     "create_nvme_subsystem_map",
			args:     map[string]any{"cluster_name": contractCluster, "svm_name": "nvmevs1", "subsystem_name": "sys1", "namespace_name": ""},
			wantText: "NVMe namespace name is required",
			wantErr:  true,
		},

		// Subsystem map delete: "Clean NVMe subsystem map".
		{
			name: "delete subsystem map by subsystem and namespace UUID",
			tool: "delete_nvme_subsystem_map",
			args: subsysMap,
			ontap: []exchange{
				lookupSubsys("nvmevs1", "sys1", "ss-uuid"),
				lookupNS("nvmevs1", "ns-uuid"),
				{method: http.MethodDelete, path: mapPath + "/ss-uuid/ns-uuid"},
			},
			wantText: "NVMe Subsystem Map deleted successfully",
		},
		{
			name:     "delete subsystem map of missing subsystem",
			tool:     "delete_nvme_subsystem_map",
			args:     subsysMap,
			ontap:    []exchange{lookupSubsys("nvmevs1", "sys1", "")},
			wantText: "failed to get detail of nvme subsystem of name sys1 in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete subsystem map of missing namespace",
			tool: "delete_nvme_subsystem_map",
			args: subsysMap,
			ontap: []exchange{
				lookupSubsys("nvmevs1", "sys1", "ss-uuid"),
				lookupNS("nvmevs1", ""),
			},
			wantText: "failed to get detail of nvme namespace of name /vol/docns/ns1 in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete subsystem map with empty subsystem records is treated as missing",
			tool: "delete_nvme_subsystem_map",
			args: subsysMap,
			ontap: []exchange{
				{method: http.MethodGet, path: subsysPath + "?svm.name=nvmevs1&name=sys1", resp: noRecords},
			},
			wantText: "failed to get detail of nvme subsystem of name sys1 in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete subsystem map with empty namespace records is treated as missing",
			tool: "delete_nvme_subsystem_map",
			args: subsysMap,
			ontap: []exchange{
				lookupSubsys("nvmevs1", "sys1", "ss-uuid"),
				{method: http.MethodGet, path: nsPath + "?svm.name=nvmevs1&name=" + nsName, resp: noRecords},
			},
			wantText: "failed to get detail of nvme namespace of name /vol/docns/ns1 in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "delete subsystem map surfaces ONTAP error",
			tool: "delete_nvme_subsystem_map",
			args: subsysMap,
			ontap: []exchange{
				lookupSubsys("nvmevs1", "sys1", "ss-uuid"),
				lookupNS("nvmevs1", "ns-uuid"),
				{method: http.MethodDelete, path: mapPath + "/ss-uuid/ns-uuid", status: http.StatusNotFound,
					resp: ontapErr("entry doesn't exist", "4")},
			},
			wantText: "message: entry doesn't exist code: 4",
			wantErr:  true,
		},

		// NVMe service. The integration test does not run these, but they are registered NVMe tools.
		{
			name: "create nvme service",
			tool: "create_nvme_service",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "nvmevs1", "enabled": "true"},
			ontap: []exchange{
				{method: http.MethodPost, path: svcPath, status: http.StatusCreated, body: `{"svm":{"name":"nvmevs1"},"enabled":"true"}`},
			},
			wantText: "NVMe Service created successfully",
		},
		{
			name: "create nvme service surfaces ONTAP error",
			tool: "create_nvme_service",
			args: map[string]any{"cluster_name": contractCluster, "svm_name": "nvmevs1"},
			ontap: []exchange{
				{method: http.MethodPost, path: svcPath, status: http.StatusConflict, body: `{"svm":{"name":"nvmevs1"}}`,
					resp: ontapErr("NVMe service already exists.", "72089900")},
			},
			wantText: "message: NVMe service already exists. code: 72089900",
			wantErr:  true,
		},
		{
			name: "modify nvme service disables by SVM UUID",
			tool: "modify_nvme_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nvmevs1", "nvme_service_update": map[string]any{"enabled": "false"}},
			ontap: []exchange{
				lookupService("nvmevs1", "svm-uuid"),
				{method: http.MethodPatch, path: svcPath + "/svm-uuid", body: `{"enabled":"false"}`},
			},
			wantText: "NVMe Service updated successfully",
		},
		{
			name:     "modify nvme service update without fields sends nothing",
			tool:     "modify_nvme_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nvmevs1"},
			wantText: "at least one field must be provided for update",
			wantErr:  true,
		},
		{
			name:     "modify nvme service update with empty update object sends nothing",
			tool:     "modify_nvme_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nvmevs1", "nvme_service_update": map[string]any{}},
			wantText: "at least one field must be provided for update",
			wantErr:  true,
		},
		{
			name: "modify nvme service update with empty records is treated as missing",
			tool: "modify_nvme_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nvmevs1", "nvme_service_update": map[string]any{"enabled": "false"}},
			ontap: []exchange{
				{method: http.MethodGet, path: svcPath + "?svm.name=nvmevs1", resp: noRecords},
			},
			wantText: "failed to get detail of nvme service in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name: "modify nvme service delete",
			tool: "modify_nvme_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "nvmevs1"},
			ontap: []exchange{
				lookupService("nvmevs1", "svm-uuid"),
				{method: http.MethodDelete, path: svcPath + "/svm-uuid"},
			},
			wantText: "NVMe Service deleted successfully",
		},
		{
			name:     "modify nvme service delete of missing service",
			tool:     "modify_nvme_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "nvmevs1"},
			ontap:    []exchange{lookupService("nvmevs1", "")},
			wantText: "failed to get detail of nvme service in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name:     "modify nvme service update refuses ambiguous match",
			tool:     "modify_nvme_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "update", "svm_name": "nvmevs1", "nvme_service_update": map[string]any{"enabled": "false"}},
			ontap:    []exchange{{method: http.MethodGet, path: svcPath + "?svm.name=nvmevs1", resp: `{"num_records":2,"records":[{"svm":{"name":"nvmevs1","uuid":"a"}},{"svm":{"name":"nvmevs1","uuid":"b"}}]}`}},
			wantText: "failed to get nvme service on svm=nvmevs1 because there are 2 matching records",
			wantErr:  true,
		},
		{
			name:     "modify nvme service delete refuses ambiguous match",
			tool:     "modify_nvme_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "nvmevs1"},
			ontap:    []exchange{{method: http.MethodGet, path: svcPath + "?svm.name=nvmevs1", resp: `{"num_records":2,"records":[{"svm":{"name":"nvmevs1","uuid":"a"}},{"svm":{"name":"nvmevs1","uuid":"b"}}]}`}},
			wantText: "failed to get nvme service on svm=nvmevs1 because there are 2 matching records",
			wantErr:  true,
		},
		{
			name: "modify nvme service delete with empty records is treated as missing",
			tool: "modify_nvme_service",
			args: map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": "nvmevs1"},
			ontap: []exchange{
				{method: http.MethodGet, path: svcPath + "?svm.name=nvmevs1", resp: noRecords},
			},
			wantText: "failed to get detail of nvme service in svm nvmevs1 because it does not exist",
			wantErr:  true,
		},
		{
			name:     "modify nvme service requires SVM",
			tool:     "modify_nvme_service",
			args:     map[string]any{"cluster_name": contractCluster, "operation": "delete", "svm_name": ""},
			wantText: "SVM name is required",
			wantErr:  true,
		},

		{
			name:     "unknown cluster",
			tool:     "modify_nvme_subsystem",
			args:     map[string]any{"cluster_name": "nope", "operation": "delete", "svm_name": "marketing", "name": "sys2"},
			wantText: "cluster nope not found",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newContractSession(t, "multiplex", tt.ontap)
			text, isErr := callTool(t, cs, tt.tool, tt.args)
			if isErr != tt.wantErr {
				t.Errorf("isError = %v, want %v; text: %s", isErr, tt.wantErr, text)
			}
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("result %q does not contain %q", text, tt.wantText)
			}
		})
	}
}
