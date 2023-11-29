package api

import (
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
	"github.com/yiwen-ai/yiwen-api/src/service"
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

	src, err := a.tryReadOne(ctx, &bll.ImplicitQueryPublication{
		GID:      &input.GID,
		CID:      input.CID,
		Language: input.Language,
		Version:  input.Version,
	}, true)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	wallet, err := a.blls.Walletbase.Get(ctx, sess.UserID)
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

	tokens := a.blls.Jarvis.EstimateTranslatingTokens(trans, input.Language, toLang)
	output := &EstimateOutput{
		Balance: wallet.Balance(),
		Tokens:  tokens,
		Models:  make(map[string]ModelCost, len(bll.AIModels)),
	}

	models := bll.AIModels
	if wallet.Level < 2 {
		models = []bll.AIModel{bll.DefaultModel}
	}
	for _, md := range models {
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

	sess := gear.CtxValue[middleware.Session](ctx)
	wallet, err := a.blls.Walletbase.Get(ctx, sess.UserID)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if wallet.Balance() < 1 {
		return gear.ErrPaymentRequired.WithMsg("insufficient balance")
	}

	if wallet.Level < 2 && input.Model != bll.DefaultModel.ID {
		return gear.ErrBadRequest.WithMsgf("model %q is not allowed for user level < 2", input.Model)
	}

	src, err := a.tryReadOne(ctx, &bll.ImplicitQueryPublication{
		GID:      &input.GID,
		CID:      input.CID,
		Language: input.Language,
		Version:  input.Version,
	}, true)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}

	teContents, err := src.ToTEContents()
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	teData, err := cbor.Marshal(teContents)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	trans, err := teContents.EstimateTranslatingString()
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if tokens := util.Tiktokens(trans); tokens > util.MAX_TOKENS {
		return gear.ErrUnprocessableEntity.WithMsgf("too many tokens: %d, expected <= %d",
			tokens, util.MAX_TOKENS)
	}

	tokens := a.blls.Jarvis.EstimateTranslatingTokens(trans, input.Language, *input.ToLanguage)
	estimate_cost := model.CostWEN(tokens)
	if b := wallet.Balance(); b < estimate_cost {
		return gear.ErrPaymentRequired.WithMsgf("insufficient balance, expected %d, got %d", estimate_cost, b)
	}

	if input.Context == nil {
		input.Context = util.Ptr(fmt.Sprintf("The text is part or all of the %q", *src.Title))
	}

	dst, _ := a.blls.Writing.GetPublication(ctx, &bll.ImplicitQueryPublication{
		GID:      input.ToGID,
		CID:      input.CID,
		Language: *input.ToLanguage,
		Version:  input.Version,
		Fields:   "status,creator,updated_at",
	}, nil)
	if dst != nil && dst.Status != nil && *dst.Status >= 0 {
		return gear.ErrConflict.WithMsgf("%s publication already exists", *input.ToLanguage)
	}

	if dst != nil {
		a.blls.Writing.DeletePublication(ctx, &bll.QueryPublication{
			GID:      dst.GID,
			CID:      dst.CID,
			Language: *input.ToLanguage,
			Version:  dst.Version,
		})
	}

	gctx := middleware.WithGlobalCtx(ctx)
	key := fmt.Sprintf("CP:%s:%s:%s:%d", input.ToGID.String(), input.CID.String(), *input.ToLanguage, input.Version)
	locker, err := a.blls.Locker.Lock(gctx, key, 20*60*time.Second)
	if err != nil {
		return gear.ErrLocked.From(err)
	}

	payload := &bll.LogPayload{
		GID:      *input.ToGID,
		CID:      src.CID,
		Language: input.ToLanguage,
		Version:  &src.Version,
		Kind:     util.Ptr(int8(1)),
	}

	log, err := a.blls.Logbase.Log(ctx, bll.LogActionPublicationCreate, 0, payload.GID, payload)
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
			GID:          *input.ToGID,
			CID:          src.CID,
			Language:     *input.ToLanguage,
			Version:      src.Version,
			FromLanguage: util.Ptr(input.Language),
			Context:      input.Context,
			Model:        util.Ptr(input.Model),
			Content:      util.Ptr(util.Bytes(teData)),
		})

		var draft *bll.PublicationDraft
		if err == nil {
			auditLog.Tokens = util.Ptr(teOutput.Tokens)

			exp := bll.SpendPayload{
				GID:      *input.ToGID,
				CID:      &src.CID,
				Action:   bll.LogActionPublicationCreate,
				Language: *input.ToLanguage,
				Version:  src.Version,
				Model:    model.ID,
				Price:    model.Price,
				Tokens:   teOutput.Tokens,
			}

			wallet, err = a.blls.Walletbase.Spend(gctx, sess.UserID, &exp)
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
						err = a.blls.Walletbase.CommitTxn(gctx, txn)
					}
				}

				if err != nil {
					_ = a.blls.Walletbase.CancelTxn(gctx, txn)
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
	input := &bll.ImplicitQueryPublication{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	now := time.Now().Unix()
	subscription_in := &util.ZeroID
	subtoken, err := util.DecodeMac0[SubscriptionToken](a.blls.MACer, input.SubToken, []byte("SubscriptionToken"))
	if err == nil && subtoken.ExpireAt >= now {
		// fast API calling with subtoken
		subscription_in = &subtoken.GID
		if input.Parent != nil && *input.Parent != subtoken.CID {
			return gear.ErrBadRequest.WithMsg("invalid parent")
		}
		input.Parent = &subtoken.CID
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	role := int8(-2)
	if sess.UserID.Compare(util.MinID) > 0 && input.GID != nil {
		role, _ = a.blls.Userbase.UserGroupRole(ctx, sess.UserID, *input.GID)
		if role >= -1 {
			subscription_in = nil
		}
	}

	var output *bll.PublicationOutput
	if input.GID != nil && input.Language != "" && input.Version > 0 {
		output, err = a.blls.Writing.GetPublication(ctx, input, subscription_in)
		if err == nil && role < -1 && *output.Status < 2 {
			err = gear.ErrForbidden.WithMsg("no permission")
		}
	} else {
		if role < -1 {
			input.GID = nil
		}
		output, err = a.blls.Writing.ImplicitGetPublication(ctx, input, subscription_in)
	}

	if err != nil {
		return gear.ErrBadRequest.From(err)
	}

	result := bll.PublicationOutputs{*output}
	result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: &result[0]})
}

