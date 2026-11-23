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
      name:
        description: Interface name
      ip:
        $ref: "#/definitions/ip_info"
      secret:
        description: internal
        x-ntap-visibility:
          type: private
`

func TestGenerateAPICatalogKeepsUndescribedFields(t *testing.T) {
	out := filepath.Join(t.TempDir(), "catalog.json")
	assert.Nil(t, generateAPICatalog([]byte(testSwagger), "9.16", out))

	cat, err := catalog.Load(out)
	assert.Nil(t, err)
	assert.Equal(t, cat["/network/ip/interfaces"].Fields, map[string]catalog.FieldInfo{
		"name": {Desc: "Interface name"},
		"ip":   {},
	})
}
