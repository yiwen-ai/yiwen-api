package bll

import (
	"context"
	"fmt"
	"time"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Wechat struct {
	redis *service.Redis
}

func (b *Wechat) InitApp(ctx context.Context, _app *gear.App) error {
	if conf.Config.Wechat.Secret == "" {
		return nil
	}

	go logging.CtxRun(ctx, "Wechat.InitApp", b.fetchTicket)
	return nil
}

func (b *Wechat) GetAccessToken(ctx context.Context) (string, error) {
	output := &WechatToken{}
	err := b.redis.GetCBOR(ctx, b.redisKey("token"), output)
	if err == nil {
		n := util.Int63n(120 * 1000)
		if time.Now().Add(time.Duration(n) * time.Millisecond).
			Before(time.Unix(int64(output.ExpireAt), 0)) {
			go logging.CtxRun(ctx, "Wechat.GetAccessToken", b.fetchAccessToken)
		}

		return output.Token, nil
	}

	if err = b.fetchAccessToken(ctx); err != nil {
		return "", err
	}
	if err = b.redis.GetCBOR(ctx, b.redisKey("token"), output); err != nil {
		return "", err
	}
	return output.Token, nil
}

func (b *Wechat) GetTicket(ctx context.Context) (string, error) {
	output := &WechatTicket{}
	err := b.redis.GetCBOR(ctx, b.redisKey("ticket"), output)
	if err == nil {
		n := util.Int63n(120 * 1000)
		if time.Now().Add(time.Duration(n) * time.Millisecond).
			Before(time.Unix(int64(output.ExpireAt), 0)) {
			go logging.CtxRun(ctx, "Wechat.GetTicket", b.fetchTicket)
		}

		return output.Ticket, nil
	}

	if err = b.fetchTicket(ctx); err != nil {
		return "", err
	}
	if err = b.redis.GetCBOR(ctx, b.redisKey("ticket"), output); err != nil {
		return "", err
	}
	return output.Ticket, nil
}

func (b *Wechat) redisKey(key string) string {
	return fmt.Sprintf("wechat:%s:%s", conf.Config.Wechat.AppID, key)
}

// {"access_token":"ACCESS_TOKEN","expires_in":7200}
type WechatToken struct {
	Token    string `json:"access_token" cbor:"access_token"`
	Expires  uint   `json:"expires_in" cbor:"expires_in"`
	ExpireAt uint   `json:"expire_at" cbor:"expire_at"`
	ErrMsg   string `json:"errmsg" cbor:"errmsg"`
	ErrCode  int    `json:"errcode" cbor:"errcode"`
}

func (b *Wechat) fetchAccessToken(ctx context.Context) error {
	if conf.Config.Wechat.Secret == "" {
		return fmt.Errorf("Wechat secret is empty")
	}

	api := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		conf.Config.Wechat.AppID, conf.Config.Wechat.Secret)
	output := &WechatToken{}
	if err := util.RequestJSON(ctx, util.ExternalHTTPClient, "GET", api, nil, output); err != nil {
		return err
	}
	if output.ErrCode > 0 {
		return fmt.Errorf("Wechat fetchAccessToken error: %d, %s", output.ErrCode, output.ErrMsg)
	}
	if output.ErrCode < 0 {
		// retry
		time.Sleep(3 * time.Second)
		return b.fetchAccessToken(ctx)
	}

	output.ExpireAt = uint(time.Now().Add(time.Duration(output.Expires-3) * time.Second).Unix())
	return b.redis.SetCBOR(ctx, b.redisKey("token"), output, output.Expires-30)
}

//	{
//	  "errcode":0,
//	  "errmsg":"ok",
//	  "ticket":"bxLdikRXVbTPdHSM05e5u5sUoXNKdvsdshFKA",
//	  "expires_in":7200
//	}
type WechatTicket struct {
	Ticket   string `json:"ticket" cbor:"ticket"`
	Expires  uint   `json:"expires_in" cbor:"expires_in"`
	ExpireAt uint   `json:"expire_at" cbor:"expire_at"`
	ErrMsg   string `json:"errmsg" cbor:"errmsg"`
	ErrCode  int    `json:"errcode" cbor:"errcode"`
}

func (b *Wechat) fetchTicket(ctx context.Context) error {
	token, err := b.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	api := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=wx_card", token)
	output := &WechatTicket{}
	if err := util.RequestJSON(ctx, util.ExternalHTTPClient, "GET", api, nil, output); err != nil {
		return err
	}
	if output.ErrCode > 0 {
		return fmt.Errorf("Wechat fetchTicket error: %d, %s", output.ErrCode, output.ErrMsg)
	}
	if output.ErrCode < 0 {
		// retry
		time.Sleep(3 * time.Second)
		return b.fetchTicket(ctx)
	}

	output.ExpireAt = uint(time.Now().Add(time.Duration(output.Expires-3) * time.Second).Unix())
	return b.redis.SetCBOR(ctx, b.redisKey("ticket"), output, output.Expires-30)
}
