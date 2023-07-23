package bll

import (
	"context"
	"net/url"
	"strconv"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

// TODO: more validation
type CreatePublicationInput struct {
	GID      util.ID           `json:"gid" cbor:"gid" validate:"required"`
	CID      util.ID           `json:"cid" cbor:"cid" validate:"required"`
	Language string            `json:"language" cbor:"language" validate:"required"`
	Version  int16             `json:"version" cbor:"version" validate:"required,gte=1,lte=10000"`
	Draft    *PublicationDraft `json:"draft,omitempty" cbor:"draft,omitempty"`
}

type PublicationDraft struct {
	GID      util.ID    `json:"gid" cbor:"gid" validate:"required"`
	Language string     `json:"language" cbor:"language" validate:"required"`
	Title    string     `json:"title" cbor:"title" validate:"required,gte=4,lte=256"`
	Model    *string    `json:"model,omitempty" cbor:"model,omitempty" validate:"omitempty,gte=2,lte=16"`
	Genre    *[]string  `json:"genre,omitempty" cbor:"genre,omitempty"`
	Cover    *string    `json:"cover,omitempty" cbor:"cover,omitempty" validate:"omitempty,http_url"`
	Keywords *[]string  `json:"keywords,omitempty" cbor:"keywords,omitempty" validate:"omitempty,gte=0,lte=5"`
	Summary  string     `json:"summary" cbor:"summary" validate:"required,gte=4,lte=2048"`
	Content  util.Bytes `json:"content" cbor:"content" validate:"required"`
}

func (i *CreatePublicationInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type PublicationOutput struct {
	GID         util.ID     `json:"gid" cbor:"gid"`
	CID         util.ID     `json:"cid" cbor:"cid"`
	Language    string      `json:"language" cbor:"language"`
	Version     int16       `json:"version" cbor:"version"`
	Rating      *int8       `json:"rating,omitempty" cbor:"rating,omitempty"`
	Status      *int8       `json:"status,omitempty" cbor:"status,omitempty"`
	Creator     *util.ID    `json:"creator,omitempty" cbor:"creator,omitempty"`
	CreatedAt   *int64      `json:"created_at,omitempty" cbor:"created_at,omitempty"`
	UpdatedAt   *int64      `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	Model       *string     `json:"model,omitempty" cbor:"model,omitempty"`
	OriginalUrl *string     `json:"original_url,omitempty" cbor:"original_url,omitempty"`
	Genre       *[]string   `json:"genre,omitempty" cbor:"genre,omitempty"`
	Title       *string     `json:"title,omitempty" cbor:"title,omitempty"`
	Cover       *string     `json:"cover,omitempty" cbor:"cover,omitempty"`
	Keywords    *[]string   `json:"keywords,omitempty" cbor:"keywords,omitempty"`
	Authors     *[]string   `json:"authors,omitempty" cbor:"authors,omitempty"`
	Summary     *string     `json:"summary,omitempty" cbor:"summary,omitempty"`
	Content     *util.Bytes `json:"content,omitempty" cbor:"content,omitempty"`
	License     *string     `json:"license,omitempty" cbor:"license,omitempty"`
}

func (b *Writing) CreatePublication(ctx context.Context, input *CreatePublicationInput) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}
	if err := b.svc.Post(ctx, "/v1/publication", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type QueryPublication struct {
	GID      util.ID `json:"gid" cbor:"gid" query:"gid" validate:"required"`
	CID      util.ID `json:"cid" cbor:"cid" query:"cid" validate:"required"`
	Language string  `json:"language" cbor:"language" validate:"required"`
	Version  int16   `json:"version" cbor:"version"  validate:"required,gte=1,lte=10000"`
	Fields   string  `json:"fields" cbor:"fields" query:"fields"`
}

func (i *QueryPublication) Validate() error {
	return nil
}

func (b *Writing) GetPublication(ctx context.Context, input *QueryPublication) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}

	query := url.Values{}
	query.Add("gid", input.GID.String())
	query.Add("cid", input.CID.String())
	query.Add("language", input.Language)
	query.Add("version", strconv.Itoa(int(input.Version)))
	if input.Fields != "" {
		query.Add("fields", input.Fields)
	}
	if err := b.svc.Get(ctx, "/v1/publication?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type UpdatePublicationInput struct {
	GID       util.ID   `json:"gid" cbor:"gid" validate:"required"`
	ID        util.ID   `json:"id" cbor:"id" validate:"required"`
	Language  string    `json:"language" cbor:"language" validate:"required"`
	Version   int16     `json:"version" cbor:"version" validate:"required,gte=1,lte=10000"`
	UpdatedAt int64     `json:"updated_at" cbor:"updated_at"  validate:"required"`
	Model     *string   `json:"model,omitempty" cbor:"model,omitempty" validate:"omitempty,gte=2,lte=16"`
	Title     *string   `json:"title,omitempty" cbor:"title,omitempty" validate:"omitempty,gte=4,lte=256"`
	Cover     *string   `json:"cover,omitempty" cbor:"cover,omitempty" validate:"omitempty,http_url"`
	Keywords  *[]string `json:"keywords,omitempty" cbor:"keywords,omitempty" validate:"omitempty,gte=0,lte=5"`
	Summary   *string   `json:"summary,omitempty" cbor:"summary,omitempty" validate:"omitempty,gte=4,lte=2048"`
}

func (i *UpdatePublicationInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) UpdatePublication(ctx context.Context, input *UpdatePublicationInput) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}
	if err := b.svc.Patch(ctx, "/v1/publication", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Writing) DeletePublication(ctx context.Context, input *QueryPublication) (bool, error) {
	output := SuccessResponse[bool]{}

	query := url.Values{}
	query.Add("gid", input.GID.String())
	query.Add("cid", input.CID.String())
	query.Add("language", input.Language)
	query.Add("version", strconv.Itoa(int(input.Version)))

	if err := b.svc.Delete(ctx, "/v1/publication?"+query.Encode(), &output); err != nil {
		return false, err
	}

	return output.Result, nil
}

func (b *Writing) ListPublication(ctx context.Context, input *GIDPagination) (*SuccessResponse[[]*PublicationOutput], error) {
	output := SuccessResponse[[]*PublicationOutput]{}
	if err := b.svc.Post(ctx, "/v1/publication/list", input, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

type QueryAPublication struct {
	GID util.ID `json:"gid" cbor:"gid" query:"gid" validate:"required"`
	CID util.ID `json:"cid" cbor:"cid" query:"cid" validate:"required"`
}

func (i *QueryAPublication) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) GetPublicationList(ctx context.Context, input *QueryAPublication) (*SuccessResponse[[]*PublicationOutput], error) {
	output := SuccessResponse[[]*PublicationOutput]{}
	query := url.Values{}
	query.Add("gid", input.GID.String())
	query.Add("cid", input.CID.String())
	if err := b.svc.Get(ctx, "/v1/publication/publish_list?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output, nil
}

type UpdatePublicationStatusInput struct {
	GID       util.ID `json:"gid" cbor:"gid" validate:"required"`
	CID       util.ID `json:"cid" cbor:"cid" validate:"required"`
	Language  string  `json:"language" cbor:"language" validate:"required"`
	Version   int16   `json:"version" cbor:"version" validate:"required,gte=1,lte=10000"`
	UpdatedAt int64   `json:"updated_at" cbor:"updated_at" validate:"required"`
	Status    int8    `json:"status" cbor:"status" validate:"required,gte=-2,lte=2"`
}

func (i *UpdatePublicationStatusInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) UpdatePublicationStatus(ctx context.Context, input *UpdatePublicationStatusInput) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}
	if err := b.svc.Patch(ctx, "/v1/publication/update_status", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

// TODO: more validation
type UpdatePublicationContentInput struct {
	GID       util.ID    `json:"gid" cbor:"gid" validate:"required"`
	CID       util.ID    `json:"cid" cbor:"cid" validate:"required"`
	Language  string     `json:"language" cbor:"language" validate:"required"`
	Version   int16      `json:"version" cbor:"version" validate:"required,gte=1,lte=10000"`
	UpdatedAt int64      `json:"updated_at" cbor:"updated_at" validate:"required"`
	Content   util.Bytes `json:"content" cbor:"content" validate:"required"`
}

func (i *UpdatePublicationContentInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) UpdatePublicationContent(ctx context.Context, input *UpdatePublicationContentInput) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}
	if err := b.svc.Put(ctx, "/v1/publication/update_content", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}
