package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/yiwen-ai/yiwen-api/src/util"
)

type ScrapingOutput struct {
	ID      util.ID           `json:"id" cbor:"id"`
	Url     string            `json:"url" cbor:"url"`
	Src     string            `json:"src" cbor:"src"`
	Title   string            `json:"title" cbor:"title"`
	Meta    map[string]string `json:"meta" cbor:"meta"`
	Content util.Bytes        `json:"content" cbor:"content"`
}

func GetWeb(ctx context.Context, gid, targetUrl string) (*ScrapingOutput, error) {
	output := SuccessResponse[ScrapingOutput]{}
	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "GET", fmt.Sprintf("%s/v1/scraping?gid=%s&url=%s", apiHost, gid, url.QueryEscape(targetUrl)), nil, &output); err != nil {
		return nil, err
	}

	joutput := output.Result
	joutput.Content = nil
	data, err := json.Marshal(joutput)
	if err != nil {
		return nil, err
	}
	fmt.Printf("GetWeb: %s, length: %d\n", string(data), len(output.Result.Content))
	return &output.Result, nil
}
