package bll

import (
	"context"
	"net/url"
	"strconv"

	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

// TODO: more validation
type CreatePublicationInput struct {
	GID           util.ID  `json:"gid" cbor:"gid" validate:"required"`
	CID           util.ID  `json:"cid" cbor:"cid" validate:"required"`
	Language      string   `json:"language" cbor:"language" validate:"required"`
	Version       uint16   `json:"version" cbor:"version" validate:"gte=1,lte=10000"`
	Model         string   `json:"model" cbor:"model" validate:"omitempty,gte=2,lte=16"`
	ToGID         *util.ID `json:"to_gid,omitempty" cbor:"to_gid,omitempty"`
	ToLanguage    *string  `json:"to_language,omitempty" cbor:"to_language,omitempty"`
	Context       *string  `json:"context,omitempty" cbor:"context,omitempty"` // Contextual definition for translating
	ContentFilter *bool    `json:"content_filter,omitempty" cbor:"content_filter,omitempty"`
}

func (i *CreatePublicationInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}
	if i.Context != nil {
		if tk := util.Tiktokens(*i.Context); tk > 2048 {
			return gear.ErrBadRequest.WithMsgf("context is too long, max tokens is 2048, got %d", tk)
		}
	}

	return nil
}

type CreatePublication struct {
	GID      util.ID           `json:"gid" cbor:"gid"`
	CID      util.ID           `json:"cid" cbor:"cid"`
	Language string            `json:"language" cbor:"language"`
	Version  uint16            `json:"version" cbor:"version"`
	Draft    *PublicationDraft `json:"draft,omitempty" cbor:"draft,omitempty"`
}

type PublicationDraft struct {
	GID      util.ID    `json:"gid" cbor:"gid"`
	Language string     `json:"language" cbor:"language"`
	Title    string     `json:"title" cbor:"title"`
	Model    string     `json:"model" cbor:"model"`
	Cover    string     `json:"cover" cbor:"cover"`
	Keywords []string   `json:"keywords" cbor:"keywords"`
	Summary  string     `json:"summary" cbor:"summary"`
	Content  util.Bytes `json:"content" cbor:"content"`
}

