package content

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yiwen-ai/yiwen-api/src/util"
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

func TestFromTEContents(t *testing.T) {
	assert := assert.New(t)

	data, err := os.ReadFile("./content.json")
	require.NoError(t, err)

	var doc DocumentNode
	err = json.Unmarshal(data, &doc)
	require.NoError(t, err)

	teData, err := os.ReadFile("./content.te.zho.json")
	require.NoError(t, err)

	var te TEContents
	err = json.Unmarshal(teData, &te)
	require.NoError(t, err)

	doc.FromTEContents(te)
	data, err = json.Marshal(doc)
	require.NoError(t, err)
	// os.WriteFile("./content.zho.json", data, 0644)

	// Should:
	// 1. process nested text content in order;
	// 2. Missing text does not affect processing.
	zhData, err := os.ReadFile("./content.zho.json")
	require.NoError(t, err)
	assert.JSONEq(string(data), string(zhData))
}

func TestDocumentNodeAmender(t *testing.T) {
	assert := assert.New(t)
	amender := NewDocumentNodeAmender()
	obj := DocumentNode{
		Type: "doc",
		Content: []DocumentNode{
			{
				Type: "heading",
				Attrs: map[string]AttrValue{
					"id":    String("abcdef"),
					"level": Int64(1),
				},
				Content: []DocumentNode{
					{
						Type: "text",
						Text: util.Ptr("Hello"),
					},
				},
			},
			{
				Type: "heading",
				Attrs: map[string]AttrValue{
					"id":    String("abcdef"),
					"level": Int64(1),
				},
				Content: []DocumentNode{
					{
						Type: "text",
						Text: util.Ptr("world"),
					},
				},
			},
			{
				Type: "paragraph",
				Content: []DocumentNode{
					{
						Type: "text",
						Text: util.Ptr("some text"),
					},
				},
			},
		},
	}

	assert.Equal(obj.Content[0].Attrs["id"].ToAny(), obj.Content[1].Attrs["id"].ToAny())
	assert.Nil(obj.Content[2].Attrs["id"].ToAny())
	amender.AmendNode(&obj)
	assert.NotEqual(obj.Content[0].Attrs["id"].ToAny(), obj.Content[1].Attrs["id"].ToAny())
	assert.NotNil(obj.Content[2].Attrs["id"].ToAny())

	data, err := json.Marshal(obj)
	fmt.Println(string(data))
	assert.Nil(err)
}

func TestEstimateTranslatingString(t *testing.T) {
	assert := assert.New(t)
	obj := DocumentNode{
		Type: "doc",
		Content: []DocumentNode{
			{
				Type: "heading",
				Attrs: map[string]AttrValue{
					"id":    String("abcdef"),
					"level": Int64(1),
				},
				Content: []DocumentNode{
					{
						Type: "text",
						Text: util.Ptr("Hello"),
					},
				},
			},
			{
				Type: "heading",
				Attrs: map[string]AttrValue{
					"id":    String("123456"),
					"level": Int64(1),
				},
				Content: []DocumentNode{
					{
						Type: "text",
						Text: util.Ptr("world"),
					},
				},
			},
			{
				Type: "paragraph",
				Content: []DocumentNode{
					{
						Type: "text",
						Text: util.Ptr("some text"),
					},
				},
			},
		},
	}

	te := obj.ToTEContents()
	require.Equal(t, 4, len(te))
	assert.Equal("abcdef", te[0].ID)
	assert.Equal("------", te[1].ID)
	assert.Equal("123456", te[2].ID)
	assert.Equal("------", te[3].ID)

	data, err := cbor.Marshal(&obj)
	require.Nil(t, err)

	str, err := EstimateTranslatingString(util.Ptr(util.Bytes(data)))
	require.Nil(t, err)
	fmt.Println(str)
	assert.Equal("[\"0\"]\n[\"Hello\"]\n[\"2\"]\n[\"world\"]\n[\"4\"]\n[\"some text\"]\n", str)
}

func TestContentFilter(t *testing.T) {
	assert := assert.New(t)
	obj := DocumentNode{
		Type: "doc",
		Content: []DocumentNode{
			{
				Type:  "paragraph",
				Attrs: map[string]AttrValue{"id": String("123456")},
				Content: []DocumentNode{
					{
						Type: "text",
						Text: util.Ptr("some 暴力强奸 text"),
					},
				},
			},
		},
	}

	te := obj.ToTEContents()
	assert.Equal(te[0].Texts, []string{"some 暴力强奸 text"})
	te.ContentFilter()
	assert.Equal(te[0].Texts, []string{"some 暴力** text"})
}
