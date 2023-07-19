package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
)

type Scraping struct {
	blls *bll.Blls
}

func (a *Scraping) Create(ctx *gear.Context) error {
	targetUrl := ctx.Query("url")
	output, err := a.blls.Webscraper.Create(ctx, targetUrl)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bll.ScrapingOutput]{Result: *output})
}