type PublicationOutput struct {
	GID          util.ID             `json:"gid" cbor:"gid"`
	CID          util.ID             `json:"cid" cbor:"cid"`
	Language     string              `json:"language" cbor:"language"`
	Version      uint16              `json:"version" cbor:"version"`
	Rating       *int8               `json:"rating,omitempty" cbor:"rating,omitempty"`
	Price        *int64              `json:"price,omitempty" cbor:"price,omitempty"`
	Status       *int8               `json:"status,omitempty" cbor:"status,omitempty"`
	Creator      *util.ID            `json:"creator,omitempty" cbor:"creator,omitempty"`
	CreatedAt    *int64              `json:"created_at,omitempty" cbor:"created_at,omitempty"`
	UpdatedAt    *int64              `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	Model        *string             `json:"model,omitempty" cbor:"model,omitempty"`
	OriginalUrl  *string             `json:"original_url,omitempty" cbor:"original_url,omitempty"`
	FromLanguage *string             `json:"from_language,omitempty" cbor:"from_language,omitempty"`
	Genre        *[]string           `json:"genre,omitempty" cbor:"genre,omitempty"`
	Title        *string             `json:"title,omitempty" cbor:"title,omitempty"`
	Cover        *string             `json:"cover,omitempty" cbor:"cover,omitempty"`
	Keywords     *[]string           `json:"keywords,omitempty" cbor:"keywords,omitempty"`
	Authors      *[]string           `json:"authors,omitempty" cbor:"authors,omitempty"`
	Summary      *string             `json:"summary,omitempty" cbor:"summary,omitempty"`
	Content      *util.Bytes         `json:"content,omitempty" cbor:"content,omitempty"`
	License      *string             `json:"license,omitempty" cbor:"license,omitempty"`
	Subscription *SubscriptionOutput `json:"subscription,omitempty" cbor:"subscription,omitempty"`
	RFP          *RFP                `json:"rfp,omitempty" cbor:"rfp,omitempty"`
	FromGID      *util.ID            `json:"from_gid,omitempty" cbor:"from_gid,omitempty"`
	GroupInfo    *GroupInfo          `json:"group_info,omitempty" cbor:"group_info,omitempty"`
}

type PublicationOutputs []PublicationOutput

func (list *PublicationOutputs) LoadGroups(loader func(ids ...util.ID) []GroupInfo) {
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

func (list PublicationOutputs) PreferVersion(lang string) *PublicationOutput {
	if len(list) == 0 {
		return nil
	}

	for i := range list {
		if list[i].Language == lang {
			return &list[i]
		}
	}

	for i := range list {
		if list[i].FromLanguage != nil && list[i].Language == *list[i].FromLanguage {
			return &list[i]
		}
	}

	return &list[0]
}

func (i *PublicationOutput) ToTEContents() (content.TEContents, error) {
	if i.Title == nil || i.Summary == nil || i.Content == nil {
		return nil, gear.ErrInternalServerError.WithMsg("empty title or summary or content")
	}
	doc, err := content.ParseDocumentNode(*i.Content)
	if err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}

	contents := content.TEContents{&content.TEContent{
		ID:    "title",
		Texts: []string{*i.Title},
	}, &content.TEContent{
		ID:    "summary",
		Texts: []string{*i.Summary},
	}}
	if i.Keywords != nil && len(*i.Keywords) > 0 {
		contents = append(contents, &content.TEContent{
			ID:    "keywords",
			Texts: *i.Keywords,
		})
	}
	contents = append(contents, doc.ToTEContents()...)
	return contents, nil
}

func (i *PublicationOutput) IntoPublicationDraft(gid util.ID, language, model string, input []byte) (*PublicationDraft, error) {
	draft := &PublicationDraft{
		GID:      gid,
		Language: language,
		Model:    model,
		Keywords: []string{},
		Title:    *i.Title,
		Summary:  *i.Summary,
	}
	if i.Keywords != nil && len(*i.Keywords) > 0 {
		draft.Keywords = *i.Keywords
	}
	if i.Cover != nil {
		draft.Cover = *i.Cover
	}

	teContents := content.TEContents{}
	if err := cbor.Unmarshal(input, &teContents); err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}

	for _, te := range teContents {
		switch te.ID {
		case "title":
			if len(te.Texts) > 0 {
				draft.Title = te.Texts[0]
			}
		case "summary":
			if len(te.Texts) > 0 {
				draft.Summary = te.Texts[0]
			}
		case "keywords":
			if len(te.Texts) > 0 {
				draft.Keywords = te.Texts
			}
		}
	}

	doc := &content.DocumentNode{}
	if err := cbor.Unmarshal([]byte(*i.Content), doc); err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}

	doc.FromTEContents(teContents)
	data, err := cbor.Marshal(doc)
	if err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}
	draft.Content = data
	return draft, nil
}

func (b *Writing) InitApp(ctx context.Context, _ *gear.App) error {
	for _, v := range conf.Config.Recommendations {
		res, err := b.GetPublicationList(ctx, 2, &QueryGidCid{
			GID: v.GID,
			CID: v.CID,
		})
		if err != nil {
			return err
		}
		b.Recommendations = append(b.Recommendations, res.Result)
	}

	return nil
}

func (b *Writing) CreatePublication(ctx context.Context, input *CreatePublication) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}
	if err := b.svc.Post(ctx, "/v1/publication", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type QueryPublication struct {
	GID      util.ID `json:"gid" cbor:"gid" query:"gid" validate:"required"`
	CID      util.ID `json:"cid" cbor:"cid" query:"cid" validate:"required"`
	Language string  `json:"language" cbor:"language" query:"language" validate:"required"`
	Version  uint16  `json:"version" cbor:"version" query:"version" validate:"gte=1,lte=10000"`
	Fields   string  `json:"fields" cbor:"fields" query:"fields"`
}

func (i *QueryPublication) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type QueryJob struct {
	ID     util.ID `json:"job" cbor:"job" query:"job" validate:"required"`
	Fields string  `json:"fields" cbor:"fields" query:"fields"`
}

func (i *QueryJob) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}
	return nil
}

func (b *Writing) GetPublication(ctx context.Context, input *ImplicitQueryPublication,
	subscription_in *util.ID) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}

	query := url.Values{}
	query.Add("gid", input.GID.String())
	query.Add("cid", input.CID.String())
	query.Add("language", input.Language)
	query.Add("version", strconv.Itoa(int(input.Version)))
	if input.Parent != nil {
		query.Add("parent", input.Parent.String())
	}
	if subscription_in != nil {
		query.Add("subscription_in", subscription_in.String())
	}
	if input.Fields != "" {
		query.Add("fields", input.Fields)
	}
	if err := b.svc.Get(ctx, "/v1/publication?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type ImplicitQueryPublication struct {
	CID      util.ID  `json:"cid" cbor:"cid" query:"cid" validate:"required"`
	GID      *util.ID `json:"gid" cbor:"gid" query:"gid"`
	Parent   *util.ID `json:"parent" cbor:"parent" query:"parent"`
	Language string   `json:"language" cbor:"language" query:"language"`
	Version  uint16   `json:"version" cbor:"version" query:"version" validate:"omitempty,gte=0,lte=10000"`
	Fields   string   `json:"fields" cbor:"fields" query:"fields"`
	SubToken string   `json:"subtoken" cbor:"subtoken" query:"subtoken"`
}

func (i *ImplicitQueryPublication) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

// ImplicitGetPublication is used to get a publication.
// It will check the subscription if subscription_in privided. (ignore checking if nil)
func (b *Writing) ImplicitGetPublication(ctx context.Context, input *ImplicitQueryPublication,
	subscription_in *util.ID) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}

	query := url.Values{}
	query.Add("cid", input.CID.String())
	if input.GID != nil {
		query.Add("gid", input.GID.String())
	}
	if input.Parent != nil {
		query.Add("parent", input.Parent.String())
	}
	if subscription_in != nil {
		query.Add("subscription_in", subscription_in.String())
	}
	if input.Language != "" {
		query.Add("language", input.Language)
	}
	if input.Fields != "" {
		query.Add("fields", input.Fields)
	}
	if err := b.svc.Get(ctx, "/v1/publication/implicit_get?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type UpdatePublicationInput struct {
	GID       util.ID   `json:"gid" cbor:"gid" validate:"required"`
	CID       util.ID   `json:"cid" cbor:"cid" validate:"required"`
	Language  string    `json:"language" cbor:"language" validate:"required"`
	Version   uint16    `json:"version" cbor:"version" validate:"gte=1,lte=10000"`
	UpdatedAt int64     `json:"updated_at" cbor:"updated_at"  validate:"required"`
	Title     *string   `json:"title,omitempty" cbor:"title,omitempty" validate:"omitempty,gte=1,lte=256"`
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

func (b *Writing) ListPublication(ctx context.Context, input *GIDPagination) (*SuccessResponse[PublicationOutputs], error) {
	output := SuccessResponse[PublicationOutputs]{}
	if err := b.svc.Post(ctx, "/v1/publication/list", input, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

func (b *Writing) CountPublicationPublish(ctx context.Context, input *GIDPagination) (uint, error) {
	output := SuccessResponse[uint]{}
	if err := b.svc.Post(ctx, "/v1/publication/count_publish", input, &output); err != nil {
		return 0, err
	}

	return output.Result, nil
}

type GIDsPagination struct {
	GIDs      []util.ID   `json:"gids" cbor:"gids"`
	PageToken *util.Bytes `json:"page_token,omitempty" cbor:"page_token,omitempty"`
	PageSize  *uint16     `json:"page_size,omitempty" cbor:"page_size,omitempty"`
	Fields    *[]string   `json:"fields,omitempty" cbor:"fields,omitempty"`
}

func (i *GIDsPagination) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (b *Writing) ListPublicationByGIDs(ctx context.Context, input *GIDsPagination) (*SuccessResponse[PublicationOutputs], error) {
	output := SuccessResponse[PublicationOutputs]{}
	if err := b.svc.Post(ctx, "/v1/publication/list_by_gids", input, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

func (b *Writing) ListLatestPublications(ctx context.Context, input *Pagination) (*SuccessResponse[PublicationOutputs], error) {
	output := SuccessResponse[PublicationOutputs]{}
	if err := b.svc.Post(ctx, "/v1/publication/list_latest", input, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

func (b *Writing) GetPublicationList(ctx context.Context, from_status int8, input *QueryGidCid) (*SuccessResponse[PublicationOutputs], error) {
	output := SuccessResponse[PublicationOutputs]{}
	query := url.Values{}
	query.Add("gid", input.GID.String())
	query.Add("cid", input.CID.String())
	query.Add("status", strconv.Itoa(int(from_status)))
	if err := b.svc.Get(ctx, "/v1/publication/publish?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output, nil
}

type UpdatePublicationStatusInput struct {
	GID       util.ID `json:"gid" cbor:"gid" validate:"required"`
	CID       util.ID `json:"cid" cbor:"cid" validate:"required"`
	Language  string  `json:"language" cbor:"language" validate:"required"`
	Version   uint16  `json:"version" cbor:"version" validate:"gte=1,lte=10000"`
	UpdatedAt int64   `json:"updated_at" cbor:"updated_at" validate:"required"`
	Status    int8    `json:"status" cbor:"status" validate:"gte=-2,lte=2"`
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
	Version   uint16     `json:"version" cbor:"version" validate:"gte=1,lte=10000"`
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
