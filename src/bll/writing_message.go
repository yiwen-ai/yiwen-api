package bll

import (
	"context"
	"errors"
	"net/url"

	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type MessageContainer interface {
	cbor.Unmarshaler
	cbor.Marshaler
	FromTEContents(content.TEContents)
	ToTEContents() content.TEContents
	NewlyAdd(MessageContainer) (MessageContainer, error)
	IsEmpty() bool
	New() MessageContainer
}

type KVMessage map[string]string
type ArrayMessage content.TEContents

func (m KVMessage) ToTEContents() content.TEContents {
	tes := make(content.TEContents, 0, len(m))
	for k, v := range m {
		tes = append(tes, &content.TEContent{ID: k, Texts: []string{v}})
	}
	return tes
}

func (m ArrayMessage) ToTEContents() content.TEContents {
	return content.TEContents(m)
}

func (m *KVMessage) FromTEContents(te content.TEContents) {
	for _, v := range te {
		if len(v.Texts) > 0 {
			map[string]string(*m)[v.ID] = v.Texts[0]
		}
	}
}

func (m *ArrayMessage) FromTEContents(te content.TEContents) {
	mp := make(map[string]*content.TEContent, len(*m))
	for _, v := range *m {
		mp[v.ID] = v
	}
	for _, v := range te {
		if e, ok := mp[v.ID]; !ok {
			*m = append(*m, v)
		} else {
			e.Texts = v.Texts
		}
	}
}

func (m *KVMessage) NewlyAdd(mc MessageContainer) (MessageContainer, error) {
	mm, ok := mc.(*KVMessage)
	if !ok {
		return nil, errors.New("KVMessage.NewlyAdd: invalid message container")
	}

	na := make(KVMessage, len(*m))
	for k := range *m {
		if _, ok := (*mm)[k]; !ok {
			na[k] = (*m)[k]
		}
	}
	return &na, nil
}

func (m *ArrayMessage) NewlyAdd(mc MessageContainer) (MessageContainer, error) {
	mm, ok := mc.(*ArrayMessage)
	if !ok {
		return nil, errors.New("ArrayMessage.NewlyAdd: invalid message container")
	}
	keys := make(map[string]struct{}, len(*mm))
	for _, v := range *mm {
		keys[v.ID] = struct{}{}
	}
	na := make(ArrayMessage, 0, len(*m))
	for _, v := range *m {
		if _, ok := keys[v.ID]; !ok {
			na = append(na, v)
		}
	}
	return &na, nil
}

func (m KVMessage) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(map[string]string(m))
}

func (m ArrayMessage) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(content.TEContents(m))
}

func (m *KVMessage) UnmarshalCBOR(data []byte) error {
	if m == nil {
		return errors.New("KVMessage.UnmarshalCBOR: nil pointer")
	}

	var mm map[string]string
	if err := cbor.Unmarshal(data, &mm); err != nil {
		return errors.New("KVMessage.UnmarshalCBOR: " + err.Error())
	}

	*m = KVMessage(mm)
	return nil
}

func (m *ArrayMessage) UnmarshalCBOR(data []byte) error {
	if m == nil {
		return errors.New("ArrayMessage.UnmarshalCBOR: nil pointer")
	}

	var mm content.TEContents
	if err := cbor.Unmarshal(data, &mm); err != nil {
		return errors.New("ArrayMessage.UnmarshalCBOR: " + err.Error())
	}

	*m = ArrayMessage(mm)
	return nil
}

func (m KVMessage) IsEmpty() bool {
	return len(map[string]string(m)) == 0
}

func (m ArrayMessage) IsEmpty() bool {
	return len(content.TEContents(m)) == 0
}

func (m KVMessage) New() MessageContainer {
	return new(KVMessage)
}

func (m ArrayMessage) New() MessageContainer {
	return new(ArrayMessage)
}

func WithContent[T MessageContainer](m T, data util.Bytes) error {
	var te content.TEContents
	if err := cbor.Unmarshal(data, &te); err != nil {
		return err
	}

	m.FromTEContents(te)
	return nil
}

func FromContent[T MessageContainer](data util.Bytes) (T, error) {
	var m T
	if err := cbor.Unmarshal(data, &m); err != nil {
		return m, err
	}
	return m, nil
}

