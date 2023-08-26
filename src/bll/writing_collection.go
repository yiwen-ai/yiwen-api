package bll

import (
	"context"
	"net/url"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

// TODO: more validation
type CreateCollectionInput struct {
	GID      util.ID   `json:"gid" cbor:"gid" validate:"required"`
	CID      util.ID   `json:"cid" cbor:"cid" validate:"required"`
	Language string    `json:"language" cbor:"language" validate:"required"`
	Version  uint16    `json:"version" cbor:"version" validate:"gte=1,lte=10000"`
	Title    string    `json:"title" cbor:"title" validate:"gte=4,lte=256"`
	Labels   *[]string `json:"labels,omitempty" cbor:"labels,omitempty" validate:"omitempty,gte=0,lte=5"`
}

func (i *CreateCollectionInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type CollectionOutput struct {
	ID        util.ID    `json:"id" cbor:"id"`
	GID       util.ID    `json:"gid" cbor:"gid"`
	CID       util.ID    `json:"cid" cbor:"cid"`
	Language  string     `json:"language" cbor:"language"`
	Version   uint16     `json:"version" cbor:"version"`
	UpdatedAt *int64     `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	Title     *string    `json:"title,omitempty" cbor:"title,omitempty"`
	Labels    *[]string  `json:"labels,omitempty" cbor:"labels,omitempty"`
	GroupInfo *GroupInfo `json:"group_info,omitempty" cbor:"group_info,omitempty"`
}

type CollectionOutputs []CollectionOutput

func (list *CollectionOutputs) LoadGroups(loader func(ids ...util.ID) []GroupInfo) {
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

func (b *Writing) CreateCollection(ctx context.Context, input *CreateCollectionInput) (*CollectionOutput, error) {
	output := SuccessResponse[CollectionOutput]{}
	if err := b.svc.Post(ctx, "/v1/collection", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type QueryCollection struct {
	ID     util.ID `json:"id" cbor:"id" query:"id" validate:"required"`
	Fields string  `json:"fields" cbor:"fields" query:"fields"`
}

func (i *QueryCollection) Validate() error {
	return nil
}

// TODO: more validation
type UpdateCollectionInput struct {
	ID        util.ID   `json:"id" cbor:"id" validate:"required"`
	UpdatedAt int64     `json:"updated_at" cbor:"updated_at"  validate:"required"`
	Version   *uint16   `json:"version" cbor:"version" validate:"omitempty,gte=1,lte=10000"`
	Title     *string   `json:"title,omitempty" cbor:"title,omitempty" validate:"omitempty,gte=4,lte=256"`
	Labels    *[]string `json:"labels,omitempty" cbor:"labels,omitempty" validate:"omitempty,gte=0,lte=5"`
}

func (i *UpdateCollectionInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) UpdateCollection(ctx context.Context, input *UpdateCollectionInput) (*CollectionOutput, error) {
	output := SuccessResponse[CollectionOutput]{}
	if err := b.svc.Patch(ctx, "/v1/collection", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Writing) DeleteCollection(ctx context.Context, input *QueryCollection) (bool, error) {
	output := SuccessResponse[bool]{}

	query := url.Values{}
	query.Add("id", input.ID.String())
	if err := b.svc.Delete(ctx, "/v1/collection?"+query.Encode(), &output); err != nil {
		return false, err
	}

	return output.Result, nil
}

func (b *Writing) ListCollection(ctx context.Context, input *Pagination) (*SuccessResponse[CollectionOutputs], error) {
	output := SuccessResponse[CollectionOutputs]{}
	if err := b.svc.Post(ctx, "/v1/collection/list", input, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

type QueryCollectionByCid struct {
	CID    util.ID `json:"cid" cbor:"cid" query:"cid" validate:"required"`
	Fields string  `json:"fields" cbor:"fields" query:"fields"`
}

func (i *QueryCollectionByCid) Validate() error {
	return nil
}

func (b *Writing) GetCollectionByCid(ctx context.Context, input *QueryCollectionByCid) (*SuccessResponse[CollectionOutputs], error) {
	output := SuccessResponse[CollectionOutputs]{}
	query := url.Values{}
	query.Add("id", input.CID.String())
	if input.Fields != "" {
		query.Add("fields", input.Fields)
	}
	if err := b.svc.Get(ctx, "/v1/collection/by_cid?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output, nil
}
