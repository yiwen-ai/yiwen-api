package api

import (
	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Publication struct {
	blls *bll.Blls
}

func (a *Publication) Create(ctx *gear.Context) error {
	input := &bll.CreatePublicationInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if input.Draft == nil {
		return gear.ErrBadRequest.WithMsg("draft is required")
	}

	doc := &content.DocumentNode{}
	if err := cbor.Unmarshal([]byte(input.Draft.Content), doc); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	if err := a.checkCreatePermission(ctx, input.GID); err != nil {
		return err
	}

	// TODO: AI
	output, err := a.blls.Writing.CreatePublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) Get(ctx *gear.Context) error {
	input := &bll.QueryPublication{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.GetPublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) Update(ctx *gear.Context) error {
	input := &bll.UpdatePublicationInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	publication, err := a.checkWritePermission(ctx, input.GID, input.ID, input.Language, input.Version)
	if err != nil {
		return err
	}

	if *publication.Status != 0 {
		return gear.ErrBadRequest.WithMsg("cannot update publication, status is not 0")
	}

	output, err := a.blls.Writing.UpdatePublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) Delete(ctx *gear.Context) error {
	input := &bll.QueryPublication{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	creation, err := a.checkWritePermission(ctx, input.GID, input.CID, input.Language, input.Version)
	if err != nil {
		return err
	}

	if *creation.Status != -1 {
		return gear.ErrBadRequest.WithMsg("cannot delete publication, status is not -1")
	}

	output, err := a.blls.Writing.DeletePublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Publication) List(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.ListPublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Publication) ListArchived(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	input.Status = bll.Int8Ptr(-1)
	output, err := a.blls.Writing.ListPublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Publication) ListPublished(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	input.Status = bll.Int8Ptr(2)
	output, err := a.blls.Writing.ListPublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Publication) GetPublishList(ctx *gear.Context) error {
	input := &bll.QueryAPublication{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		input.GID = util.ANON
	}

	output, err := a.blls.Writing.GetPublicationList(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Publication) Archive(ctx *gear.Context) error {
	input := &bll.UpdatePublicationStatusInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	publication, err := a.checkWritePermission(ctx, input.GID, input.CID, input.Language, input.Version)
	if err != nil {
		return err
	}

	if *publication.Status != 0 && *publication.Status != 1 {
		return gear.ErrBadRequest.WithMsg("cannot update publication, status is not 0 or 1")
	}

	input.Status = -1
	output, err := a.blls.Writing.UpdatePublicationStatus(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) Redraft(ctx *gear.Context) error {
	input := &bll.UpdatePublicationStatusInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	publication, err := a.checkWritePermission(ctx, input.GID, input.CID, input.Language, input.Version)
	if err != nil {
		return err
	}

	if *publication.Status != -1 && *publication.Status != 1 {
		return gear.ErrBadRequest.WithMsg("cannot update publication, status is not -1 or 1")
	}

	input.Status = 0
	output, err := a.blls.Writing.UpdatePublicationStatus(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) Publish(ctx *gear.Context) error {
	input := &bll.UpdatePublicationStatusInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	publication, err := a.checkWritePermission(ctx, input.GID, input.CID, input.Language, input.Version)
	if err != nil {
		return err
	}

	if *publication.Status != 1 {
		return gear.ErrBadRequest.WithMsg("cannot update publication, status is not 1")
	}

	input.Status = 2
	output, err := a.blls.Writing.UpdatePublicationStatus(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) UpdateContent(ctx *gear.Context) error {
	input := &bll.UpdatePublicationContentInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	publication, err := a.checkWritePermission(ctx, input.GID, input.CID, input.Language, input.Version)
	if err != nil {
		return err
	}

	if *publication.Status != 0 {
		return gear.ErrBadRequest.WithMsg("cannot update publication content, status is not 0 or 1")
	}

	output, err := a.blls.Writing.UpdatePublicationContent(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) checkReadPermission(ctx *gear.Context, gid util.ID) error {
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

func (a *Publication) checkCreatePermission(ctx *gear.Context, gid util.ID) error {
	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if role < 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	return nil
}

func (a *Publication) checkWritePermission(ctx *gear.Context, gid, cid util.ID, language string, version int16) (*bll.PublicationOutput, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}
	if role < 0 {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	publication, err := a.blls.Writing.GetPublication(ctx, &bll.QueryPublication{
		GID:      gid,
		CID:      cid,
		Language: language,
		Version:  version,
		Fields:   "status,creator,updated_at",
	})

	if err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}
	if publication.Creator == nil || publication.Status == nil {
		return nil, gear.ErrInternalServerError.WithMsg("invalid publication")
	}

	if role < 1 && *publication.Creator != sess.UserID {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	return publication, nil
}
