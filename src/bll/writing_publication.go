package bll

import (
	"context"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

// TODO: more validation
type CreatePublicationInput struct {
	GID      util.ID           `json:"gid" cbor:"gid" validate:"required"`
	CID      util.ID           `json:"cid" cbor:"cid" validate:"required"`
	Language string            `json:"language" cbor:"language" validate:"required"`
	Version  int16             `json:"version" cbor:"version" validate:"required"`
	Draft    *PublicationDraft `json:"draft,omitempty" cbor:"draft,omitempty"`
}

type PublicationDraft struct {
	GID         util.ID      `json:"gid" cbor:"gid" validate:"required"`
	Language    string       `json:"language" cbor:"language" validate:"required"`
	Model       string       `json:"model" cbor:"model" validate:"required"`
	Genre       []string     `json:"genre" cbor:"genre"`
	Title       string       `json:"title" cbor:"title" validate:"required"`
	Description string       `json:"description" cbor:"description"`
	Cover       string       `json:"cover" cbor:"cover" validate:"http_url"`
	Keywords    []string     `json:"keywords" cbor:"keywords"`
	Summary     string       `json:"summary" cbor:"summary"`
	Content     util.CBORRaw `json:"content" cbor:"content" validate:"required"`
}

func (i *CreatePublicationInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type PublicationOutput struct {
	GID         util.ID       `json:"gid" cbor:"gid"`
	CID         util.ID       `json:"cid" cbor:"cid"`
	Language    string        `json:"language" cbor:"language"`
	Version     int16         `json:"version" cbor:"version"`
	Rating      *int8         `json:"rating,omitempty" cbor:"rating,omitempty"`
	Status      *int8         `json:"status,omitempty" cbor:"status,omitempty"`
	Creator     *util.ID      `json:"creator,omitempty" cbor:"creator,omitempty"`
	CreatedAt   *int64        `json:"created_at,omitempty" cbor:"created_at,omitempty"`
	UpdatedAt   *int64        `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	Model       *string       `json:"model,omitempty" cbor:"model,omitempty"`
	OriginalUrl *string       `json:"original_url,omitempty" cbor:"original_url,omitempty"`
	Genre       *[]string     `json:"genre,omitempty" cbor:"genre,omitempty"`
	Title       *string       `json:"title,omitempty" cbor:"title,omitempty"`
	Description *string       `json:"description,omitempty" cbor:"description,omitempty"`
	Cover       *string       `json:"cover,omitempty" cbor:"cover,omitempty"`
	Keywords    *[]string     `json:"keywords,omitempty" cbor:"keywords,omitempty"`
	Authors     *[]string     `json:"authors,omitempty" cbor:"authors,omitempty"`
	Summary     *string       `json:"summary,omitempty" cbor:"summary,omitempty"`
	Content     *util.CBORRaw `json:"content,omitempty" cbor:"content,omitempty"`
	License     *string       `json:"license,omitempty" cbor:"license,omitempty"`
}

func (b *Writing) CreatePublication(ctx context.Context, input *CreatePublicationInput) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}
	if err := b.svc.Post(ctx, "/v1/publication", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}
