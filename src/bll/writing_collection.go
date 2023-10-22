package bll

import (
	"context"
	"net/url"
	"strconv"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type CreateCollectionInput struct {
	GID           util.ID        `json:"gid" cbor:"gid" validate:"required"`
	Language      string         `json:"language" cbor:"language" validate:"required"`
	Context       string         `json:"context" cbor:"context" validate:"gte=0,lte=1024"`
	Info          CollectionInfo `json:"info" cbor:"info" validate:"required"`
	Cover         *string        `json:"cover" cbor:"cover" validate:"omitempty,http_url"`
	Price         *int64         `json:"price" cbor:"price" validate:"omitempty,gte=-1,lte=1000000"`
	CreationPrice *int64         `json:"creation_price" cbor:"creation_price" validate:"omitempty,gte=-1,lte=100000"`
	Parent        *util.ID       `json:"parent,omitempty" cbor:"parent,omitempty"`
}

type CollectionInfo struct {
	Title    string    `json:"title" cbor:"title" validate:"gte=1,lte=256"`
	Summary  *string   `json:"summary,omitempty" cbor:"summary,omitempty" validate:"omitempty,gte=1,lte=2048"`
	Keywords *[]string `json:"keywords,omitempty" cbor:"keywords,omitempty" validate:"omitempty,gte=0,lte=10"`
	Authors  *[]string `json:"authors,omitempty" cbor:"authors,omitempty" validate:"omitempty,gte=0,lte=10"`
}

func (i *CreateCollectionInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type CollectionOutput struct {
	ID            util.ID                   `json:"id" cbor:"id"`
	GID           util.ID                   `json:"gid" cbor:"gid"`
	Status        *int8                     `json:"status,omitempty" cbor:"status,omitempty"`
	Rating        *int8                     `json:"rating,omitempty" cbor:"rating,omitempty"`
	UpdatedAt     *int64                    `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	Cover         *string                   `json:"cover,omitempty" cbor:"cover,omitempty"`
	Price         *int64                    `json:"price,omitempty" cbor:"price,omitempty"`
	CreationPrice *int64                    `json:"creation_price,omitempty" cbor:"creation_price,omitempty"`
	Language      *string                   `json:"language,omitempty" cbor:"language,omitempty"`
	Version       *uint16                   `json:"version,omitempty" cbor:"version,omitempty"`
	Info          *CollectionInfo           `json:"info,omitempty" cbor:"info,omitempty"`
	I18nInfo      map[string]CollectionInfo `json:"i18n_info,omitempty" cbor:"i18n_info,omitempty"`
	Subscription  *SubscriptionOutput       `json:"subscription,omitempty" cbor:"subscription,omitempty"`
	RFP           *RFP                      `json:"rfp,omitempty" cbor:"rfp,omitempty"`
	SubToken      *string                   `json:"subtoken,omitempty" cbor:"subtoken,omitempty"`
	GroupInfo     *GroupInfo                `json:"group_info,omitempty" cbor:"group_info,omitempty"`
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

func (b *Writing) GetCollection(ctx context.Context, input *QueryGidID, status int8) (*CollectionOutput, error) {
	output := SuccessResponse[CollectionOutput]{}
	query := url.Values{}
	query.Add("id", input.ID.String())
	query.Add("gid", input.GID.String())
	query.Add("status", strconv.Itoa(int(status)))
	if input.Fields != "" {
		query.Add("fields", input.Fields)
	}
	if err := b.svc.Get(ctx, "/v1/collection?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type UpdateCollectionInput struct {
	ID            util.ID `json:"id" cbor:"id" validate:"required"`
	GID           util.ID `json:"gid" cbor:"gid" validate:"required"`
	UpdatedAt     int64   `json:"updated_at" cbor:"updated_at" validate:"gte=1"`
	Cover         *string `json:"cover" cbor:"cover" validate:"omitempty,http_url"`
	Price         *int64  `json:"price" cbor:"price" validate:"omitempty,gte=-1,lte=1000000"`
	CreationPrice *int64  `json:"creation_price" cbor:"creation_price" validate:"omitempty,gte=-1,lte=100000"`
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

func (b *Writing) DeleteCollection(ctx context.Context, input *QueryGidID) (bool, error) {
	output := SuccessResponse[bool]{}

	query := url.Values{}
	query.Add("id", input.ID.String())
	query.Add("gid", input.GID.String())
	if err := b.svc.Delete(ctx, "/v1/collection?"+query.Encode(), &output); err != nil {
		return false, err
	}

	return output.Result, nil
}

func (b *Writing) GetCollectionInfo(ctx context.Context, input *QueryGidID) (*MessageOutput, error) {
	output := SuccessResponse[MessageOutput]{}
	query := url.Values{}
	query.Add("id", input.ID.String())
	query.Add("gid", input.GID.String())
	if input.Fields != "" {
		query.Add("fields", input.Fields)
	}
	if err := b.svc.Get(ctx, "/v1/collection/info?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Writing) UpdateCollectionInfo(ctx context.Context, input *UpdateMessageInput) (*MessageOutput, error) {
	output := SuccessResponse[MessageOutput]{}
	if err := b.svc.Patch(ctx, "/v1/collection/info", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Writing) UpdateCollectionStatus(ctx context.Context, input *UpdateStatusInput) (*CollectionOutput, error) {
	output := SuccessResponse[CollectionOutput]{}
	if err := b.svc.Patch(ctx, "/v1/collection/update_status", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type AddCollectionChildrenInput struct {
	ID   util.ID   `json:"id" cbor:"id" validate:"required"`
	GID  util.ID   `json:"gid" cbor:"gid" validate:"required"`
	Cids []util.ID `json:"cids" cbor:"cids" validate:"gte=1,lte=100"`
	Kind int8      `json:"kind" cbor:"kind" validate:"gte=0,lte=2"`
}

func (i *AddCollectionChildrenInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) AddCollectionChildren(ctx context.Context, input *AddCollectionChildrenInput) ([]util.ID, error) {
	output := SuccessResponse[[]util.ID]{}
	if err := b.svc.Post(ctx, "/v1/collection/child", input, &output); err != nil {
		return nil, err
	}

	return output.Result, nil
}

type UpdateCollectionChildInput struct {
	ID  util.ID `json:"id" cbor:"id" validate:"required"`
	GID util.ID `json:"gid" cbor:"gid" validate:"required"`
	CID util.ID `json:"cid" cbor:"cid" validate:"required"`
	Ord float64 `json:"ord" cbor:"ord" validate:"gte=0"`
}

func (i *UpdateCollectionChildInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) UpdateCollectionChild(ctx context.Context, input *UpdateCollectionChildInput) (bool, error) {
	output := SuccessResponse[bool]{}
	if err := b.svc.Patch(ctx, "/v1/collection/child", input, &output); err != nil {
		return false, err
	}

	return output.Result, nil
}

func (b *Writing) RemoveCollectionChild(ctx context.Context, input *QueryGidIdCid) (bool, error) {
	output := SuccessResponse[bool]{}

	query := url.Values{}
	query.Add("id", input.ID.String())
	query.Add("gid", input.GID.String())
	query.Add("cid", input.CID.String())
	if err := b.svc.Delete(ctx, "/v1/collection/child?"+query.Encode(), &output); err != nil {
		return false, err
	}

	return output.Result, nil
}

func (b *Writing) InternalGetCollectionSubscription(ctx context.Context, id util.ID) (*SubscriptionOutput, error) {
	output := SuccessResponse[SubscriptionOutput]{}
	query := url.Values{}
	query.Add("id", id.String())
	if err := b.svc.Get(ctx, "/v1/collection/subscription?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Writing) InternalUpdateCollectionSubscription(ctx context.Context, input *SubscriptionInput) (*SubscriptionOutput, error) {
	output := SuccessResponse[SubscriptionOutput]{}
	if err := b.svc.Put(ctx, "/v1/collection/subscription", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Writing) ListCollection(ctx context.Context, input *GIDPagination) (*SuccessResponse[CollectionOutputs], error) {
	output := SuccessResponse[CollectionOutputs]{}
	if err := b.svc.Post(ctx, "/v1/collection/list", input, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

type CollectionChildrenOutput struct {
	Parent    util.ID `json:"parent" cbor:"parent"`
	GID       util.ID `json:"gid" cbor:"gid"`
	CID       util.ID `json:"cid" cbor:"cid"`
	Kind      int8    `json:"kind" cbor:"kind"`
	Ord       float64 `json:"ord" cbor:"ord"`
	Status    int8    `json:"status" cbor:"status"`
	Rating    int8    `json:"rating" cbor:"rating"`
	UpdatedAt int64   `json:"updated_at" cbor:"updated_at"`
	Cover     string  `json:"cover" cbor:"cover"`
	Price     int64   `json:"price" cbor:"price"`
	Language  string  `json:"language" cbor:"language"`
	Title     string  `json:"title" cbor:"title"`
	Summary   string  `json:"summary" cbor:"summary"`
}

func (b *Writing) ListCollectionChildren(ctx context.Context, input *IDGIDPagination) (*SuccessResponse[[]CollectionChildrenOutput], error) {
	output := SuccessResponse[[]CollectionChildrenOutput]{}
	if err := b.svc.Post(ctx, "/v1/collection/list_children", input, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

func (b *Writing) ListCollectionByChild(ctx context.Context, input *QueryGidCid) (*SuccessResponse[CollectionOutputs], error) {
	output := SuccessResponse[CollectionOutputs]{}
	query := url.Values{}
	query.Add("gid", input.GID.String())
	query.Add("cid", input.CID.String())
	query.Add("status", strconv.Itoa(int(input.Status)))
	query.Add("fields", input.Fields)
	if err := b.svc.Get(ctx, "/v1/collection/list_by_child?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output, nil
}
