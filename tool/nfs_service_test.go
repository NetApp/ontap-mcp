package tool

import (
	"reflect"
	"strings"
	"testing"
)

// TestNFSServiceAndUpdateJSONTagsMatch guards the two tool structs that share
// nfsFieldsFromService and nfsFieldsFromUpdate. jsonschema-go flattens embedded
// structs, but these descriptions are not the same: create records ONTAP's
// create defaults, and update must not. Compare the JSON names instead.
func TestNFSServiceAndUpdateJSONTagsMatch(t *testing.T) {
	create := jsonTags(t, reflect.TypeFor[NFSService]())
	update := jsonTags(t, reflect.TypeFor[NFSServiceUpdate]())
	delete(create, "cluster_name")
	delete(create, "svm_name")

	if len(create) != len(update) {
		t.Errorf("shared field count: NFSService %d, NFSServiceUpdate %d", len(create), len(update))
	}
	for name, tag := range create {
		if update[name] != tag {
			t.Errorf("json tag %s: NFSService %q, NFSServiceUpdate %q", name, tag, update[name])
		}
	}
	for name, tag := range update {
		if create[name] != tag {
			t.Errorf("json tag %s: NFSServiceUpdate %q, NFSService %q", name, tag, create[name])
		}
	}
}

func jsonTags(t *testing.T, typ reflect.Type) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for field := range typ.Fields() {
		tag := field.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			t.Fatalf("field %s has no json name", field.Name)
		}
		out[name] = tag
	}
	return out
}
