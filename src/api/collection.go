package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Bookmark struct {
	blls *bll.Blls
}

func (a *Bookmark) Update(ctx *gear.Context) error {
	input := &bll.UpdateBookmarkInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateBookmark(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.BookmarkOutput]{Result: output})
}

func (a *Bookmark) Delete(ctx *gear.Context) error {
	input := &bll.QueryBookmark{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.DeleteBookmark(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Bookmark) List(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.ListBookmark(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Bookmark) GetByCid(ctx *gear.Context) error {
	input := &bll.QueryBookmarkByCid{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.GetBookmarkByCid(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}
