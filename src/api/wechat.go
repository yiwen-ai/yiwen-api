package api

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Wechat struct {
	blls *bll.Blls
}

type WechatTicketInput struct {
	Url string `json:"url" cbor:"url" validate:"required"`
}

func (i *WechatTicketInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	if !strings.HasPrefix(i.Url, "https://www.yiwen.pub") {
		return gear.ErrBadRequest.WithMsg("url must start with https://www.yiwen.pub")
	}

	return nil
}

type WechatTicketOutput struct {
	Url       string `json:"url" cbor:"url"`
	AppID     string `json:"appId" cbor:"appId"`
	NonceStr  string `json:"nonceStr" cbor:"nonceStr"`
	Timestamp uint   `json:"timestamp" cbor:"timestamp"`
	Signature string `json:"signature" cbor:"signature"`
}

func (a *Wechat) JsapiTicket(ctx *gear.Context) error {
	input := &WechatTicketInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	ticket, err := a.blls.Wechat.GetTicket(ctx)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output := &WechatTicketOutput{
		Url:       input.Url,
		AppID:     conf.Config.Wechat.AppID,
		NonceStr:  util.RandString(8),
		Timestamp: uint(time.Now().Unix()),
	}
	h := sha1.New()
	s := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s",
		ticket, output.NonceStr, output.Timestamp, output.Url)
	h.Write([]byte(s))
	output.Signature = hex.EncodeToString(h.Sum(nil))
	logging.SetTo(ctx, "input_url", output.Url)
	logging.SetTo(ctx, "output_s", s)
	logging.SetTo(ctx, "output_sig", output.Signature)

	return ctx.OkSend(bll.SuccessResponse[*WechatTicketOutput]{Result: output})
}
