package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
)

type Creation struct {
	blls *bll.Blls
}

func (a *Creation) Create(ctx *gear.Context) error {
	// idp := ctx.Param("idp")
	// xid := ctx.GetHeader(gear.HeaderXRequestID)

	// nextURL, ok := a.authURL.CheckNextUrl(ctx.Query("next_url"))
	// if !ok {
	// 	next := a.authURL.GenNextUrl(&nextURL, 400, xid)
	// 	logging.SetTo(ctx, "error", fmt.Sprintf("invalid next_url %q", ctx.Query("next_url")))
	// 	return ctx.Redirect(next)
	// }

	// provider, ok := a.providers[idp]
	// if !ok {
	// 	next := a.authURL.GenNextUrl(&nextURL, 400, xid)
	// 	logging.SetTo(ctx, "error", fmt.Sprintf("unknown provider %q", idp))
	// 	return ctx.Redirect(next)
	// }

	// state, err := a.createState(idp, provider.ClientID, nextURL.String())
	// if err != nil {
	// 	next := a.authURL.GenNextUrl(&nextURL, 500, xid)
	// 	logging.SetTo(ctx, "error", fmt.Sprintf("failed to create state: %v", err))
	// 	return ctx.Redirect(next)
	// }

	// url := provider.AuthCodeURL(state)
	return nil
}
