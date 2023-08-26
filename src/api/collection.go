package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Collection struct {
	blls *bll.Blls
}

func (a *Collection) Create(ctx *gear.Context) error {
	input := &bll.CreateCollectionInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	output, err := a.blls.Writing.CreateCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionUserCollect, 1, sess.UserID, &bll.Payload{
		GID:      input.GID,
		CID:      input.CID,
		Language: &input.Language,
		Version:  &input.Version,
	}); err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: output})
}

func (a *Collection) Update(ctx *gear.Context) error {
	input := &bll.UpdateCollectionInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: output})
}

func (a *Collection) Delete(ctx *gear.Context) error {
	input := &bll.QueryCollection{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.DeleteCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Collection) List(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.ListCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Collection) GetByCid(ctx *gear.Context) error {
	input := &bll.QueryCollectionByCid{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.GetCollectionByCid(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}
