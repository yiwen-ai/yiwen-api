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
	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/service"
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

	doc, err := content.ParseDocumentNode(input.Content)
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}
	teContents := doc.ToTEContents()
	if len(teContents) == 0 {
		return gear.ErrBadRequest.WithMsg("invalid content")
	}
	input.Content, err = cbor.Marshal(doc)
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionCreationCreate, 1, input.GID, &bll.LogPayload{
		GID:      output.GID,
		CID:      output.ID,
		Language: output.Language,
		Version:  output.Version,
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) Get(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
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

	result := bll.CreationOutputs{*output}
	result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
		return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	})
	result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: &result[0]})
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionCreationUpdate, 1, input.GID, &bll.LogPayload{
		GID:      output.GID,
		CID:      output.ID,
		Language: output.Language,
		Version:  output.Version,
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) Delete(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionCreationDelete, 1, input.GID, &bll.LogPayload{
		GID:    input.GID,
		CID:    input.ID,
		Status: util.Ptr(int8(-2)),
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
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

	// output.Result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
	// 	return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	// })
	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

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

	input.Status = util.Ptr(int8(-1))
	output, err := a.blls.Writing.ListCreation(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	// output.Result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
	// 	return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	// })
	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Creation) Archive(ctx *gear.Context) error {
	input := &bll.UpdateStatusInput{}
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionCreationUpdate, 1, input.GID, &bll.LogPayload{
		GID:    input.GID,
		CID:    input.ID,
		Status: util.Ptr(int8(-1)),
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) Redraft(ctx *gear.Context) error {
	input := &bll.UpdateStatusInput{}
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionCreationUpdate, 1, input.GID, &bll.LogPayload{
		GID:    input.GID,
		CID:    input.ID,
		Status: util.Ptr(int8(0)),
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) checkTokens(ctx *gear.Context, gid, cid util.ID) error {
	src, err := a.blls.Writing.GetCreation(ctx, &bll.QueryGidID{
		GID:    gid,
		ID:     cid,
		Fields: "content",
	})

	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	trans, err := content.EstimateTranslatingString(src.Content)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if tokens := util.Tiktokens(trans); tokens > util.MAX_CREATION_TOKENS {
		return gear.ErrUnprocessableEntity.WithMsgf("too many tokens: %d, expected <= %d",
			tokens, util.MAX_CREATION_TOKENS)
	}
	return nil
}

func (a *Creation) Release(ctx *gear.Context) error {
	input := &bll.CreatePublicationInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	input.Model = bll.AIModels[0].ID
	creation, err := a.checkWritePermission(ctx, input.GID, input.CID)
	if err != nil {
		return err
	}
	if *creation.Status < 0 {
		return gear.ErrBadRequest.WithMsg("cannot release creation, status is -1")
	}

	gctx := middleware.WithGlobalCtx(ctx)
	key := fmt.Sprintf("RC:%s:%s", input.GID.String(), input.CID.String())
	locker, err := a.blls.Locker.Lock(gctx, key, 10*60*time.Second)
	if err != nil {
		return gear.ErrLocked.From(err)
	}

	if err = a.checkTokens(ctx, input.GID, input.CID); err != nil {
		locker.Release(gctx)
		return err
	}

	sess := gear.CtxValue[middleware.Session](gctx)

	log, err := a.blls.Logbase.Log(ctx, bll.LogActionCreationRelease, 0, input.GID, &bll.LogPayload{
		GID:      creation.GID,
		CID:      creation.ID,
		Language: creation.Language,
		Version:  creation.Version,
	})

	if err != nil {
		locker.Release(gctx)
		return gear.ErrInternalServerError.From(err)
	}

	auditLog := &bll.UpdateLog{
		UID: log.UID,
		ID:  log.ID,
	}

	go logging.Run(func() logging.Log {
		conf.Config.ObtainJob()
		defer conf.Config.ReleaseJob()
		defer locker.Release(gctx)

		now := time.Now()
		err := a.release(gctx, creation, auditLog)
		log := logging.Log{
			"action":   "publication.create",
			"rid":      sess.RID,
			"uid":      sess.UserID.String(),
			"gid":      creation.GID.String(),
			"cid":      creation.ID.String(),
			"language": *creation.Language,
			"version":  *creation.Version,
			"elapsed":  time.Since(now) / 1e6,
		}

		if err != nil {
			auditLog.Status = -1
			auditLog.Error = util.Ptr(err.Error())
			log["error"] = err.Error()
		} else {
			auditLog.Status = 1

			go a.blls.Taskbase.Create(gctx, &bll.CreateTaskInput{
				UID:       sess.UserID,
				GID:       creation.GID,
				Kind:      "publication.review",
				Threshold: 1,
				Approvers: []util.ID{util.JARVIS},
				Assignees: []util.ID{},
			}, &bll.LogPayload{
				GID:      creation.GID,
				CID:      creation.ID,
				Language: creation.Language,
				Version:  creation.Version,
			})
		}

		go a.blls.Logbase.Update(gctx, auditLog)
		return log
	})

	return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.PublicationOutput]{
		Job:    log.ID.String(),
		Result: nil,
	})
}

func (a *Creation) release(gctx context.Context, creation *bll.CreationOutput, auditLog *bll.UpdateLog) error {
	if *creation.Status != 2 {
		sess := gear.CtxValue[middleware.Session](gctx)
		if sess.UserID != creation.GID {
			return errors.New("cannot release creation, status is not 2")
		}

		// 用户私有 group 自动提升 status，无需 review 和 approve
		statusInput := &bll.UpdateStatusInput{
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
			cr, err := a.summarize(gctx, creation.GID, creation.ID, auditLog)
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

	doc, err := content.ParseDocumentNode(input.Content)
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}
	teContents := doc.ToTEContents()
	if len(teContents) == 0 {
		return gear.ErrBadRequest.WithMsg("invalid content")
	}
	input.Content, err = cbor.Marshal(doc)
	if err != nil {
		return gear.ErrBadRequest.From(err)
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionCreationUpdateContent, 1, input.GID, &bll.LogPayload{
		GID:      input.GID,
		CID:      input.ID,
		Language: output.Language,
		Version:  output.Version,
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: output})
}

func (a *Creation) UploadFile(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
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

	output := a.blls.Writing.SignPostPolicy(creation.GID, creation.ID, *creation.Language, uint(*creation.Version))
	return ctx.OkSend(bll.SuccessResponse[service.PostFilePolicy]{Result: output})
}

func (a *Creation) UpdatePrice(ctx *gear.Context) error {
	input := &bll.UpdateCreationPriceInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	doc, err := a.checkWritePermission(ctx, input.GID, input.ID)
	if err != nil {
		return err
	}

	_, err = a.blls.Writing.UpdateCreationPrice(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionCreationUpdate, 1, input.GID, &bll.LogPayload{
		GID:   input.GID,
		CID:   input.ID,
		Price: &input.Price,
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	doc.Price = &input.Price
	return ctx.OkSend(bll.SuccessResponse[*bll.CreationOutput]{Result: doc})
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

	creation, err := a.blls.Writing.GetCreation(ctx, &bll.QueryGidID{
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
func (a *Creation) summarize(gctx context.Context, gid, cid util.ID, auditLog *bll.UpdateLog) (*bll.CreationOutput, error) {
	creation, err := a.blls.Writing.GetCreation(gctx, &bll.QueryGidID{
		GID:    gid,
		ID:     cid,
		Fields: "status,creator,updated_at,language,version,keywords,summary,content",
	})

	if err != nil {
		return nil, err
	}

	// do not update summary if exists
	if creation.Summary != nil && len(*creation.Summary) > 0 {
		return creation, nil
	}

	if creation.Content == nil || creation.Status == nil {
		return nil, errors.New("invalid creation")
	}

	if *creation.Status != 1 {
		return nil, errors.New("cannot summarize creation content, status is not 1")
	}

	doc, err := content.ParseDocumentNode(*creation.Content)
	if err != nil {
		return nil, err
	}
	teData, err := cbor.Marshal(doc.ToTEContents())
	if err != nil {
		return nil, err
	}

	sess := gear.CtxValue[middleware.Session](gctx)
	teInput := &bll.TEInput{
		GID:      gid,
		CID:      cid,
		Language: *creation.Language,
		Version:  *creation.Version,
		Content:  util.Ptr(util.Bytes(teData)),
	}

	summary, err := a.blls.Jarvis.Summarize(gctx, teInput)
	if err != nil {
		return nil, err
	}

	// summary will not generated by ai if the content is too short
	// tokens will be 0
	if summary.Tokens > 0 {
		go logging.Run(func() logging.Log {
			now := time.Now()
			_, err := a.blls.Jarvis.Embedding(gctx, teInput)
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
	}

	if len(summary.Keywords) > 5 {
		summary.Keywords = summary.Keywords[:5]
	}

	auditLog.Tokens = &summary.Tokens
	input := &bll.UpdateCreationInput{
		GID:       gid,
		ID:        cid,
		UpdatedAt: *creation.UpdatedAt,
		Summary:   &summary.Summary,
		Keywords:  &summary.Keywords,
	}

	// do not update keywords if exists
	if creation.Keywords != nil && len(*creation.Keywords) > 0 {
		input.Keywords = nil
	}

	output, err := a.blls.Writing.UpdateCreation(gctx, input)
	if err != nil {
		return nil, err
	}

	return output, nil
}
