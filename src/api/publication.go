package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/logging"
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
	input.Model = bll.DefaultModel

	if input.ToGID == nil {
		return gear.ErrBadRequest.WithMsg("to_gid is required")
	}

	if input.ToLanguage == nil {
		return gear.ErrBadRequest.WithMsg("to_language is required")
	}

	if err := a.checkCreatePermission(ctx, *input.ToGID); err != nil {
		return gear.ErrForbidden.From(err)
	}

	_, err := a.tryReadOne(ctx, input.GID, input.CID, input.Language, input.Version)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}

	src, err := a.blls.Writing.GetPublication(ctx, &bll.QueryPublication{
		GID:      input.GID,
		CID:      input.CID,
		Language: input.Language,
		Version:  input.Version,
		Fields:   "",
	})

	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	teContents, err := src.ToTEContents()
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	teData, err := cbor.Marshal(teContents)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	gctx := middleware.WithGlobalCtx(ctx)
	key := fmt.Sprintf("CP:%s:%s:%s:%d", input.ToGID.String(), input.CID.String(), *input.ToLanguage, input.Version)
	locker, err := a.blls.Locker.Lock(gctx, key, 120*time.Second)
	if err != nil {
		return gear.ErrLocked.From(err)
	}

	model := input.Model
	sess := gear.CtxValue[middleware.Session](gctx)
	job := &bll.Job{
		GID:       *input.ToGID,
		CID:       src.CID,
		Language:  *input.ToLanguage,
		Version:   src.Version,
		ExpiresIn: time.Now().Unix() + 120,
	}

	go logging.Run(func() logging.Log {
		defer locker.Release(gctx)

		now := time.Now()
		teOutput, err := a.blls.Jarvis.Translate(gctx, &bll.TEInput{
			GID:      *input.ToGID,
			CID:      src.CID,
			Language: *input.ToLanguage,
			Version:  src.Version,
			Model:    util.Ptr(model),
			Content:  util.Ptr(util.Bytes(teData)),
		})

		var draft *bll.PublicationDraft
		if err == nil {
			draft, err = src.IntoPublicationDraft(job.GID, job.Language, model, teOutput.Content)
			if err == nil {
				_, err = a.blls.Writing.CreatePublication(gctx, &bll.CreatePublication{
					GID:      src.GID,
					CID:      src.CID,
					Language: src.Language,
					Version:  src.Version,
					Draft:    draft,
				})
			}
		}

		log := logging.Log{
			"action":      "release_creation",
			"rid":         sess.RID,
			"uid":         sess.UserID.String(),
			"gid":         src.GID.String(),
			"cid":         src.CID.String(),
			"language":    src.Language,
			"version":     src.Version,
			"to_gid":      job.GID.String(),
			"to_language": job.Language,
			"elapsed":     time.Since(now) / 1e6,
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

func (a *Publication) Get(ctx *gear.Context) error {
	input := &bll.QueryPublication{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if _, err := a.tryReadOne(ctx, input.GID, input.CID, input.Language, input.Version); err != nil {
		return err
	}

	output, err := a.blls.Writing.GetPublication(ctx, input)
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) GetJob(ctx *gear.Context) error {
	input := &bll.QueryPublicationJob{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.Job.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.GetPublication(ctx, &bll.QueryPublication{
		GID:      input.Job.GID,
		CID:      input.Job.CID,
		Language: input.Job.Language,
		Version:  input.Job.Version,
	})

	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.PublicationOutput]{
				Job:    input.JobID,
				Result: nil,
			})
		}

		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) Update(ctx *gear.Context) error {
	input := &bll.UpdatePublicationInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	publication, err := a.checkWritePermission(ctx, input.GID, input.CID, input.Language, input.Version)
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
	input := &bll.GIDPagination{}
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
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	input.Status = util.Ptr(int8(-1))
	output, err := a.blls.Writing.ListPublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Publication) ListPublished(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	input.Status = util.Ptr(int8(2))
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

func (a *Publication) checkWritePermission(ctx *gear.Context, gid, cid util.ID, language string, version uint16) (*bll.PublicationOutput, error) {
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

func (a *Publication) tryReadOne(ctx *gear.Context, gid, cid util.ID, language string, version uint16) (*bll.PublicationOutput, error) {
	var err error
	var role int8 = -2

	if sess := gear.CtxValue[middleware.Session](ctx); sess != nil {
		role, _ = a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
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

	if role < -1 && *publication.Status < 2 {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	return publication, nil
}