func (a *Publication) GetByJob(ctx *gear.Context) error {
	input := &bll.QueryJob{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	log, err := a.blls.Logbase.Get(ctx, sess.UserID, input.ID, "")
	if err != nil {
		return gear.ErrBadRequest.WithMsgf("invalid job: %s", err.Error())
	}

	if log.Action != bll.LogActionCreationRelease && log.Action != bll.LogActionPublicationCreate {
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

	if log.Status == 0 {
		progress := int8(0)
		if log.Action == bll.LogActionCreationRelease {
			res, err := a.blls.Jarvis.GetSummary(ctx, teInput)
			if err != nil {
				er := gear.ErrInternalServerError.From(err)
				if er.Code != 404 {
					return er
				}
			} else if res != nil {
				progress = res.Progress
			}
		} else {
			res, err := a.blls.Jarvis.GetTranslation(ctx, teInput)
			if err != nil {
				er := gear.ErrInternalServerError.From(err)
				if er.Code != 404 {
					return er
				}
			} else if res != nil {
				progress = res.Progress
			}
		}

		if progress < 100 {
			return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.PublicationOutput]{
				Job:      input.ID.String(),
				Progress: util.Ptr(progress),
				Result:   nil,
			})
		}
	}

	output, err := a.blls.Writing.GetPublication(ctx, &bll.ImplicitQueryPublication{
		GID:      &teInput.GID,
		CID:      teInput.CID,
		Language: teInput.Language,
		Version:  teInput.Version,
	}, nil)

	if err != nil {
		if util.IsNotFoundErr(err) {
			return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.PublicationOutput]{
				Job:      input.ID.String(),
				Progress: util.Ptr(int8(99)),
				Result:   nil,
			})
		}

		return gear.ErrInternalServerError.From(err)
	}

	result := bll.PublicationOutputs{*output}
	result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: &result[0]})
}

