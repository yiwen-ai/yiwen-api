package api

import (
	"time"

	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Payment struct {
	blls *bll.Blls
}

type PaymentCode struct {
	Kind     int8     `cbor:"1,keyasint"`           // 0: subscribe creation; 2: subscribe collection
	ExpireAt int64    `cbor:"2,keyasint"`           // code 的失效时间，unix 秒
	Payee    util.ID  `cbor:"3,keyasint"`           // 收款人 id
	SubPayee *util.ID `cbor:"4,keyasint,omitempty"` // 分成收款人 id
	Amount   int64    `cbor:"5,keyasint"`           // 花费的亿文币数量
	GID      util.ID  `cbor:"6,keyasint"`           // 订阅对象所属 group
	UID      util.ID  `cbor:"7,keyasint"`           // 受益人 id
	CID      util.ID  `cbor:"8,keyasint"`           // 订阅对象 id
	Duration int64    `cbor:"9,keyasint"`           // 增加的订阅时长，单位秒
}

type SubscriptionToken struct {
	Kind     int8    `cbor:"1,keyasint"` // 2: collection subscription
	ExpireAt int64   `cbor:"2,keyasint"` // 订阅失效时间，unix 秒
	GID      util.ID `cbor:"3,keyasint"` // 订阅对象所属 group
	UID      util.ID `cbor:"4,keyasint"` // 受益人 id
	CID      util.ID `cbor:"5,keyasint"` // 订阅对象 id
}

type QueryPaymentCode struct {
	Kind int8    `json:"kind" cbor:"kind" query:"kind" validate:"gte=0,lte=2"`
	CID  util.ID `json:"cid" cbor:"cid" query:"cid" validate:"required"`
	// 触发支付的 group，如果不是订阅对象所属 group，则分享收益给该 group
	GID util.ID `json:"gid" cbor:"gid" query:"gid" validate:"required"`
}

func (i *QueryPaymentCode) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type PaymentCodeOutput struct {
	Kind      int8           `json:"kind" cbor:"kind"`
	Title     string         `json:"title" cbor:"title"`
	Duration  int64          `json:"duration" cbor:"duration"`
	Amount    int64          `json:"amount" cbor:"amount"`
	Code      string         `json:"code" cbor:"code"`
	ExpireAt  int64          `json:"expire_at" cbor:"expire_at"`
	GroupInfo *bll.GroupInfo `json:"group_info" cbor:"group_info"`
}

func (a *Payment) GetCode(ctx *gear.Context) error {
	input := &QueryPaymentCode{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	code := &PaymentCode{
		Kind:     input.Kind,
		ExpireAt: time.Now().Add(time.Hour).Unix(),
		GID:      input.GID,
		UID:      sess.UserID,
		CID:      input.CID,
		Duration: 3600 * 24 * 365 * 3, // seconds
	}
	output := &PaymentCodeOutput{
		Kind:     code.Kind,
		Duration: code.Duration,
		ExpireAt: code.ExpireAt,
	}

	language := ""
	version := uint16(1)
	PayeeGID := input.GID
	var SubPayeeGID *util.ID

	switch input.Kind {
	default:
		return gear.ErrBadRequest.WithMsg("invalid kind")
	case 0, 1:
		doc, err := a.blls.Writing.ImplicitGetPublication(ctx, &bll.ImplicitQueryPublication{
			CID:    input.CID,
			GID:    &input.GID,
			Fields: "title",
		}, nil)
		if err != nil {
			return gear.ErrInternalServerError.From(err)
		}
		if doc.Title == nil || doc.Price == nil || doc.FromGID == nil {
			return gear.ErrInternalServerError.WithMsg("title or price or from_gid is nil")
		}

		if *doc.FromGID != input.GID {
			PayeeGID = *doc.FromGID
			SubPayeeGID = &doc.GID
		}

		code.Amount = *doc.Price
		if code.Amount <= 0 {
			return gear.ErrBadRequest.WithMsg("creation is free")
		}
		output.Amount = code.Amount
		output.Title = *doc.Title
		language = doc.Language
		version = doc.Version
	case 2:
		doc, err := a.blls.Writing.GetCollection(ctx, &bll.QueryGidID{
			GID:    util.ZeroID,
			ID:     input.CID,
			Fields: "gid,info",
		})
		if err != nil {
			return gear.ErrInternalServerError.From(err)
		}
		if doc.Info == nil || doc.Price == nil {
			return gear.ErrInternalServerError.WithMsg("title or price is nil")
		}
		if doc.GID != input.GID {
			PayeeGID = doc.GID
			SubPayeeGID = &input.GID
		}

		code.Amount = *doc.Price
		if code.Amount <= 0 {
			return gear.ErrBadRequest.WithMsg("collection is free")
		}
		output.Amount = code.Amount
		output.Title = doc.Info.Title
		language = *doc.Language
		version = *doc.Version
		if len(doc.I18nInfo) > 0 {
			for k, v := range doc.I18nInfo {
				language = k
				output.Title = v.Title
				break
			}
		}
	}

	if SubPayeeGID != nil {
		group, err := a.blls.Userbase.GetGroup(ctx, *SubPayeeGID, "uid,status")
		if err == nil && *group.Status >= 0 && *group.UID != sess.UserID {
			code.SubPayee = group.UID
		}
	}

	group, err := a.blls.Userbase.GetGroup(ctx, PayeeGID, "uid,cn,name,logo,status,slogan")
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if *group.Status < 0 {
		return gear.ErrBadRequest.WithMsg("group is not active")
	}
	code.Payee = *group.UID

	output.GroupInfo = &bll.GroupInfo{
		ID:     *group.ID,
		CN:     group.CN,
		Name:   group.Name,
		Logo:   *group.Logo,
		Slogan: *group.Slogan,
		Status: *group.Status,
	}

	output.Code, err = util.EncodeEncrypt0(a.blls.Encryptor, code, []byte("PaymentCode"))
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	_, _ = a.blls.Writing.CreateBookmark(ctx, &bll.CreateBookmarkInput{
		GID:      code.GID,
		CID:      code.CID,
		Language: language,
		Version:  version,
		Kind:     output.Kind,
		Title:    output.Title,
	})

	return ctx.OkSend(bll.SuccessResponse[[]*PaymentCodeOutput]{Result: []*PaymentCodeOutput{output}})
}

type PaymentInput struct {
	Code string `json:"code" cbor:"code" query:"code" validate:"required"`
}

func (i *PaymentInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func (a *Payment) PayByCode(ctx *gear.Context) error {
	input := &PaymentInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	code, err := util.DecodeEncrypt0[PaymentCode](a.blls.Encryptor, input.Code, []byte("PaymentCode"))
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}
	now := time.Now().Unix()
	if code.ExpireAt < now {
		return gear.ErrBadRequest.WithMsg("code expired")
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	wallet, err := a.blls.Walletbase.Get(ctx, sess.UserID)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if wallet.Balance() < code.Amount {
		return gear.ErrPaymentRequired.WithMsg("insufficient balance")
	}

	var subscription *bll.SubscriptionOutput
	var logAction string
	switch code.Kind {
	default:
		return gear.ErrBadRequest.WithMsg("invalid kind")
	case 0, 1:
		logAction = bll.LogActionCreationSubscribe
		subscription, _ = a.blls.Writing.InternalGetCreationSubscription(ctx, code.CID)
	case 2:
		logAction = bll.LogActionCollectionSubscribe
		subscription, _ = a.blls.Writing.InternalGetCollectionSubscription(ctx, code.CID)
	}
	if subscription != nil && subscription.ExpireAt > (now+code.Duration/2) {
		return gear.ErrBadRequest.WithMsg("already subscribed")
	}

	subscriptionInput := &bll.SubscriptionInput{
		UID:      code.UID,
		CID:      code.CID,
		ExpireAt: now + code.Duration,
	}
	if subscription != nil {
		subscriptionInput.UpdatedAt = subscription.UpdatedAt
		if subscription.ExpireAt > now {
			subscriptionInput.ExpireAt = subscription.ExpireAt + code.Duration
		}
	}

	payload, err := util.Marshal(code)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	log, err := a.blls.Logbase.Log(ctx, logAction, 0, code.GID, code)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	auditLog := &bll.UpdateLog{
		UID: log.UID,
		ID:  log.ID,
	}

	wallet, err = a.blls.Walletbase.Subscribe(ctx, &bll.SpendInput{
		UID:         sess.UserID,
		Amount:      code.Amount,
		Payee:       &code.Payee,
		SubPayee:    code.SubPayee,
		Description: logAction,
		Payload:     payload,
	})
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	subscriptionInput.Txn = wallet.Txn
	txn := &bll.TransactionPK{
		UID: sess.UserID,
		ID:  wallet.Txn,
	}

	switch code.Kind {
	default:
		err = gear.ErrBadRequest.WithMsg("invalid kind")
	case 0:
		subscription, err = a.blls.Writing.InternalUpdateCreationSubscription(ctx, subscriptionInput)
	case 2:
		subscription, err = a.blls.Writing.InternalUpdateCollectionSubscription(ctx, subscriptionInput)
	}

	if err == nil {
		auditLog.Status = 1
		err = a.blls.Walletbase.CommitTxn(ctx, txn)
	}

	if err != nil {
		auditLog.Status = -1
		auditLog.Error = util.Ptr(err.Error())
		_ = a.blls.Walletbase.CancelTxn(ctx, txn)
		a.blls.Logbase.Update(ctx, auditLog)
		return gear.ErrInternalServerError.From(err)
	}

	a.blls.Logbase.Update(ctx, auditLog)
	return ctx.OkSend(bll.SuccessResponse[*bll.SubscriptionOutput]{Result: subscription})
}
