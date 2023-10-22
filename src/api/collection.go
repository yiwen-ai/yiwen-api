package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Collection struct {
	blls *bll.Blls
}

func (a *Collection) Get(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	role, _ := a.checkReadPermission(ctx, input.GID)
	status := int8(2)
	switch role {
	case 2, 1:
		status = -1
	case 0:
		status = 0
	case -1:
		status = 1
	}

	output, err := a.blls.Writing.GetCollection(ctx, input, status)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if output.Subscription != nil {
		subtoken, err := util.EncodeMac0(a.blls.MACer, SubscriptionToken{
			Kind:     2,
			ExpireAt: output.Subscription.ExpireAt,
			UID:      output.Subscription.UID,
			CID:      output.Subscription.CID,
			GID:      output.Subscription.GID,
		}, []byte("SubscriptionToken"))
		if err == nil {
			output.SubToken = &subtoken
		}
	}
	result := bll.CollectionOutputs{*output}
	result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: &result[0]})
}

func (a *Collection) ListByChild(ctx *gear.Context) error {
	input := &bll.QueryGidCid{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}
	role, _ := a.checkReadPermission(ctx, input.GID)
	input.Status = int8(2)
	switch role {
	case 2, 1, 0:
		input.Status = 0
	case -1:
		input.Status = 1
	}
	input.Fields = "gid,status,info"

	output, err := a.blls.Writing.ListCollectionByChild(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	for i := range output.Result {
		if s := output.Result[i].Subscription; s != nil {
			subtoken, err := util.EncodeMac0(a.blls.MACer, SubscriptionToken{
				Kind:     2,
				ExpireAt: s.ExpireAt,
				UID:      s.UID,
				CID:      s.CID,
				GID:      s.GID,
			}, []byte("SubscriptionToken"))
			if err == nil {
				output.Result[i].SubToken = &subtoken
			}
		}
	}

	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Collection) ListChildren(ctx *gear.Context) error {
	input := &bll.IDGIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	role, _ := a.checkReadPermission(ctx, input.GID)
	input.Status = util.Ptr(int8(2))
	switch role {
	case 2, 1, 0:
		input.Status = util.Ptr(int8(0))
	case -1:
		input.Status = util.Ptr(int8(1))
	}

	output, err := a.blls.Writing.ListCollectionChildren(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Collection) List(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	role, _ := a.checkReadPermission(ctx, input.GID)
	input.Status = util.Ptr(int8(2))
	switch role {
	case 2, 1, 0:
		input.Status = util.Ptr(int8(0))
	case -1:
		input.Status = util.Ptr(int8(1))
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

func (a *Collection) ListArchived(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	input.Status = util.Ptr(int8(-1))
	output, err := a.blls.Writing.ListCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Collection) Create(ctx *gear.Context) error {
	input := &bll.CreateCollectionInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.CreateCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: output})
}

func (a *Collection) Update(ctx *gear.Context) error {
	input := &bll.UpdateCollectionInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: output})
}

func (a *Collection) Delete(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.DeleteCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Collection) GetInfo(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.GetCollectionInfo(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[*bll.MessageOutput]{Result: output})
}

