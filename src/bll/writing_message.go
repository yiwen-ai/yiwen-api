package bll

import (
	"context"
	"net/url"

	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type KVMessage map[string]string

func (m KVMessage) ToTEContents() content.TEContents {
	tes := make(content.TEContents, 0, len(m))
	for k, v := range m {
		tes = append(tes, &content.TEContent{ID: k, Texts: []string{v}})
	}
	return tes
}

func (d *KVMessage) FromTEContents(te content.TEContents) {
	for _, v := range te {
		if len(v.Texts) == 1 {
			map[string]string(*d)[v.ID] = v.Texts[0]
		}
	}
}

func (d *KVMessage) WithContent(data util.Bytes) error {
	var te content.TEContents
	if err := cbor.Unmarshal(data, &te); err != nil {
		return err
	}

	d.FromTEContents(te)
	return nil
}

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

	return nil
}

type MessageOutput struct {
	ID           util.ID               `json:"id" cbor:"id"`
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
	Version  uint16      `json:"version" cbor:"version" validate:"gte=1,lte=32767"`
	Context  *string     `json:"context,omitempty" cbor:"context,omitempty" validate:"omitempty,gte=4,lte=1024"`
	Language *string     `json:"language,omitempty" cbor:"language,omitempty"`
	Message  *util.Bytes `json:"message,omitempty" cbor:"message,omitempty"`
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