var _ MessageContainer = (*KVMessage)(nil)
var _ MessageContainer = (*ArrayMessage)(nil)

type CreateMessageInput struct {
	AttachTo util.ID    `json:"attach_to" cbor:"attach_to" validate:"required"`
	Kind     string     `json:"kind" cbor:"kind" validate:"required"`
	Language string     `json:"language" cbor:"language" validate:"required"`
	Context  string     `json:"context" cbor:"context" validate:"gte=0,lte=1024"`
	Message  util.Bytes `json:"message" cbor:"message" validate:"required"`
}

func (i *CreateMessageInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}
	if i.Kind != "group.message" {
		return gear.ErrBadRequest.WithMsg("invalid kind")
	}
	if len(i.Message) > 1024*100 {
		return gear.ErrBadRequest.WithMsg("message is too large")
	}

	if i.Context != "" {
		if tk := util.Tiktokens(i.Context); tk > 2048 {
			return gear.ErrBadRequest.WithMsgf("context is too long, max tokens is 2048, got %d", tk)
		}
	}

	return nil
}

type MessageOutput struct {
	ID           util.ID               `json:"id" cbor:"id"`
	Languages    []string              `json:"languages" cbor:"languages"`
	I18nMessages map[string]util.Bytes `json:"i18n_messages" cbor:"i18n_messages"`
	AttachTo     *util.ID              `json:"attach_to,omitempty" cbor:"attach_to,omitempty"`
	Kind         *string               `json:"kind,omitempty" cbor:"kind,omitempty"`
	Language     *string               `json:"language,omitempty" cbor:"language,omitempty"`
	Version      *uint16               `json:"version,omitempty" cbor:"version,omitempty"`
	CreatedAt    *int64                `json:"created_at,omitempty" cbor:"created_at,omitempty"`
	UpdatedAt    *int64                `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	Context      *string               `json:"context,omitempty" cbor:"context,omitempty"`
	Message      *util.Bytes           `json:"message,omitempty" cbor:"message,omitempty"`
}

func (b *Writing) CreateMessage(ctx context.Context, input *CreateMessageInput) (*MessageOutput, error) {
	output := SuccessResponse[MessageOutput]{}
	if err := b.svc.Post(ctx, "/v1/message", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type UpdateMessageInput struct {
	ID       util.ID     `json:"id" cbor:"id" validate:"required"`
	GID      util.ID     `json:"gid" cbor:"gid" validate:"required"`
	Version  uint16      `json:"version" cbor:"version" validate:"gte=1,lte=32767"`
	Context  *string     `json:"context,omitempty" cbor:"context,omitempty" validate:"omitempty,gte=4,lte=1024"`
	Language *string     `json:"language,omitempty" cbor:"language,omitempty"`
	Message  *util.Bytes `json:"message,omitempty" cbor:"message,omitempty"`
	NewlyAdd *bool       `json:"newly_add,omitempty" cbor:"newly_add,omitempty"` // default true
}

func (i *UpdateMessageInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}
	if i.Message != nil && len(*i.Message) > 1024*100 {
		return gear.ErrBadRequest.WithMsg("message is too large")
	}
	if i.Message != nil && i.Language == nil {
		return gear.ErrBadRequest.WithMsg("language is required with message")
	}
	if i.Context != nil {
		if tk := util.Tiktokens(*i.Context); tk > 2048 {
			return gear.ErrBadRequest.WithMsgf("context is too long, max tokens is 2048, got %d", tk)
		}
	}
	if i.NewlyAdd == nil {
		i.NewlyAdd = util.Ptr(true)
	}

	return nil
}

func (b *Writing) UpdateMessage(ctx context.Context, input *UpdateMessageInput) (*MessageOutput, error) {
	output := SuccessResponse[MessageOutput]{}
	if err := b.svc.Patch(ctx, "/v1/message", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Writing) GetMessage(ctx context.Context, input *QueryID) (*MessageOutput, error) {
	output := SuccessResponse[MessageOutput]{}
	query := url.Values{}
	query.Add("id", input.ID.String())
	if input.Fields != "" {
		query.Add("fields", input.Fields)
	}
	if err := b.svc.Get(ctx, "/v1/message?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}
