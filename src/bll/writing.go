package bll

import (
	"context"
	"net/url"

	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Writing struct {
	svc             service.APIHost
	oss             *service.OSS
	Recommendations []PublicationOutputs
}

type SearchInput struct {
	Q        string   `json:"q" cbor:"q" query:"q" validate:"required"`
	Language *string  `json:"language,omitempty" cbor:"language,omitempty" query:"language"`
	GID      *util.ID `json:"gid,omitempty" cbor:"gid,omitempty" query:"gid"`
}

func (i *SearchInput) Validate() error {
	return nil
}

type SearchDocument struct {
	GID       util.ID    `json:"gid" cbor:"gid"`
	CID       util.ID    `json:"cid" cbor:"cid"`
	Language  string     `json:"language" cbor:"language"`
	Version   uint16     `json:"version" cbor:"version"`
	UpdatedAt int64      `json:"updated_at" cbor:"updated_at"`
	Kind      int8       `json:"kind" cbor:"kind"`
	Title     string     `json:"title" cbor:"title"`
	Summary   string     `json:"summary" cbor:"summary"`
	GroupInfo *GroupInfo `json:"group_info,omitempty" cbor:"group_info,omitempty"`
}

type SearchOutput struct {
	Hits      []SearchDocument `json:"hits" cbor:"hits"`
	Languages map[string]int   `json:"languages" cbor:"languages"`
}

func (so *SearchOutput) LoadGroups(loader func(ids ...util.ID) []GroupInfo) {
	if len(so.Hits) == 0 {
		return
	}

	ids := make([]util.ID, 0, len(so.Hits))
	for _, hit := range so.Hits {
		ids = append(ids, hit.GID)
	}
	groups := loader(ids...)
	if len(groups) == 0 {
		return
	}

	infoMap := make(map[util.ID]*GroupInfo, len(groups))
	for i := range groups {
		infoMap[groups[i].ID] = &groups[i]
	}

	for i := range so.Hits {
		so.Hits[i].GroupInfo = infoMap[so.Hits[i].GID]
	}
}

func (b *Writing) Search(ctx context.Context, input *SearchInput) SearchOutput {
	output := SuccessResponse[SearchOutput]{Result: SearchOutput{
		Hits:      []SearchDocument{},
		Languages: map[string]int{},
	}}

	query := url.Values{}
	query.Add("q", input.Q)
	if input.GID != nil {
		query.Add("gid", input.GID.String())
	}
	if input.Language != nil && *input.Language != "" {
		query.Add("language", *input.Language)
	}

	// ignore error
	_ = b.svc.Get(ctx, "/v1/search?"+query.Encode(), &output)

	return output.Result
}

func (b *Writing) GroupSearch(ctx context.Context, input *SearchInput) SearchOutput {
	output := SuccessResponse[SearchOutput]{Result: SearchOutput{
		Hits:      []SearchDocument{},
		Languages: map[string]int{},
	}}

	query := url.Values{}
	query.Add("q", input.Q)
	query.Add("gid", input.GID.String())
	if input.Language != nil && *input.Language != "" {
		query.Add("language", *input.Language)
	}

	// ignore error
	_ = b.svc.Get(ctx, "/v1/search/in_group?"+query.Encode(), &output)

	return output.Result
}

func (b *Writing) OriginalSearch(ctx context.Context, input *ScrapingInput) SearchOutput {
	output := SuccessResponse[SearchOutput]{Result: SearchOutput{
		Hits:      []SearchDocument{},
		Languages: map[string]int{},
	}}

	query := url.Values{}
	query.Add("q", input.Url)
	query.Add("gid", input.GID.String())

	// ignore error
	_ = b.svc.Get(ctx, "/v1/search/by_original_url?"+query.Encode(), &output)

	return output.Result
}

func (b *Writing) SignPostPolicy(gid, cid util.ID, lang string, version uint) service.PostFilePolicy {
	return b.oss.SignPostPolicy(gid.String(), cid.String(), lang, version)
}
