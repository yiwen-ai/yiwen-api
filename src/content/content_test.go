package content

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentNode(t *testing.T) {
	assert := assert.New(t)

	jsonData, err := os.ReadFile("./content.json")
	require.NoError(t, err)

	var jsonObj DocumentNode
	err = json.Unmarshal(jsonData, &jsonObj)
	require.NoError(t, err)

	cborData, err := cbor.Marshal(jsonObj)
	require.NoError(t, err)

	var cborObj DocumentNode
	err = cbor.Unmarshal(cborData, &cborObj)
	require.NoError(t, err)

	assert.Equal(len(jsonObj.Content), len(cborObj.Content))
	assert.True(reflect.DeepEqual(jsonObj, cborObj))

	cborData2, err := cbor.Marshal(cborObj)
	require.NoError(t, err)

	var cborObj2 DocumentNode
	err = cbor.Unmarshal(cborData2, &cborObj2)
	require.NoError(t, err)

	assert.True(reflect.DeepEqual(cborObj2, cborObj))
}

func TestToTEContents(t *testing.T) {
	assert := assert.New(t)

	data, err := os.ReadFile("./content.json")
	require.NoError(t, err)

	var doc DocumentNode
	err = json.Unmarshal(data, &doc)
	require.NoError(t, err)

	contents := doc.ToTEContents()
	data, err = json.Marshal(contents)
	require.NoError(t, err)
	// os.WriteFile("./content.te.json", data, 0644)
	teData, err := os.ReadFile("./content.te.json")
	require.NoError(t, err)
	assert.JSONEq(string(data), string(teData))
}
