package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Group struct {
	blls *bll.Blls
}

func (a *Group) ListMy(ctx *gear.Context) error {
	output, err := a.blls.Userbase.MyGroups(ctx)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	groups := bll.Groups(output)
	(&groups).LoadUsers(func(ids ...util.ID) []bll.UserInfo {
		return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	})
	return ctx.OkSend(bll.SuccessResponse[bll.Groups]{Result: groups})
}
