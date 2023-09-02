package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/xid"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Publication struct {
	blls *bll.Blls
}

type EstimateOutput struct {
	Balance int64                `json:"balance" cbor:"balance"`
	Tokens  uint32               `json:"tokens" cbor:"tokens"`
	Models  map[string]ModelCost `json:"models" cbor:"models"`
}

type ModelCost struct {
	ID    string  `json:"id" cbor:"id"`
	Name  string  `json:"name" cbor:"name"`
	Price float64 `json:"price" cbor:"price"`
	Cost  int64   `json:"cost" cbor:"cost"`
}

func (a *Publication) Estimate(ctx *gear.Context) error {
	input := &bll.CreatePublicationInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	_, err := a.tryReadOne(ctx, input.GID, input.CID, input.Language, input.Version)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	wallet, err := a.blls.Walletbase.Get(ctx, sess.UserID)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
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

	toLang := input.Language
	if input.ToLanguage != nil {
		toLang = *input.ToLanguage
	}

	trans, err := content.EstimateTranslatingString(src.Content)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	tokens := util.EstimateTranslatingTokens(trans, input.Language, toLang)
	output := &EstimateOutput{
		Balance: wallet.Balance(),
		Tokens:  tokens,
		Models:  make(map[string]ModelCost, len(bll.AIModels)),
	}

	for _, md := range bll.AIModels {
		output.Models[md.ID] = ModelCost{
			ID:    md.ID,
			Name:  md.Name,
			Price: md.Price,
			Cost:  md.CostWEN(tokens),
		}
	}

	return ctx.OkSend(bll.SuccessResponse[*EstimateOutput]{Result: output})
}

