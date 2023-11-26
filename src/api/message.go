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
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Message struct {
	blls *bll.Blls
}

func (a *Message) Create(ctx *gear.Context) error {
	input := &bll.CreateMessageInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	_, err := bll.FromContent[*bll.KVMessage](input.Message)
	if err != nil {
		_, err = bll.FromContent[*bll.ArrayMessage](input.Message)
	}
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}

	if err := a.checkWritePermission(ctx, input.AttachTo); err != nil {
		return err
	}

	output, err := a.blls.Writing.CreateMessage(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionMessageCreate, 1, input.AttachTo, &bll.LogMessage{
		ID:       output.ID,
		AttachTo: input.AttachTo,
		Kind:     output.Kind,
		Language: output.Language,
		Version:  output.Version,
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.MessageOutput]{Result: output})
}

func (a *Message) Update(ctx *gear.Context) error {
	input := &bll.UpdateMessageInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if input.Message != nil {
		_, err := bll.FromContent[*bll.KVMessage](*input.Message)
		if err != nil {
			_, err = bll.FromContent[*bll.ArrayMessage](*input.Message)
		}
		if err != nil {
			return gear.ErrBadRequest.From(err)
		}
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateMessage(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if _, err = a.blls.Logbase.Log(ctx, bll.LogActionMessageUpdate, 1, input.GID, &bll.LogMessage{
		ID:       input.ID,
		AttachTo: input.GID,
		Language: input.Language,
		Version:  &input.Version,
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.MessageOutput]{Result: output})
}

func (a *Message) UpdateI18n(ctx *gear.Context) error {
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

	model := bll.DefaultModel
	if input.Model != nil {
		model = bll.GetAIModel(*input.Model)
	}

	lang := *input.Language
	msg, err := a.blls.Writing.GetMessage(ctx, &bll.QueryID{
		ID: input.ID, Fields: "version,language,attach_to,context,message," + lang,
	})
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if *msg.Version != input.Version {
		return gear.ErrBadRequest.WithMsg("version mismatch")
	}
	if *msg.Language == *input.Language {
		return gear.ErrBadRequest.WithMsg("language is the same")
	}
	if *msg.AttachTo != input.GID {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	var srcMsg bll.MessageContainer
	srcMsg, err = bll.FromContent[*bll.KVMessage](*msg.Message)
	if err != nil {
		srcMsg, err = bll.FromContent[*bll.ArrayMessage](*msg.Message)
	}
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	dstMsg := srcMsg.New()
	if data, ok := msg.I18nMessages[lang]; ok {
		if err = dstMsg.UnmarshalCBOR(data); err != nil {
			return gear.ErrInternalServerError.From(err)
		}
	}

	if input.NewlyAdd != nil && *input.NewlyAdd {
		srcMsg, err = srcMsg.NewlyAdd(dstMsg)
		if err != nil {
			return gear.ErrInternalServerError.From(err)
		}
	}

	if srcMsg.IsEmpty() {
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

	teContents := srcMsg.ToTEContents()
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
	estimate_cost := model.CostWEN(tokens)
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

	log, err := a.blls.Logbase.Log(ctx, bll.LogActionMessageUpdate, 0, payload.AttachTo, payload)
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
			Model:        util.Ptr(model.ID),
			Content:      util.Ptr(util.Bytes(teData)),
		})

		if err == nil {
			err = bll.WithContent(dstMsg, tmOutput.Content)
		}

		if err == nil {
			auditLog.Tokens = util.Ptr(tmOutput.Tokens)

			exp := bll.SpendPayload{
				GID:      *msg.AttachTo,
				ID:       &msg.ID,
				Action:   bll.LogActionMessageUpdate,
				Language: *input.Language,
				Version:  *msg.Version,
				Model:    model.ID,
				Price:    model.Price,
				Tokens:   tmOutput.Tokens,
			}

			wallet, err = a.blls.Walletbase.Spend(gctx, sess.UserID, &exp)
			if err == nil {
				txn := &bll.TransactionPK{
					UID: sess.UserID,
					ID:  wallet.Txn,
				}

				data, err := cbor.Marshal(dstMsg)
				if err == nil {
					input.Message = util.Ptr(util.Bytes(data))
					_, err = a.blls.Writing.UpdateMessage(gctx, input)
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
			log["cost"] = model.CostWEN(*auditLog.Tokens)
		}

		go a.blls.Logbase.Update(gctx, auditLog)
		return log
	})

	return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.MessageOutput]{
		Job:    auditLog.ID.String(),
		Result: nil,
	})
}

func (a *Message) Get(ctx *gear.Context) error {
	input := &bll.QueryID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.GetMessage(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.MessageOutput]{Result: output})
}

func (a *Message) GetByJob(ctx *gear.Context) error {
	input := &bll.QueryJob{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	log, err := a.blls.Logbase.Get(ctx, sess.UserID, input.ID, "")
	if err != nil {
		return gear.ErrBadRequest.WithMsgf("invalid job: %s", err.Error())
	}

	if log.Action != bll.LogActionMessageUpdate {
		return gear.ErrBadRequest.WithMsgf("invalid job action: %s", log.Action)
	}

	if log.Error != nil {
		return gear.ErrInternalServerError.WithMsgf("job %s error: %s", log.Action, *log.Error)
	}

	p, err := util.Unmarshal[bll.LogMessage](log.Payload)
	if err != nil {
		return gear.ErrBadRequest.WithMsgf("invalid job: %v", err)
	}

	if err := a.checkWritePermission(ctx, p.AttachTo); err != nil {
		return err
	}

	tmInput := &bll.TMInput{
		ID:       p.ID,
		Language: *p.Language,
		Version:  *p.Version,
	}

	if log.Status == 0 {
		progress := int8(0)

		res, err := a.blls.Jarvis.GetMessageTranslation(ctx, tmInput)
		if err != nil {
			er := gear.ErrInternalServerError.From(err)
			if er.Code != 404 {
				return er
			}
		} else if res != nil {
			progress = res.Progress
		}

		if progress < 100 {
			return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.MessageOutput]{
				Job:      input.ID.String(),
				Progress: util.Ptr(progress),
				Result:   nil,
			})
		}
	}

	output, err := a.blls.Writing.GetMessage(ctx, &bll.QueryID{
		ID: tmInput.ID,
	})

	if err != nil {
		if util.IsNotFoundErr(err) {
			return ctx.Send(http.StatusAccepted, bll.SuccessResponse[*bll.MessageOutput]{
				Job:      input.ID.String(),
				Progress: util.Ptr(int8(99)),
				Result:   nil,
			})
		}

		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.MessageOutput]{Result: output})
}

func (a *Message) checkWritePermission(ctx *gear.Context, gid util.ID) error {
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
