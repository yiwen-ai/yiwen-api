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

	if err := a.checkCreatePermission(ctx, input.GID); err != nil {
		return err
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

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
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

	creation, err := a.checkWritePermission(ctx, input.GID, input.ID)
	if err != nil {
		return err
	}

	if *creation.Status != 0 && *creation.Status != 1 {
		return gear.ErrBadRequest.WithMsg("cannot update creation, status is not 0 or 1")
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

	creation, err := a.checkWritePermission(ctx, input.GID, input.ID)
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

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.ListCreation(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Creation) ListArchived(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	input.Status = bll.Int8Ptr(-1)
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

	_, err := a.checkWritePermission(ctx, input.GID, input.ID)
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

	_, err := a.checkWritePermission(ctx, input.GID, input.ID)
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

	creation, err := a.checkWritePermission(ctx, input.GID, input.CID)
	if err != nil {
		return err
	}

	if *creation.Status != 2 {
		sess := gear.CtxValue[middleware.Session](ctx)
		if sess.UserID != input.GID {
			return gear.ErrBadRequest.WithMsg("cannot release creation, status is not 2")
		}
		if *creation.Status == -1 {
			return gear.ErrBadRequest.WithMsg("cannot release creation, status is -1")
		}

		// 用户私有 group 自动提升 status，无需 review 和 approve
		statusInput := &bll.UpdateCreationStatusInput{
			GID: input.GID,
			ID:  input.CID,
		}

		if *creation.Status == 0 {
			statusInput.Status = 1
			statusInput.UpdatedAt = *creation.UpdatedAt
			output, err := a.blls.Writing.UpdateCreationStatus(ctx, statusInput)
			if err != nil {
				return gear.ErrInternalServerError.From(err)
			}
			creation.Status = output.Status
			creation.UpdatedAt = output.UpdatedAt
		}

		if *creation.Status == 1 {
			statusInput.Status = 2
			statusInput.UpdatedAt = *creation.UpdatedAt
			output, err := a.blls.Writing.UpdateCreationStatus(ctx, statusInput)
			if err != nil {
				return gear.ErrInternalServerError.From(err)
			}
			creation.Status = output.Status
			creation.UpdatedAt = output.UpdatedAt
		}
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

	creation, err := a.checkWritePermission(ctx, input.GID, input.ID)
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

func (a *Creation) checkReadPermission(ctx *gear.Context, gid util.ID) error {
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

func (a *Creation) checkCreatePermission(ctx *gear.Context, gid util.ID) error {
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

func (a *Creation) checkWritePermission(ctx *gear.Context, gid, cid util.ID) (*bll.CreationOutput, error) {
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
		Fields: "status,creator,updated_at",
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
