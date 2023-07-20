package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
)

type Scraping struct {
	blls *bll.Blls
}

func (a *Scraping) Create(ctx *gear.Context) error {
	input := bll.ScrapingInput{}
	if err := ctx.ParseURL(&input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, input.GID)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}
	if role < 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	output, err := a.blls.Webscraper.Create(ctx, input.Url)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bll.ScrapingOutput]{Result: *output})
}
