package content

import (
	"encoding/json"
	"errors"

	"github.com/fxamacker/cbor/v2"
	"github.com/jaevor/go-nanoid"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/util"
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

func (d *DocumentNode) FromTEContents(te TEContents) {
	textMap := make(map[string][]string, len(te))
	for i := range te {
		textMap[te[i].ID] = te[i].Texts
	}
	d.setTexts(textMap)
}

func (d *DocumentNode) setTexts(m map[string][]string) {
	if len(d.Content) == 0 {
		return
	}
	var texts []string
	if v, ok := d.Attrs["id"]; ok {
		texts = m[v.ToString()]
	}

	for i := range d.Content {
		n := &d.Content[i]
		if n.Type == "text" {
			if n.Text != nil && len(texts) > 0 {
				n.Text = &texts[0]
				texts = texts[1:]
			} else {
				n.Text = nil
			}
		} else {
			n.setTexts(m)
		}
	}

	arr := d.Content[:]
	d.Content = d.Content[:0]
	for i := 0; i < len(arr); i++ {
		n := &arr[i]
		if n.Type == "text" && (n.Text == nil || *n.Text == "") {
			continue
		} else {
			d.Content = append(d.Content, *n)
		}
	}

	// should not happen
	for i := range texts {
		d.Content = append(d.Content, DocumentNode{
			Type: "text",
			Text: util.Ptr(texts[i]),
		})
	}
}

func ParseDocumentNode(content []byte) (*DocumentNode, error) {
	if len(content) == 0 {
		return nil, errors.New("empty content")
	}

	doc := &DocumentNode{}
	if err := cbor.Unmarshal(content, doc); err != nil {
		return nil, err
	}
	amender := NewDocumentNodeAmender()
	amender.AmendNode(doc)

	return doc, nil
}

type DocumentNodeAmender struct {
	ids        map[string]struct{}
	generateID func() string
}

func NewDocumentNodeAmender() *DocumentNodeAmender {
	generateID, err := nanoid.Standard(6)
	if err != nil {
		panic(err)
	}

	return &DocumentNodeAmender{ids: make(map[string]struct{}), generateID: generateID}
}

func (a *DocumentNodeAmender) HasId(id string) bool {
	_, ok := a.ids[id]
	return ok
}

// ensure id is unique
func (a *DocumentNodeAmender) amendId(id string) string {
	if id == "" {
		id = a.generateID()
	}

	for a.HasId(id) {
		id = a.generateID()
	}

	a.ids[id] = struct{}{}
	return id
}

var uidTypes = []string{"blockquote", "codeBlock", "detailsSummary", "detailsContent", "heading", "listItem", "paragraph", "tableHeader", "tableCell"}

// https://prosemirror.net/docs/ref/#model.Document_Structure
func (a *DocumentNodeAmender) AmendNode(node *DocumentNode) {
	// attrs: Attrs
	if util.SliceHas(uidTypes, node.Type) {
		if node.Attrs == nil {
			node.Attrs = map[string]AttrValue{"id": String(a.amendId(""))}
		} else {
			node.Attrs["id"] = String(a.amendId(node.Attrs["id"].ToString()))
		}
	}

	// content: Node[]
	if len(node.Content) > 0 {
		for i := range node.Content {
			a.AmendNode(&node.Content[i])
		}
	}
}

func EstimateTranslatingString(content *util.Bytes) (string, error) {
	if content == nil {
		return "", gear.ErrInternalServerError.WithMsg("empty content")
	}

	doc, err := ParseDocumentNode(*content)
	if err != nil {
		return "", gear.ErrInternalServerError.From(err)
	}
	contents := doc.ToTEContents()
	for i := range contents {
		contents[i].ID = ""
	}

	teTokens, err := json.Marshal(contents)
	if err != nil {
		return "", gear.ErrInternalServerError.From(err)
	}
	return string(teTokens), nil
}