func (a *Publication) Create(ctx *gear.Context) error {
	input := &bll.CreatePublicationInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	model := bll.GetAIModel(input.Model)
	input.Model = model.ID

	if input.ToGID == nil {
		return gear.ErrBadRequest.WithMsg("to_gid is required")
	}

	if input.ToLanguage == nil {
		return gear.ErrBadRequest.WithMsg("to_language is required")
	}

	if *input.ToLanguage == input.Language {
		return gear.ErrBadRequest.WithMsg("to_language is same as language")
	}

	if err := a.checkCreatePermission(ctx, *input.ToGID); err != nil {
		return gear.ErrForbidden.From(err)
	}

	_, err := a.tryReadOne(ctx, input.GID, input.CID, input.Language, input.Version)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	wallet, err := a.blls.Walletbase.Get(ctx, sess.UserID)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if wallet.Balance() < 1 {
		return gear.ErrPaymentRequired.WithMsg("insufficient balance")
	}

	dst, _ := a.blls.Writing.GetPublication(ctx, &bll.QueryPublication{
		GID:      input.GID,
		CID:      input.CID,
		Language: *input.ToLanguage,
		Version:  input.Version,
		Fields:   "status,creator,updated_at",
	})
	if dst != nil {
		return gear.ErrConflict.WithMsgf("%s publication already exists", *input.ToLanguage)
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

	trans, err := content.EstimateTranslatingString(src.Content)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if tokens := util.Tiktokens(trans); tokens > util.MAX_TOKENS {
		return gear.ErrUnprocessableEntity.WithMsgf("too many tokens: %d, expected <= %d",
			tokens, util.MAX_TOKENS)
	}

	tokens := uint32(float32(util.EstimateTranslatingTokens(trans, input.Language, *input.ToLanguage)) * 0.9)
	estimate_cost := model.CostWEN(tokens)
	if b := wallet.Balance(); b < estimate_cost {
		return gear.ErrPaymentRequired.WithMsgf("insufficient balance, expected %d, got %d", estimate_cost, b)
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
	locker, err := a.blls.Locker.Lock(gctx, key, 600*time.Second)
	if err != nil {
		return gear.ErrLocked.From(err)
	}

	payload := &bll.LogPayload{
		GID:      *input.ToGID,
		CID:      src.CID,
		Language: input.ToLanguage,
		Version:  &src.Version,
	}

	log, err := a.blls.Logbase.Log(ctx, bll.LogActionPublicationCreate, 0, input.GID, payload)
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
		teOutput, err := a.blls.Jarvis.Translate(gctx, &bll.TEInput{
			GID:      *input.ToGID,
			CID:      src.CID,
			Language: *input.ToLanguage,
			Version:  src.Version,
			Model:    util.Ptr(input.Model),
			Content:  util.Ptr(util.Bytes(teData)),
		})

		var draft *bll.PublicationDraft
		if err == nil {
			auditLog.Tokens = util.Ptr(teOutput.Tokens)

			exp := bll.ExpendPayload{
				GID:      *input.ToGID,
				CID:      src.CID,
				Language: *input.ToLanguage,
				Version:  src.Version,
				Model:    model.ID,
				Price:    model.Price,
				Tokens:   teOutput.Tokens,
			}

			wallet, err = a.blls.Walletbase.Expend(gctx, sess.UserID, &exp)
			if err == nil {
				txn := &bll.TransactionPK{
					UID: sess.UserID,
					ID:  wallet.Txn,
				}

				draft, err = src.IntoPublicationDraft(payload.GID, *payload.Language, input.Model, teOutput.Content)
				if err == nil {
					_, err = a.blls.Writing.CreatePublication(gctx, &bll.CreatePublication{
						GID:      src.GID,
						CID:      src.CID,
						Language: src.Language,
						Version:  src.Version,
						Draft:    draft,
					})

					if err == nil {
						err = a.blls.Walletbase.CommitExpending(gctx, txn)
					}
				}

				if err != nil {
					_ = a.blls.Walletbase.CancelExpending(gctx, txn)
				}
			}
		}

		log := logging.Log{
			"action":      "publication.create",
			"rid":         sess.RID,
			"uid":         sess.UserID.String(),
			"gid":         src.GID.String(),
			"cid":         src.CID.String(),
			"language":    src.Language,
			"version":     src.Version,
			"to_gid":      payload.GID.String(),
			"to_language": *payload.Language,
			"elapsed":     time.Since(now) / 1e6,
			"tokens":      auditLog.Tokens,
		}

		if err != nil {
			auditLog.Status = -1
			auditLog.Error = util.Ptr(err.Error())
			log["error"] = err.Error()
		} else {
			auditLog.Status = 1
			log["cost"] = model.CostWEN(*auditLog.Tokens)

			go a.blls.Taskbase.Create(gctx, &bll.CreateTaskInput{
				UID:       sess.UserID,
				GID:       payload.GID,
				Kind:      "publication.review",
				Threshold: 2,
				Approvers: []util.ID{util.JARVIS},
				Assignees: []util.ID{},
			}, payload)
		}

		go a.blls.Logbase.Update(gctx, auditLog)
		return log
	})

	return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.PublicationOutput]{
		Job:    auditLog.ID.String(),
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

	result := bll.PublicationOutputs{*output}
	result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
		return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	})
	result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: &result[0]})
}

func (a *Publication) GetByJob(ctx *gear.Context) error {
	input := &bll.QueryPublicationJob{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	log, err := a.blls.Logbase.Get(ctx, sess.UserID, input.ID, "")
	if err != nil {
		return gear.ErrBadRequest.WithMsgf("invalid job: %s", err.Error())
	}

	if log.Action != "creation.release" && log.Action != "publication.create" {
		return gear.ErrBadRequest.WithMsgf("invalid job action: %s", log.Action)
	}

	if log.Error != nil {
		return gear.ErrInternalServerError.WithMsgf("job %s error: %s", log.Action, *log.Error)
	}

	p, err := util.Unmarshal[bll.LogPayload](log.Payload)
	if err != nil {
		return gear.ErrBadRequest.WithMsgf("invalid job: %v", err)
	}
	if p.Language == nil || p.Version == nil {
		return gear.ErrBadRequest.WithMsgf("invalid job payload: %v", p)
	}

	if _, err := a.checkReadPermission(ctx, p.GID); err != nil {
		return err
	}

	teInput := &bll.TEInput{
		GID:      p.GID,
		CID:      p.CID,
		Language: *p.Language,
		Version:  *p.Version,
	}

	progress := int8(0)
	if log.Action == "creation.release" {
		res, err := a.blls.Jarvis.GetSummary(ctx, teInput)
		if err != nil {
			return gear.ErrInternalServerError.From(err)
		}

		progress = res.Progress
	} else {
		res, err := a.blls.Jarvis.GetTranslation(ctx, teInput)
		if err != nil {
			return gear.ErrInternalServerError.From(err)
		}
		progress = res.Progress
	}

	if progress < 100 {
		return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.PublicationOutput]{
			Job:      input.ID.String(),
			Progress: util.Ptr(progress),
			Result:   nil,
		})
	}

	output, err := a.blls.Writing.GetPublication(ctx, &bll.QueryPublication{
		GID:      teInput.GID,
		CID:      teInput.CID,
		Language: teInput.Language,
		Version:  teInput.Version,
	})

	if err != nil {
		if errors.Is(err, util.ErrNotFound) {
			return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.PublicationOutput]{
				Job:      input.ID.String(),
				Progress: util.Ptr(int8(99)),
				Result:   nil,
			})
		}

		return gear.ErrInternalServerError.From(err)
	}

	result := bll.PublicationOutputs{*output}
	result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
		return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	})
	result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: &result[0]})
}

