package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/logging"
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

	teContents, err := content.ToTEContents([]byte(input.Content))
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}
	teData, err := cbor.Marshal(teContents)
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}

	if err := a.checkCreatePermission(ctx, input.GID); err != nil {
		return err
	}

	te, err := a.blls.Jarvis.DetectLang(ctx, &bll.DetectLangInput{
		GID:      input.GID,
		Language: input.Language,
		Content:  teData,
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
	input := &bll.GIDPagination{}
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
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	input.Status = bll.Ptr(int8(-1))
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
	input.Model = bll.DefaultModel

	creation, err := a.checkWritePermission(ctx, input.GID, input.CID)
	if err != nil {
		return err
	}
	if *creation.Status < 0 {
		return gear.ErrBadRequest.WithMsg("cannot release creation, status is -1")
	}

	gctx := middleware.WithGlobalCtx(ctx)
	key := fmt.Sprintf("RC:%s:%s", input.GID.String(), input.CID.String())
	locker, err := a.blls.Locker.Lock(gctx, key, 120*time.Second)
	if err != nil {
		return gear.ErrLocked.From(err)
	}

	sess := gear.CtxValue[middleware.Session](gctx)
	job := &bll.Job{
		GID:       creation.GID,
		CID:       creation.ID,
		Language:  *creation.Language,
		Version:   *creation.Version,
		ExpiresIn: time.Now().Unix() + 120,
	}
	go logging.Run(func() logging.Log {
		defer locker.Release(gctx)

		now := time.Now()
		err := a.release(gctx, creation)
		log := logging.Log{
			"action":   "release_creation",
			"rid":      sess.RID,
			"uid":      sess.UserID.String(),
			"gid":      creation.GID.String(),
			"cid":      creation.ID.String(),
			"language": *creation.Language,
			"version":  *creation.Version,
			"elapsed":  time.Since(now) / 1e6,
		}

		if err != nil {
			log["error"] = err.Error()
		}
		return log
	})

	return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.PublicationOutput]{
		Job:    job.String(),
		Result: nil,
	})
}

func (a *Creation) release(gctx context.Context, creation *bll.CreationOutput) error {
	if *creation.Status != 2 {
		sess := gear.CtxValue[middleware.Session](gctx)
		if sess.UserID != creation.GID {
			return errors.New("cannot release creation, status is not 2")
		}

		// 用户私有 group 自动提升 status，无需 review 和 approve
		statusInput := &bll.UpdateCreationStatusInput{
			GID: creation.GID,
			ID:  creation.ID,
		}

		if *creation.Status == 0 {
			statusInput.Status = 1
			statusInput.UpdatedAt = *creation.UpdatedAt
			output, err := a.blls.Writing.UpdateCreationStatus(gctx, statusInput)
			if err != nil {
				return err
			}
			creation.Status = output.Status
			creation.UpdatedAt = output.UpdatedAt
		}

		if *creation.Status == 1 {
			cr, err := a.summarize(gctx, creation.GID, creation.ID)
			if err != nil {
				return err
			}

			statusInput.Status = 2
			statusInput.UpdatedAt = *cr.UpdatedAt
			output, err := a.blls.Writing.UpdateCreationStatus(gctx, statusInput)
			if err != nil {
				return gear.ErrInternalServerError.From(err)
			}
			creation.Status = output.Status
			creation.UpdatedAt = output.UpdatedAt
		}
	}

	_, err := a.blls.Writing.CreatePublication(gctx, &bll.CreatePublication{
		GID:      creation.GID,
		CID:      creation.ID,
		Language: *creation.Language,
		Version:  *creation.Version,
	})
	return err
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
		Fields: "status,creator,updated_at,language,version",
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

// summarize and embedding when updating status from 1 to 2
func (a *Creation) summarize(gctx context.Context, gid, cid util.ID) (*bll.CreationOutput, error) {
	creation, err := a.blls.Writing.GetCreation(gctx, &bll.QueryCreation{
		GID:    gid,
		ID:     cid,
		Fields: "status,creator,updated_at,language,version,content",
	})

	if err != nil {
		return nil, err
	}

	if creation.Content == nil || creation.Status == nil {
		return nil, errors.New("invalid creation")
	}

	if *creation.Status != 1 {
		return nil, errors.New("cannot summarize creation content, status is not 1")
	}

	teContents, err := content.ToTEContents([]byte(*creation.Content))
	if err != nil {
		return nil, err
	}
	teData, err := cbor.Marshal(teContents)
	if err != nil {
		return nil, err
	}

	sess := gear.CtxValue[middleware.Session](gctx)
	go logging.Run(func() logging.Log {
		now := time.Now()
		_, err := a.blls.Jarvis.Embedding(gctx, &bll.EmbeddingInput{
			GID:      gid,
			CID:      cid,
			Language: *creation.Language,
			Version:  *creation.Version,
			Content:  teData,
		})
		log := logging.Log{
			"action":   "embedding",
			"rid":      sess.RID,
			"uid":      sess.UserID.String(),
			"gid":      gid.String(),
			"cid":      cid.String(),
			"language": *creation.Language,
			"version":  *creation.Version,
			"elapsed":  time.Since(now) / 1e6,
		}

		if err != nil {
			log["error"] = err.Error()
		}
		return log
	})

	summary, err := a.blls.Jarvis.Summarize(gctx, teContents)
	if err != nil {
		return nil, err
	}

	output, err := a.blls.Writing.UpdateCreation(gctx, &bll.UpdateCreationInput{
		GID:       gid,
		ID:        cid,
		UpdatedAt: *creation.UpdatedAt,
		Summary:   &summary,
	})
	if err != nil {
		return nil, err
	}

	return output, nil
}
