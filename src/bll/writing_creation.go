package bll

import (
	"context"
	"net/url"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

// TODO: more validation
type CreateCreationInput struct {
	GID         util.ID    `json:"gid" cbor:"gid" validate:"required"`
	Title       string     `json:"title" cbor:"title" validate:"gte=1,lte=256"`
	Content     util.Bytes `json:"content" cbor:"content" validate:"required"`
	Language    string     `json:"language" cbor:"language"`
	OriginalUrl *string    `json:"original_url,omitempty" cbor:"original_url,omitempty" validate:"omitempty,http_url"`
	Genre       *[]string  `json:"genre,omitempty" cbor:"genre,omitempty"`
	Cover       *string    `json:"cover,omitempty" cbor:"cover,omitempty" validate:"omitempty,http_url"`
	Keywords    *[]string  `json:"keywords,omitempty" cbor:"keywords,omitempty" validate:"omitempty,gte=0,lte=5"`
	Labels      *[]string  `json:"labels,omitempty" cbor:"labels,omitempty" validate:"omitempty,gte=0,lte=5"`
	Authors     *[]string  `json:"authors,omitempty" cbor:"authors,omitempty" validate:"omitempty,gte=0,lte=10"`
	License     *string    `json:"license,omitempty" cbor:"license,omitempty"`
}

func (i *CreateCreationInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type CreationOutput struct {
	ID          util.ID     `json:"id" cbor:"id"`
	GID         util.ID     `json:"gid" cbor:"gid"`
	Status      *int8       `json:"status,omitempty" cbor:"status,omitempty"`
	Rating      *int8       `json:"rating,omitempty" cbor:"rating,omitempty"`
	Version     *uint16     `json:"version,omitempty" cbor:"version,omitempty"`
	Language    *string     `json:"language,omitempty" cbor:"language,omitempty"`
	Creator     *util.ID    `json:"creator,omitempty" cbor:"creator,omitempty"`
	CreatedAt   *int64      `json:"created_at,omitempty" cbor:"created_at,omitempty"`
	UpdatedAt   *int64      `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	OriginalUrl *string     `json:"original_url,omitempty" cbor:"original_url,omitempty"`
	Genre       *[]string   `json:"genre,omitempty" cbor:"genre,omitempty"`
	Title       *string     `json:"title,omitempty" cbor:"title,omitempty"`
	Cover       *string     `json:"cover,omitempty" cbor:"cover,omitempty"`
	Keywords    *[]string   `json:"keywords,omitempty" cbor:"keywords,omitempty"`
	Labels      *[]string   `json:"labels,omitempty" cbor:"labels,omitempty"`
	Authors     *[]string   `json:"authors,omitempty" cbor:"authors,omitempty"`
	Reviewers   *[]util.ID  `json:"reviewers,omitempty" cbor:"reviewers,omitempty"`
	Summary     *string     `json:"summary,omitempty" cbor:"summary,omitempty"`
	Content     *util.Bytes `json:"content,omitempty" cbor:"content,omitempty"`
	License     *string     `json:"license,omitempty" cbor:"license,omitempty"`
	CreatorInfo *UserInfo   `json:"creator_info,omitempty" cbor:"creator_info,omitempty"`
	GroupInfo   *GroupInfo  `json:"group_info,omitempty" cbor:"group_info,omitempty"`
}

type CreationOutputs []CreationOutput

func (list *CreationOutputs) LoadCreators(loader func(ids ...util.ID) []UserInfo) {
	if len(*list) == 0 {
		return
	}

	ids := make([]util.ID, 0, len(*list))
	for _, v := range *list {
		if v.Creator != nil {
			ids = append(ids, *v.Creator)
		}
	}

	users := loader(ids...)
	if len(users) == 0 {
		return
	}

	infoMap := make(map[util.ID]*UserInfo, len(users))
	for i := range users {
		infoMap[*users[i].ID] = &users[i]
		infoMap[*users[i].ID].ID = nil
	}

	for i := range *list {
		(*list)[i].CreatorInfo = infoMap[*(*list)[i].Creator]
		(*list)[i].Creator = nil
	}
}

func (list *CreationOutputs) LoadGroups(loader func(ids ...util.ID) []GroupInfo) {
	if len(*list) == 0 {
		return
	}

	ids := make([]util.ID, 0, len(*list))
	for _, v := range *list {
		ids = append(ids, v.GID)
	}

	groups := loader(ids...)
	if len(groups) == 0 {
		return
	}

	infoMap := make(map[util.ID]*GroupInfo, len(groups))
	for i := range groups {
		infoMap[groups[i].ID] = &groups[i]
	}

	for i := range *list {
		(*list)[i].GroupInfo = infoMap[(*list)[i].GID]
	}
}

func (b *Writing) CreateCreation(ctx context.Context, input *CreateCreationInput) (*CreationOutput, error) {
	output := SuccessResponse[CreationOutput]{}
	if err := b.svc.Post(ctx, "/v1/creation", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type QueryCreation struct {
	GID    util.ID `json:"gid" cbor:"gid" query:"gid" validate:"required"`
	ID     util.ID `json:"id" cbor:"id" query:"id" validate:"required"`
	Fields string  `json:"fields" cbor:"fields" query:"fields"`
}

func (i *QueryCreation) Validate() error {
	return nil
}

func (b *Writing) GetCreation(ctx context.Context, input *QueryCreation) (*CreationOutput, error) {
	output := SuccessResponse[CreationOutput]{}

	query := url.Values{}
	query.Add("gid", input.GID.String())
	query.Add("id", input.ID.String())
	if input.Fields != "" {
		query.Add("fields", input.Fields)
	}
	if err := b.svc.Get(ctx, "/v1/creation?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

// TODO: more validation
type UpdateCreationInput struct {
	GID       util.ID   `json:"gid" cbor:"gid" validate:"required"`
	ID        util.ID   `json:"id" cbor:"id" validate:"required"`
	UpdatedAt int64     `json:"updated_at" cbor:"updated_at"  validate:"required"`
	Title     *string   `json:"title,omitempty" cbor:"title,omitempty" validate:"gte=4,lte=256"`
	Cover     *string   `json:"cover,omitempty" cbor:"cover,omitempty" validate:"omitempty,http_url"`
	Keywords  *[]string `json:"keywords,omitempty" cbor:"keywords,omitempty" validate:"omitempty,gte=0,lte=5"`
	Labels    *[]string `json:"labels,omitempty" cbor:"labels,omitempty" validate:"omitempty,gte=0,lte=5"`
	Authors   *[]string `json:"authors,omitempty" cbor:"authors,omitempty" validate:"omitempty,gte=0,lte=10"`
	Summary   *string   `json:"summary,omitempty" cbor:"summary,omitempty" validate:"omitempty,gte=4,lte=2048"`
	License   *string   `json:"license,omitempty" cbor:"license,omitempty"`
}

func (i *UpdateCreationInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) UpdateCreation(ctx context.Context, input *UpdateCreationInput) (*CreationOutput, error) {
	output := SuccessResponse[CreationOutput]{}
	if err := b.svc.Patch(ctx, "/v1/creation", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Writing) DeleteCreation(ctx context.Context, input *QueryCreation) (bool, error) {
	output := SuccessResponse[bool]{}

	query := url.Values{}
	query.Add("gid", input.GID.String())
	query.Add("id", input.ID.String())
	if err := b.svc.Delete(ctx, "/v1/creation?"+query.Encode(), &output); err != nil {
		return false, err
	}

	return output.Result, nil
}

func (b *Writing) ListCreation(ctx context.Context, input *GIDPagination) (*SuccessResponse[CreationOutputs], error) {
	output := SuccessResponse[CreationOutputs]{}
	if err := b.svc.Post(ctx, "/v1/creation/list", input, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

// TODO: more validation
type UpdateCreationStatusInput struct {
	GID       util.ID `json:"gid" cbor:"gid" validate:"required"`
	ID        util.ID `json:"id" cbor:"id" validate:"required"`
	UpdatedAt int64   `json:"updated_at" cbor:"updated_at" validate:"required"`
	Status    int8    `json:"status" cbor:"status" validate:"gte=-2,lte=2"`
}

func (i *UpdateCreationStatusInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) UpdateCreationStatus(ctx context.Context, input *UpdateCreationStatusInput) (*CreationOutput, error) {
	output := SuccessResponse[CreationOutput]{}
	if err := b.svc.Patch(ctx, "/v1/creation/update_status", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

// TODO: more validation
type UpdateCreationContentInput struct {
	GID       util.ID    `json:"gid" cbor:"gid" validate:"required"`
	ID        util.ID    `json:"id" cbor:"id" validate:"required"`
	UpdatedAt int64      `json:"updated_at" cbor:"updated_at" validate:"required"`
	Language  string     `json:"language" cbor:"language" validate:"required"`
	Content   util.Bytes `json:"content" cbor:"content" validate:"required"`
}

func (i *UpdateCreationContentInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) UpdateCreationContent(ctx context.Context, input *UpdateCreationContentInput) (*CreationOutput, error) {
	output := SuccessResponse[CreationOutput]{}
	if err := b.svc.Put(ctx, "/v1/creation/update_content", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}
