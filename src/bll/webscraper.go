package bll

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Webscraper struct {
	svc service.APIHost
}

type ScrapingOutput struct {
	ID      util.ID           `json:"id" cbor:"id"`
	Url     string            `json:"url" cbor:"url"`
	Src     string            `json:"src" cbor:"src"`
	Title   string            `json:"title" cbor:"title"`
	Meta    map[string]string `json:"meta" cbor:"meta"`
	Content util.Raw          `json:"content" cbor:"content"`
}

func (b *Webscraper) Create(ctx context.Context, targetUrl string) (*ScrapingOutput, error) {
	output := SuccessResponse[ScrapingOutput]{}
	api := fmt.Sprintf("/v1/scraping?url=%s", url.QueryEscape(targetUrl))
	if err := b.svc.Get(ctx, api, &output); err != nil {
		return nil, err
	}

	time.Sleep(time.Duration(output.Retry) * time.Second)
	api = fmt.Sprintf("/v1/document?id=%s&output=detail", output.Result.ID.String())
	i := 0
	for {
		i += 1
		if err := b.svc.Get(ctx, api, &output); err != nil {
			return nil, err
		}

		if len(output.Result.Content) > 0 || i > 10 {
			break
		}
		time.Sleep(time.Second)
	}

	return &output.Result, nil
}
