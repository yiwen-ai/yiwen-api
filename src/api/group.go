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

func (a *Group) Follow(ctx *gear.Context) error {
	input := &bll.QueryIdCn{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	output, err := a.blls.Userbase.FollowGroup(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Group) UnFollow(ctx *gear.Context) error {
	input := &bll.QueryIdCn{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	output, err := a.blls.Userbase.UnFollowGroup(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Group) ListFollowing(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}
	output, err := a.blls.Userbase.ListFollowing(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bll.Groups]{Result: output})
}
