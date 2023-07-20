package api

import (
	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Creation struct {
	blls *bll.Blls
}

func (a *Creation) Create(ctx *gear.Context) error {
	input := &bll.CreateCreationInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	doc := &content.DocumentNode{}
	if err := cbor.Unmarshal([]byte(input.Content), doc); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, input.GID)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}
	if role < 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	te, err := a.blls.Jarvis.DetectLang(ctx, bll.DetectLangInput{
		GID:      input.GID,
		Language: input.Language,
		Content:  doc.ToTEContents(),
	})
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	input.Language = te.Language
	output, err := a.blls.Writing.CreateCreation(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) Get(ctx *gear.Context) error {
	input := &bll.QueryCreation{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkCreationReadPermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.GetCreation(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) Update(ctx *gear.Context) error {
	input := &bll.UpdateCreationInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	creation, err := a.checkCreationWritePermission(ctx, input.GID, input.ID)
	if err != nil {
		return err
	}

	if *creation.Status != 0 && *creation.Status != 1 {
		return gear.ErrBadRequest.WithMsg("cannot delete creation, status is not 0 or 1")
	}

	output, err := a.blls.Writing.UpdateCreation(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) Delete(ctx *gear.Context) error {
	input := &bll.QueryCreation{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	creation, err := a.checkCreationWritePermission(ctx, input.GID, input.ID)
	if err != nil {
		return err
	}

	if *creation.Status != -1 {
		return gear.ErrBadRequest.WithMsg("cannot delete creation, status is not -1")
	}

	output, err := a.blls.Writing.DeleteCreation(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Creation) List(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkCreationReadPermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.ListCreation(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Creation) Archive(ctx *gear.Context) error {
	input := &bll.UpdateCreationStatusInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	_, err := a.checkCreationWritePermission(ctx, input.GID, input.ID)
	if err != nil {
		return err
	}

	input.Status = -1
	output, err := a.blls.Writing.UpdateCreationStatus(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) Redraft(ctx *gear.Context) error {
	input := &bll.UpdateCreationStatusInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	_, err := a.checkCreationWritePermission(ctx, input.GID, input.ID)
	if err != nil {
		return err
	}

	input.Status = 0
	output, err := a.blls.Writing.UpdateCreationStatus(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) Release(ctx *gear.Context) error {
	input := &bll.CreatePublicationInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	creation, err := a.checkCreationWritePermission(ctx, input.GID, input.CID)
	if err != nil {
		return err
	}

	if *creation.Status != 2 {
		return gear.ErrBadRequest.WithMsg("cannot delete creation, status is not 2")
	}

	input.Draft = nil
	output, err := a.blls.Writing.CreatePublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Creation) UpdateContent(ctx *gear.Context) error {
	input := &bll.UpdateCreationContentInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	creation, err := a.checkCreationWritePermission(ctx, input.GID, input.ID)
	if err != nil {
		return err
	}

	if *creation.Status != 0 && *creation.Status != 1 {
		return gear.ErrBadRequest.WithMsg("cannot update creation content, status is not 0 or 1")
	}

	output, err := a.blls.Writing.UpdateCreationContent(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) checkCreationReadPermission(ctx *gear.Context, gid util.ID) error {
	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if role < -1 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	return nil
}

func (a *Creation) checkCreationWritePermission(ctx *gear.Context, gid, cid util.ID) (*bll.CreationOutput, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}
	if role < 0 {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	creation, err := a.blls.Writing.GetCreation(ctx, &bll.QueryCreation{
		GID:    gid,
		ID:     cid,
		Fields: "status,creator",
	})

	if err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}
	if creation.Creator == nil || creation.Status == nil {
		return nil, gear.ErrInternalServerError.WithMsg("invalid creation")
	}

	if role < 1 && *creation.Creator != sess.UserID {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	return creation, nil
}
