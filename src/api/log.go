package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
)

type Log struct {
	blls *bll.Blls
}

func (a *Log) ListRecently(ctx *gear.Context) error {
	sess := gear.CtxValue[middleware.Session](ctx)
	output, err := a.blls.Logbase.ListRecently(ctx, &bll.ListRecentlyLogsInput{
		UID:     sess.UserID,
		Actions: []string{},
		Fields:  []string{},
	})
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[[]bll.LogOutput]{Result: output})
}
