package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Group struct {
	blls *bll.Blls
}

func (a *Group) ListMy(ctx *gear.Context) error {
	groups, err := a.blls.Userbase.MyGroups(ctx)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
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

func (a *Group) GetInfo(ctx *gear.Context) error {
	input := &bll.QueryIdCn{}
	err := ctx.ParseURL(input)
	if err != nil {
		return err
	}

	res, err := a.blls.Userbase.GroupInfo(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if res.MyRole == nil {
		res.MyRole = util.Ptr(int8(-2))
	}
	if res.Following == nil {
		res.Following = util.Ptr(false)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.GroupInfo]{Result: res})
}

func (a *Group) UploadPicture(ctx *gear.Context) error {
	input := &bll.QueryIdCn{}
	err := ctx.ParseURL(input)
	if err != nil {
		return err
	}

	if input.ID == nil {
		return gear.ErrBadRequest.WithMsgf("missing group id")
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, *input.ID)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if role < 1 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	output := a.blls.Userbase.SignPicturePolicy(*input.ID)
	return ctx.OkSend(bll.SuccessResponse[service.PostFilePolicy]{Result: output})
}

func (a *Group) UpdateInfo(ctx *gear.Context) error {
	input := &bll.UpdateGroupInfoInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, input.ID)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if role < 1 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	output, err := a.blls.Userbase.UpdateGroupInfo(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[*bll.GroupInfo]{Result: output})
}

type GroupStatisticOutput struct {
	Publications uint `json:"publications" cbor:"publications"`
	Members      uint `json:"members" cbor:"members"`
}

func (a *Group) GetStatistic(ctx *gear.Context) error {
	input := &bll.QueryIdCn{}
	err := ctx.ParseURL(input)
	if err != nil {
		return err
	}

	if input.ID == nil {
		return gear.ErrBadRequest.WithMsgf("missing group id")
	}

	res := &GroupStatisticOutput{}

	res.Publications, err = a.blls.Writing.CountPublicationPublish(ctx, &bll.GIDPagination{
		GID: *input.ID,
	})

	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*GroupStatisticOutput]{Result: res})
}