func (a *Publication) ListJob(ctx *gear.Context) error {
	sess := gear.CtxValue[middleware.Session](ctx)
	logs, err := a.blls.Logbase.ListRecently(ctx, &bll.ListRecentlyLogsInput{
		UID:     sess.UserID,
		Actions: []string{"creation.release", "publication.create"},
		Fields:  []string{"gid", "error", "tokens", "payload"},
	})

	if err != nil {
		return gear.ErrBadRequest.WithMsgf("list jobs failed: %s", err.Error())
	}
	output := make([]*bll.PublicationJob, 0, len(logs))

	for _, log := range logs {
		if len(output) > 20 || time.Since(xid.ID(log.ID).Time()) > 48*time.Hour {
			continue
		}

		p, err := util.Unmarshal[bll.LogPayload](log.Payload)
		if err != nil || p.Language == nil || p.Version == nil {
			continue
		}

		job := &bll.PublicationJob{
			Job:    log.ID.String(),
			Status: log.Status,
			Action: log.Action,
			Publication: bll.PublicationOutput{
				GID:      p.GID,
				CID:      p.CID,
				Language: *p.Language,
				Version:  *p.Version,
			},
			Error: log.Error,
		}
		if log.Tokens != nil {
			job.Tokens = *log.Tokens
		}

		teInput := &bll.TEInput{
			GID:      p.GID,
			CID:      p.CID,
			Language: *p.Language,
			Version:  *p.Version,
		}

		if log.Status == 1 {
			job.Progress = 100
		} else if log.Status == 0 {
			if log.Action == "creation.release" {
				res, err := a.blls.Jarvis.GetSummary(ctx, teInput)
				if err != nil {
					continue
				}

				job.Progress = res.Progress
			} else {
				res, err := a.blls.Jarvis.GetTranslation(ctx, teInput)
				if err != nil {
					continue
				}
				job.Progress = res.Progress
			}
		}

		doc, err := a.blls.Writing.GetPublication(ctx, &bll.QueryPublication{
			GID:      teInput.GID,
			CID:      teInput.CID,
			Language: teInput.Language,
			Version:  teInput.Version,
			Fields:   "status,from_language,updated_at,title",
		})
		if err == nil {
			job.Publication = *doc
		}

		output = append(output, job)
	}
	return ctx.OkSend(bll.SuccessResponse[[]*bll.PublicationJob]{Result: output})
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionPublicationUpdate, 1, input.GID, &bll.LogPayload{
		GID:      input.GID,
		CID:      input.CID,
		Language: &input.Language,
		Version:  &input.Version,
		Status:   output.Status,
	}); err != nil {
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionPublicationDelete, 1, input.GID, &bll.LogPayload{
		GID:      input.GID,
		CID:      input.CID,
		Language: &input.Language,
		Version:  &input.Version,
		Status:   util.Ptr(int8(-2)),
	}); err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Publication) List(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if _, err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.ListPublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output.Result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
		return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	})
	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Publication) Recommendations(ctx *gear.Context) error {
	sess := gear.CtxValue[middleware.Session](ctx)
	res := make([]*bll.PublicationOutput, 0, len(a.blls.Writing.Recommendations))
	for _, rr := range a.blls.Writing.Recommendations {
		if r := rr.PreferVersion(sess.Lang); r != nil {
			res = append(res, r)
		}
	}

	return ctx.OkSend(bll.SuccessResponse[[]*bll.PublicationOutput]{
		Result: res,
	})
}

