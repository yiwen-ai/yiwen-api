package content

import (
	"errors"

	"github.com/fxamacker/cbor/v2"
)

type PartialNode struct {
	Type  string               `json:"type" cbor:"type"`
	Attrs map[string]AttrValue `json:"attrs,omitempty" cbor:"attrs,omitempty"`
	Text  *string              `json:"text,omitempty" cbor:"text,omitempty"`
}

type DocumentNode struct {
	Type    string               `json:"type" cbor:"type"`
	Attrs   map[string]AttrValue `json:"attrs,omitempty" cbor:"attrs,omitempty"`
	Text    *string              `json:"text,omitempty" cbor:"text,omitempty"`
	Marks   []PartialNode        `json:"marks,omitempty" cbor:"marks,omitempty"`
	Content []DocumentNode       `json:"content,omitempty" cbor:"content,omitempty"`
}

type TEContent struct {
	ID    string   `json:"id" cbor:"id"`
	Texts []string `json:"texts" cbor:"texts"`
}

type TEContents []*TEContent

func (te *TEContents) visitNode(node *DocumentNode) {
	if len(node.Content) == 0 {
		return
	}

	var content *TEContent
	if v, ok := node.Attrs["id"]; ok {
		if id := v.ToString(); id != "" {
			content = &TEContent{ID: id, Texts: make([]string, 0)}
			*te = append(*te, content)
		}
	}

	for _, child := range node.Content {
		if child.Text != nil && content != nil {
			content.Texts = append(content.Texts, *child.Text)
		} else {
			te.visitNode(&child)
		}
	}
}

func (d DocumentNode) ToTEContents() []*TEContent {
	tes := new(TEContents)
	for i, node := range d.Content {
		tes.visitNode(&node)

		if i < len(d.Content)-1 {
			// dashes (------) is a horizontal rule, work as a top section separator
			*tes = append(*tes, &TEContent{ID: "------", Texts: make([]string, 0)})
		}
	}
	return *tes
}

func ToTEContents(content []byte) (TEContents, error) {
	if len(content) == 0 {
		return nil, errors.New("empty content")
	}

	doc := &DocumentNode{}
	if err := cbor.Unmarshal(content, doc); err != nil {
		return nil, err
	}

	contents := doc.ToTEContents()
	if len(contents) == 0 {
		return nil, errors.New("empty content")
	}
	return contents, nil
}