func (a *Publication) ListJob(ctx *gear.Context) error {
	sess := gear.CtxValue[middleware.Session](ctx)
	logs, err := a.blls.Logbase.ListRecently(ctx, &bll.ListRecentlyLogsInput{
		UID:     sess.UserID,
		Actions: []string{bll.LogActionCreationRelease, bll.LogActionPublicationCreate},
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
			if log.Action == bll.LogActionCreationRelease {
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

		doc, err := a.blls.Writing.GetPublication(ctx, &bll.ImplicitQueryPublication{
			GID:      &teInput.GID,
			CID:      teInput.CID,
			Language: teInput.Language,
			Version:  teInput.Version,
			Fields:   "status,from_language,updated_at,title",
		}, nil)
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
		Kind:     util.Ptr(int8(1)),
		Status:   output.Status,
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
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
		Kind:     util.Ptr(int8(1)),
		Status:   util.Ptr(int8(-2)),
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
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

	// output.Result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
	// 	return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	// })
	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Publication) List(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if _, err := a.checkReadPermission(ctx, input.GID); err != nil {
		input.Status = util.Ptr(int8(2)) // only published for anonymous
	}

	output, err := a.blls.Writing.ListPublication(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

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

	var output *bll.SuccessResponse[bll.PublicationOutputs]
	if len(gids) == 0 {
		output, err = a.blls.Writing.ListLatestPublications(ctx, &bll.Pagination{
			PageToken: input.PageToken,
			PageSize:  input.PageSize,
			Fields:    input.Fields,
		})
	} else {
		output, err = a.blls.Writing.ListPublicationByGIDs(ctx, &bll.GIDsPagination{
			GIDs:      gids,
			PageToken: input.PageToken,
			PageSize:  input.PageSize,
			Fields:    input.Fields,
		})
	}
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

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

	// output.Result.LoadCreators(func(ids ...util.ID) []bll.UserInfo {
	// 	return a.blls.Userbase.LoadUserInfo(ctx, ids...)
	// })
	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Publication) GetPublishList(ctx *gear.Context) error {
	input := &bll.QueryGidCid{}
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
		Kind:     util.Ptr(int8(1)),
		Status:   util.Ptr(int8(-1)),
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
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
		Kind:     util.Ptr(int8(1)),
		Status:   util.Ptr(int8(0)),
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
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

	status := *publication.Status
	if status == 2 {
		return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: publication})
	}

	if status < 0 {
		return gear.ErrBadRequest.WithMsgf("cannot update publication, status is %d", status)
	}

	if status == 0 {
		input.Status = 1
		output, err := a.blls.Writing.UpdatePublicationStatus(ctx, input)
		if err != nil {
			return gear.ErrInternalServerError.From(err)
		}
		input.UpdatedAt = *output.UpdatedAt
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
		Kind:     util.Ptr(int8(1)),
		Status:   util.Ptr(int8(2)),
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
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
		Kind:     util.Ptr(int8(1)),
		Status:   output.Status,
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.PublicationOutput]{Result: output})
}

func (a *Publication) Bookmark(ctx *gear.Context) error {
	input := &bll.CreateBookmarkInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if _, err := a.tryReadOne(ctx, &bll.ImplicitQueryPublication{
		GID:      &input.GID,
		CID:      input.CID,
		Language: input.Language,
		Version:  input.Version,
	}, false); err != nil {
		return gear.ErrForbidden.From(err)
	}

	input.Kind = 1
	output, err := a.blls.Writing.CreateBookmark(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionUserBookmark, 1, sess.UserID, &bll.LogPayload{
		GID:      input.GID,
		CID:      input.CID,
		Language: &input.Language,
		Version:  &input.Version,
		Kind:     util.Ptr(int8(1)),
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.BookmarkOutput]{Result: output})
}

func (a *Publication) UploadFile(ctx *gear.Context) error {
	input := &bll.QueryPublication{}
	if ctx.Method == "POST" {
		if err := ctx.ParseBody(input); err != nil {
			return err
		}
	} else if err := ctx.ParseURL(input); err != nil {
		return err
	}
	publication, err := a.checkWritePermission(ctx, input.GID, input.CID, input.Language, input.Version)
	if err != nil {
		return err
	}

	if *publication.Status != 0 {
		return gear.ErrBadRequest.WithMsg("cannot update publication content, status is not 0 or 1")
	}

	output := a.blls.Writing.SignPostPolicy(publication.GID, publication.CID, publication.Language, uint(publication.Version))
	return ctx.OkSend(bll.SuccessResponse[service.PostFilePolicy]{Result: output})
}

func (a *Publication) checkReadPermission(ctx *gear.Context, gid util.ID) (int8, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil || sess.UserID.Compare(util.MinID) <= 0 {
		return -2, gear.ErrForbidden.WithMsg("no permission")
	}

	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return -2, gear.ErrNotFound.From(err)
	}
	if role < -1 {
		return role, gear.ErrForbidden.WithMsg("no permission")
	}

	return role, nil
}

func (a *Publication) checkCreatePermission(ctx *gear.Context, gid util.ID) error {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil || sess.UserID.Compare(util.MinID) <= 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return gear.ErrNotFound.From(err)
	}
	if role < 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	return nil
}

func (a *Publication) checkWritePermission(ctx *gear.Context, gid, cid util.ID, language string, version uint16) (*bll.PublicationOutput, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil || sess.UserID.Compare(util.MinID) <= 0 {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}
	if role < 0 {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	publication, err := a.blls.Writing.GetPublication(ctx, &bll.ImplicitQueryPublication{
		GID:      &gid,
		CID:      cid,
		Language: language,
		Version:  version,
		Fields:   "status,creator,updated_at",
	}, nil)

	if err != nil {
		return nil, gear.ErrNotFound.From(err)
	}
	if publication.Creator == nil || publication.Status == nil {
		return nil, gear.ErrInternalServerError.WithMsg("invalid publication")
	}

	if role < 1 && *publication.Creator != sess.UserID {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	return publication, nil
}

func (a *Publication) tryReadOne(ctx *gear.Context, input *bll.ImplicitQueryPublication, full bool) (*bll.PublicationOutput, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil || sess.UserID.Compare(util.MinID) <= 0 {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}
	subscription_in := input.GID
	if !full {
		subscription_in = nil
		input.Fields = "status,creator,updated_at"
	}

	role, _ := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, *input.GID)
	if role > -2 {
		subscription_in = nil
	} else if subscription_in != nil {
		subscription_in = &util.ZeroID
	}

	publication, err := a.blls.Writing.GetPublication(ctx, input, subscription_in)
	if err != nil {
		return nil, gear.ErrNotFound.From(err)
	}

	if publication.Status == nil {
		return nil, gear.ErrInternalServerError.WithMsg("invalid publication")
	}

	if role < -1 && *publication.Status < 2 {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	return publication, nil
}
