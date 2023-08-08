package bll

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Webscraper struct {
	svc service.APIHost
}

type ScrapingInput struct {
	GID util.ID `json:"gid" cbor:"gid" query:"gid" validate:"required"`
	Url string  `json:"url" cbor:"url" query:"url" validate:"required,http_url"`
}

func (i *ScrapingInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type ScrapingOutput struct {
	ID      util.ID           `json:"id" cbor:"id"`
	Url     string            `json:"url" cbor:"url"`
	Src     string            `json:"src,omitempty" cbor:"src,omitempty"`
	Title   string            `json:"title,omitempty" cbor:"title,omitempty"`
	Meta    map[string]string `json:"meta,omitempty" cbor:"meta,omitempty"`
	Content util.Bytes        `json:"content,omitempty" cbor:"content,omitempty"`
}

func (b *Webscraper) Search(ctx context.Context, targetUrl string) (*ScrapingOutput, error) {
	output := SuccessResponse[ScrapingOutput]{}
	api := fmt.Sprintf("/v1/search?url=%s", url.QueryEscape(targetUrl))
	if err := b.svc.Get(ctx, api, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Webscraper) Create(ctx context.Context, targetUrl string) (*ScrapingOutput, error) {
	output := SuccessResponse[ScrapingOutput]{}
	api := fmt.Sprintf("/v1/scraping?url=%s", url.QueryEscape(targetUrl))
	if err := b.svc.Get(ctx, api, &output); err != nil {
		return nil, err
	}

	retry := 1
	if output.Retry < 10 {
		retry = output.Retry
	}
	time.Sleep(time.Duration(retry) * time.Second)
	api = fmt.Sprintf("/v1/document?id=%s&output=detail", output.Result.ID.String())
	i := 0
	for {
		i += 1
		if err := b.svc.Get(ctx, api, &output); err != nil {
			return nil, err
		}

		if len(output.Result.Content) > 0 || i > 15 {
			break
		}
		time.Sleep(time.Second)
	}

	return &output.Result, nil
}

func (b *Webscraper) Convert(ctx context.Context, file []byte, mtype string) (*util.Bytes, error) {
	output := SuccessResponse[util.Bytes]{}
	if err := b.svc.Post(ctx, "/v1/converting", file, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}