func (a *Collection) UpdateInfo(ctx *gear.Context) error {
	input := &bll.UpdateMessageInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if input.Language == nil {
		return gear.ErrBadRequest.WithMsg("invalid language")
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	lang := *input.Language
	msg, err := a.blls.Writing.GetCollectionInfo(ctx, &bll.QueryGidID{
		ID: input.ID, GID: input.GID,
		Fields: "version,language,attach_to,context,message," + lang,
	})
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if *msg.Version != input.Version {
		return gear.ErrBadRequest.WithMsg("version mismatch")
	}
	if *msg.Language == *input.Language {
		return gear.ErrBadRequest.WithMsg("language is the same")
	}

	var kv bll.KVMessage
	if err := cbor.Unmarshal(*msg.Message, &kv); err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	langKV := make(bll.KVMessage)
	if data, ok := msg.I18nMessages[lang]; ok {
		if err := cbor.Unmarshal(data, &langKV); err != nil {
			return gear.ErrInternalServerError.From(err)
		}
		for k := range langKV {
			delete(kv, k) // don't need to translate
		}
	}

	if len(kv) == 0 {
		return gear.ErrBadRequest.WithMsg("no need to translate")
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	wallet, err := a.blls.Walletbase.Get(ctx, sess.UserID)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if wallet.Balance() < 1 {
		return gear.ErrPaymentRequired.WithMsg("insufficient balance")
	}

	teContents := kv.ToTEContents()
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

	tokens := a.blls.Jarvis.EstimateTranslatingTokens(trans, *msg.Language, *input.Language)
	estimate_cost := bll.DefaultModel.CostWEN(tokens)
	if b := wallet.Balance(); b < estimate_cost {
		return gear.ErrPaymentRequired.WithMsgf("insufficient balance, expected %d, got %d", estimate_cost, b)
	}

	gctx := middleware.WithGlobalCtx(ctx)
	key := fmt.Sprintf("UM:%s:%s:%d", msg.ID.String(), *input.Language, *msg.Version)
	locker, err := a.blls.Locker.Lock(gctx, key, 10*60*time.Second)
	if err != nil {
		return gear.ErrLocked.From(err)
	}

	payload := &bll.LogMessage{
		ID:       msg.ID,
		AttachTo: *msg.AttachTo,
		Language: input.Language,
		Version:  msg.Version,
	}

	log, err := a.blls.Logbase.Log(ctx, bll.LogActionMessageUpdate, 0, input.GID, payload)
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
		tmOutput, err := a.blls.Jarvis.TranslateMessage(gctx, &bll.TMInput{
			ID:           msg.ID,
			Language:     *input.Language,
			Version:      *msg.Version,
			FromLanguage: msg.Language,
			Context:      msg.Context,
			Model:        util.Ptr(bll.DefaultModel.ID),
			Content:      util.Ptr(util.Bytes(teData)),
		})

		if err == nil {
			err = langKV.WithContent(tmOutput.Content)
		}

		if err == nil {
			auditLog.Tokens = util.Ptr(tmOutput.Tokens)

			exp := bll.SpendPayload{
				GID:      input.GID,
				ID:       &msg.ID,
				Action:   bll.LogActionMessageUpdate,
				Language: *input.Language,
				Version:  *msg.Version,
				Model:    bll.DefaultModel.ID,
				Price:    bll.DefaultModel.Price,
				Tokens:   tmOutput.Tokens,
			}

			wallet, err = a.blls.Walletbase.Spend(gctx, sess.UserID, &exp)
			if err == nil {
				txn := &bll.TransactionPK{
					UID: sess.UserID,
					ID:  wallet.Txn,
				}

				data, err := cbor.Marshal(langKV)
				if err == nil {
					input.Message = util.Ptr(util.Bytes(data))
					_, err = a.blls.Writing.UpdateCollectionInfo(gctx, input)
				}

				if err == nil {
					err = a.blls.Walletbase.CommitTxn(gctx, txn)
				}

				if err != nil {
					_ = a.blls.Walletbase.CancelTxn(gctx, txn)
				}
			}
		}

		log := logging.Log{
			"action":      "message.translate",
			"rid":         sess.RID,
			"uid":         sess.UserID.String(),
			"attach_to":   msg.AttachTo.String(),
			"id":          msg.ID.String(),
			"language":    msg.Language,
			"version":     msg.Version,
			"to_language": *input.Language,
			"elapsed":     time.Since(now) / 1e6,
			"tokens":      auditLog.Tokens,
		}

		if err != nil {
			auditLog.Status = -1
			auditLog.Error = util.Ptr(err.Error())
			log["error"] = err.Error()
		} else {
			auditLog.Status = 1
			log["cost"] = bll.DefaultModel.CostWEN(*auditLog.Tokens)
		}

		go a.blls.Logbase.Update(gctx, auditLog)
		return log
	})

	return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.MessageOutput]{
		Job:    auditLog.ID.String(),
		Result: nil,
	})
}

func (a *Collection) UpdateStatus(ctx *gear.Context) error {
	input := &bll.UpdateStatusInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateCollectionStatus(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: output})
}

func (a *Collection) AddChildren(ctx *gear.Context) error {
	input := &bll.AddCollectionChildrenInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.AddCollectionChildren(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[[]util.ID]{Result: output})
}

func (a *Collection) UpdateChild(ctx *gear.Context) error {
	input := &bll.UpdateCollectionChildInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateCollectionChild(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Collection) RemoveChild(ctx *gear.Context) error {
	input := &bll.QueryGidIdCid{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.RemoveCollectionChild(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Collection) UploadFile(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	err := a.checkWritePermission(ctx, input.GID)
	if err != nil {
		return err
	}

	input.Fields = "gid,status"
	doc, err := a.blls.Writing.GetCollection(ctx, input, -1)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if *doc.Status < 0 {
		return gear.ErrBadRequest.WithMsg("collection archived")
	}

	output := a.blls.Writing.SignPostPolicy(doc.GID, doc.ID, "", 0)
	return ctx.OkSend(bll.SuccessResponse[service.PostFilePolicy]{Result: output})
}

func (a *Collection) checkReadPermission(ctx *gear.Context, gid util.ID) (int8, error) {
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

func (a *Collection) checkWritePermission(ctx *gear.Context, gid util.ID) error {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil || sess.UserID.Compare(util.MinID) <= 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return gear.ErrNotFound.From(err)
	}
	if role <= 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	return nil
}
