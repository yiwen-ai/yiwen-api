package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/yiwen-ai/yiwen-api/src/util"
)

type SearchInput struct {
	Q        string   `json:"q" cbor:"q" query:"q" validate:"required"`
	Language string   `json:"language" cbor:"language" query:"language"`
	GID      *util.ID `json:"gid" cbor:"gid" query:"gid"`
}

type SearchDocument struct {
	GID      util.ID    `json:"gid" cbor:"gid"`
	CID      util.ID    `json:"cid" cbor:"cid"`
	Language string     `json:"language" cbor:"language"`
	Version  uint16     `json:"version" cbor:"version"`
	Kind     int8       `json:"kind" cbor:"kind"`
	Title    string     `json:"title" cbor:"title"`
	Summary  string     `json:"summary" cbor:"summary"`
	Group    *GroupInfo `json:"group,omitempty" cbor:"group,omitempty"`
}

type SearchOutput struct {
	Hits      []SearchDocument `json:"hits" cbor:"hits"`
	Languages map[string]int   `json:"languages" cbor:"languages"`
}

func OriginalSearch(ctx context.Context, gid, targetUrl string) (*SearchOutput, error) {
	output := SuccessResponse[SearchOutput]{}
	query := url.Values{}
	query.Add("gid", gid)
	query.Add("url", targetUrl)

	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "GET", apiHost+"/v1/search/by_original_url?"+query.Encode(), nil, &output); err != nil {
		return nil, err
	}

	joutput := output.Result
	data, err := json.Marshal(joutput)
	if err != nil {
		return nil, err
	}
	fmt.Printf("OriginalSearch: %s\n", string(data))
	return &output.Result, nil
}

func GroupSearch(ctx context.Context, q, gid, language string) (*SearchOutput, error) {
	output := SuccessResponse[SearchOutput]{}
	query := url.Values{}
	query.Add("q", q)
	query.Add("gid", gid)
	if language != "" {
		query.Add("language", language)
	}
	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "GET", apiHost+"/v1/search/in_group?"+query.Encode(), nil, &output); err != nil {
		return nil, err
	}

	joutput := output.Result
	data, err := json.Marshal(joutput)
	if err != nil {
		return nil, err
	}
	fmt.Printf("GroupSearch: %s\n", string(data))
	return &output.Result, nil
}