func (a *Publication) ListByFollowing(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil {
		return gear.ErrUnauthorized.WithMsg("session is required")
	}

	gids, err := a.blls.Userbase.FollowingGids(ctx)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if len(gids) == 0 {
		return ctx.OkSend(bll.SuccessResponse[[]*bll.PublicationOutput]{
			Result: []*bll.PublicationOutput{},
		})
	}

	output, err := a.blls.Writing.ListPublicationByGIDs(ctx, &bll.GIDsPagination{
		GIDs:      gids,
		PageToken: input.PageToken,
		PageSize:  input.PageSize,
		Fields:    input.Fields,
	})
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output.Result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
		return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	})
	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Publication) ListArchived(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if _, err := a.checkReadPermission(ctx, input.GID); err != nil {
		return err
	}

	input.Status = util.Ptr(int8(-1))
	output, err := a.blls.Writing.ListPublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output.Result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
		return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	})
	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Publication) ListPublished(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	input.Status = util.Ptr(int8(2))
	output, err := a.blls.Writing.ListPublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output.Result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
		return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	})
	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Publication) GetPublishList(ctx *gear.Context) error {
	input := &bll.QueryAllPublish{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}
	role, _ := a.checkReadPermission(ctx, input.GID)
	status := int8(2)

	if role >= 0 {
		status = 0
	} else if role == -1 {
		status = 1
	} else {
		input.GID = util.ANON
	}

	output, err := a.blls.Writing.GetPublicationList(ctx, status, input)
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionPublicationUpdate, 1, input.GID, &bll.LogPayload{
		GID:      input.GID,
		CID:      input.CID,
		Language: &input.Language,
		Version:  &input.Version,
		Status:   util.Ptr(int8(-1)),
	}); err != nil {
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionPublicationUpdate, 1, input.GID, &bll.LogPayload{
		GID:      input.GID,
		CID:      input.CID,
		Language: &input.Language,
		Version:  &input.Version,
		Status:   util.Ptr(int8(0)),
	}); err != nil {
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

	gctx := middleware.WithGlobalCtx(ctx)
	go a.blls.Jarvis.EmbeddingPublic(gctx, &bll.TEInput{
		GID:      input.GID,
		CID:      input.CID,
		Language: input.Language,
		Version:  input.Version,
	})

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionPublicationPublish, 1, input.GID, &bll.LogPayload{
		GID:      input.GID,
		CID:      input.CID,
		Language: &input.Language,
		Version:  &input.Version,
		Status:   util.Ptr(int8(2)),
	}); err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) UpdateContent(ctx *gear.Context) error {
	input := &bll.UpdatePublicationContentInput{}
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

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionPublicationUpdateContent, 1, input.GID, &bll.LogPayload{
		GID:      input.GID,
		CID:      input.CID,
		Language: &input.Language,
		Version:  &input.Version,
		Status:   output.Status,
	}); err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) Collect(ctx *gear.Context) error {
	input := &bll.CreateCollectionInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if _, err := a.tryReadOne(ctx, input.GID, input.CID, input.Language, input.Version); err != nil {
		return err
	}

	output, err := a.blls.Writing.CreateCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionUserCollect, 1, sess.UserID, &bll.LogPayload{
		GID:      input.GID,
		CID:      input.CID,
		Language: &input.Language,
		Version:  &input.Version,
	}); err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: output})
}

func (a *Publication) checkReadPermission(ctx *gear.Context, gid util.ID) (int8, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return -2, gear.ErrInternalServerError.From(err)
	}
	if role < -1 {
		return role, gear.ErrForbidden.WithMsg("no permission")
	}

	return role, nil
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
