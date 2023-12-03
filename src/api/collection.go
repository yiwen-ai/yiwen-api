package api

import (
	"fmt"
	"net/http"
	"strings"
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

type Collection struct {
	blls *bll.Blls
}

func (a *Collection) Get(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	role, _ := a.checkReadPermission(ctx, input.GID)
	switch role {
	case -2:
		input.GID = util.ZeroID
	}

	output, err := a.blls.Writing.GetCollection(ctx, input)
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
	case -2:
		input.GID = util.ZeroID
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
	if ctx.Method == "GET" {
		in := &bll.QueryIDGIDPagination{}
		if err := ctx.ParseURL(in); err != nil {
			return err
		}

		input = in.To()
	} else if err := ctx.ParseBody(input); err != nil {
		return err
	}

	role, _ := a.checkReadPermission(ctx, input.GID)
	input.Status = util.Ptr(int8(2))
	switch role {
	case 2, 1, 0:
		input.Status = util.Ptr(int8(0))
	case -1:
		input.Status = util.Ptr(int8(1))
	case -2:
		input.GID = util.ZeroID
	}

	output, err := a.blls.Writing.ListCollectionChildren(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Collection) List(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if ctx.Method == "GET" {
		in := &bll.QueryGIDPagination{}
		if err := ctx.ParseURL(in); err != nil {
			return err
		}

		input = in.To()
	} else if err := ctx.ParseBody(input); err != nil {
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

func (a *Collection) ListLatest(ctx *gear.Context) error {
	input := &bll.Pagination{}
	if ctx.Method == "GET" {
		in := &bll.QueryPagination{}
		if err := ctx.ParseURL(in); err != nil {
			return err
		}

		input = in.To()
	} else if err := ctx.ParseBody(input); err != nil {
		return err
	}

	output, err := a.blls.Writing.ListLatestCollections(ctx, input)
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
	if ctx.Method == "GET" {
		in := &bll.QueryGIDPagination{}
		if err := ctx.ParseURL(in); err != nil {
			return err
		}

		input = in.To()
	} else if err := ctx.ParseBody(input); err != nil {
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

	teContent := content.TEContents{
		&content.TEContent{
			ID:    "title",
			Texts: []string{input.Info.Title},
		},
	}
	if input.Info.Summary != nil {
		teContent = append(teContent, &content.TEContent{
			ID:    "summary",
			Texts: []string{*input.Info.Summary},
		})
	}
	teData, err := cbor.Marshal(teContent)
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
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
	err := a.checkWritePermission(ctx, input.GID)
	if err != nil {
		return err
	}

	output := &bll.CollectionOutput{
		ID:  input.ID,
		GID: input.GID,
	}
	if input.Cover != nil || input.Price != nil || input.CreationPrice != nil {
		output, err = a.blls.Writing.UpdateCollection(ctx, input)
		if err != nil {
			return gear.ErrInternalServerError.From(err)
		}
	}
	if input.Version != nil && (input.Context != nil || input.Info != nil || input.Languages != nil) {
		infoInput := &bll.UpdateMessageInput{
			ID:        input.ID,
			GID:       input.GID,
			Version:   *input.Version,
			Context:   input.Context,
			Language:  input.Language,
			Languages: input.Languages,
		}
		if input.Info != nil {
			msg := bll.ArrayMessage{
				&content.TEContent{
					ID:    "title",
					Texts: []string{input.Info.Title},
				},
			}
			if input.Info.Summary != nil {
				msg = append(msg, &content.TEContent{
					ID:    "summary",
					Texts: []string{*input.Info.Summary},
				})
			}
			if input.Info.Keywords != nil {
				msg = append(msg, &content.TEContent{
					ID:    "keywords",
					Texts: *input.Info.Keywords,
				})
			}
			if input.Info.Authors != nil {
				msg = append(msg, &content.TEContent{
					ID:    "authors",
					Texts: *input.Info.Authors,
				})
			}
			data, err := msg.MarshalCBOR()
			if err != nil {
				return gear.ErrInternalServerError.From(err)
			}
			infoInput.Message = util.Ptr(util.Bytes(data))
		}

		infoOutput, err := a.blls.Writing.UpdateCollectionInfo(ctx, infoInput)
		if err != nil {
			return gear.ErrInternalServerError.From(err)
		}
		output.Version = infoOutput.Version
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

	input.Fields = "status,updated_at,cover,price,creation_price"
	collection, err := a.blls.Writing.GetCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	input.Fields = "context,language,languages,version,message"
	message, err := a.blls.Writing.GetCollectionInfo(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	msg, err := bll.FromContent[*bll.ArrayMessage](*message.Message)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	info := &bll.CollectionInfo{}
	for _, v := range *msg {
		switch v.ID {
		case "title":
			info.Title = v.Texts[0]
		case "summary":
			info.Summary = &v.Texts[0]
		case "keywords":
			info.Keywords = &v.Texts
		case "authors":
			info.Authors = &v.Texts
		}
	}

	return ctx.OkSend(bll.SuccessResponse[bll.CollectionInfoOutput]{Result: bll.CollectionInfoOutput{
		ID:            collection.ID,
		GID:           collection.GID,
		MID:           message.ID,
		Status:        *collection.Status,
		UpdatedAt:     *collection.UpdatedAt,
		Cover:         *collection.Cover,
		Price:         *collection.Price,
		CreationPrice: *collection.CreationPrice,
		Language:      *message.Language,
		Languages:     message.Languages,
		Version:       *message.Version,
		Context:       *message.Context,
		Info:          *info,
	}})
}

func (a *Collection) TranslateInfo(ctx *gear.Context) error {
	input := &bll.TranslateCollectionInfoInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	model := bll.DefaultModel
	if input.Model != nil {
		model = bll.GetAIModel(*input.Model)
	}

	languages := strings.Join(input.Languages, ",")
	msg, err := a.blls.Writing.GetCollectionInfo(ctx, &bll.QueryGidID{
		ID: input.ID, GID: input.GID,
		Fields: "version,language,attach_to,context,message," + languages,
	})
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if *msg.Version != input.Version {
		return gear.ErrBadRequest.WithMsg("version mismatch")
	}

	var srcMsg bll.MessageContainer
	srcMsg, err = bll.FromContent[*bll.ArrayMessage](*msg.Message)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
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

	tokens := a.blls.Jarvis.EstimateTranslatingTokens(trans, *msg.Language, input.Languages[0])
	estimate_cost := model.CostWEN(tokens) * int64(len(input.Languages))
	if b := wallet.Balance(); b < estimate_cost {
		return gear.ErrPaymentRequired.WithMsgf("insufficient balance, expected %d, got %d", estimate_cost, b)
	}

	gctx := middleware.WithGlobalCtx(ctx)
	key := fmt.Sprintf("UM:%s:%s:%d", msg.ID.String(), *msg.Language, input.Version)
	locker, err := a.blls.Locker.Lock(gctx, key, 60*60*time.Second)
	if err != nil {
		return gear.ErrLocked.From(err)
	}

	payload := &bll.LogMessage{
		ID:        msg.ID,
		AttachTo:  *msg.AttachTo,
		Languages: input.Languages,
		Version:   msg.Version,
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

		var err error
		var usedTokens uint32
		now := time.Now()

		for _, language := range input.Languages {
			if language == *msg.Language {
				continue
			}

			dstMsg := srcMsg.New()
			if data, ok := msg.I18nMessages[language]; ok {
				err = dstMsg.UnmarshalCBOR(data)
			}

			var tmOutput *bll.TMOutput
			if err == nil {
				tmOutput, err = a.blls.Jarvis.TranslateMessage(gctx, &bll.TMInput{
					ID:           msg.ID,
					Language:     language,
					Version:      input.Version,
					FromLanguage: msg.Language,
					Context:      msg.Context,
					Model:        util.Ptr(model.ID),
					Content:      util.Ptr(util.Bytes(teData)),
				})
			}

			if err == nil {
				err = bll.WithContent(dstMsg, tmOutput.Content)
			}

			if err == nil {
				var data []byte
				data, err = cbor.Marshal(dstMsg)
				if err == nil {
					_, err = a.blls.Writing.UpdateCollectionInfo(gctx, &bll.UpdateMessageInput{
						ID:       input.ID,
						GID:      input.GID,
						Version:  input.Version,
						Language: &language,
						Message:  util.Ptr(util.Bytes(data)),
					})
				}
			}
			if err == nil {
				usedTokens += tmOutput.Tokens
			}
		}

		if usedTokens > 0 {
			auditLog.Tokens = util.Ptr(usedTokens)
			exp := bll.SpendPayload{
				GID:      input.GID,
				ID:       &msg.ID,
				Action:   bll.LogActionMessageUpdate,
				Language: languages,
				Version:  input.Version,
				Model:    model.ID,
				Price:    model.Price,
				Tokens:   usedTokens,
			}

			wallet, err = a.blls.Walletbase.Spend(gctx, sess.UserID, &exp)
			if err == nil {
				txn := &bll.TransactionPK{
					UID: sess.UserID,
					ID:  wallet.Txn,
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
			"version":     input.Version,
			"to_language": languages,
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

func (a *Collection) Bookmark(ctx *gear.Context) error {
	input := &bll.CreateBookmarkInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if _, err := a.tryReadOne(ctx, &bll.QueryGidID{
		GID: input.GID,
		ID:  input.CID,
	}); err != nil {
		return gear.ErrForbidden.From(err)
	}

	input.Kind = 2
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
		Kind:     util.Ptr(int8(2)),
	}); err != nil {
		logging.SetTo(ctx, "writeLogError", err.Error())
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.BookmarkOutput]{Result: output})
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
	doc, err := a.blls.Writing.GetCollection(ctx, input)
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

func (a *Collection) tryReadOne(ctx *gear.Context, input *bll.QueryGidID) (*bll.CollectionOutput, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil || sess.UserID.Compare(util.MinID) <= 0 {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	role, _ := a.checkReadPermission(ctx, input.GID)
	switch role {
	case -2:
		input.GID = util.ZeroID
	}
	input.Fields = "gid,status,updated_at"
	output, err := a.blls.Writing.GetCollection(ctx, input)
	if err != nil {
		return nil, gear.ErrInternalServerError.From(err)
	}

	if output.Status == nil {
		return nil, gear.ErrInternalServerError.WithMsg("invalid collection")
	}

	if role < -1 && *output.Status < 2 {
		return nil, gear.ErrForbidden.WithMsg("no permission")
	}

	return output, nil
}
