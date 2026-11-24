package cmd

import (
	"path/filepath"
	"testing"

	"github.com/netapp/ontap-mcp/assert"
	"github.com/netapp/ontap-mcp/catalog"
)

const testSwagger = `
paths:
  /network/ip/interfaces:
    get:
      responses:
        "200":
          schema:
            $ref: "#/definitions/ip_interface_response"
definitions:
  ip_interface_response:
    properties:
      records:
        items:
          $ref: "#/definitions/ip_interface"
  ip_interface:
    properties:
      _links:
        $ref: "#/definitions/self_link"
      _tags:
        description: Tags
      name:
        description: Interface name
      ip:
        $ref: "#/definitions/ip_info"
      secret:
        description: internal
        x-ntap-visibility:
          type: private
`

const inlineRecordsSwagger = `
paths:
  /cluster/nodes:
    get:
      responses:
        "200":
          schema:
            $ref: "#/definitions/node_response"
definitions:
  node_response:
    properties:
      records:
        items:
          description: Complete node information
          properties:
            _links:
              properties:
                self:
                  $ref: "#/definitions/href"
            name:
              description: Node name
            ha:
              properties:
                enabled:
                  description: HA enabled
            date:
              description: The current or "wall clock" time of the node
              x-ntap-introduced: "9.10"
`

func TestGenerateAPICatalogInlineRecords(t *testing.T) {
	out := filepath.Join(t.TempDir(), "catalog.json")
	assert.Nil(t, generateAPICatalog([]byte(inlineRecordsSwagger), "9.19", out))

	cat, _, err := catalog.Load(out)
	assert.Nil(t, err)
	assert.Equal(t, cat["/cluster/nodes"].Fields, map[string]catalog.FieldInfo{
		"name": {Desc: "Node name"},
		"ha":   {},
		"date": {Desc: `The current or "wall clock" time of the node`, Since: "9.10"},
	})
}

func TestGenerateAPICatalogKeepsUndescribedFields(t *testing.T) {
	out := filepath.Join(t.TempDir(), "catalog.json")
	assert.Nil(t, generateAPICatalog([]byte(testSwagger), "9.16", out))

	cat, ver, err := catalog.Load(out)
	assert.Nil(t, err)
	assert.Equal(t, ver, "9.16")
	assert.Equal(t, cat["/network/ip/interfaces"].Fields, map[string]catalog.FieldInfo{
		"_tags": {Desc: "Tags"},
		"name":  {Desc: "Interface name"},
		"ip":    {},
	})
}
