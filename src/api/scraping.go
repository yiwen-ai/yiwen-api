package api

import (
	"io"
	"mime"
	"net/http"

	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Scraping struct {
	blls *bll.Blls
}

func (a *Scraping) Create(ctx *gear.Context) error {
	input := bll.ScrapingInput{}
	if err := ctx.ParseURL(&input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, input.GID)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}
	if role < 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	output, err := a.blls.Webscraper.Create(ctx, input.Url)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bll.ScrapingOutput]{Result: *output})
}

// 目前仅支持 .html, .pdf, .md, .txt 文件，
// 即：`Content-Type: text/html`, `Content-Type: application/pdf`, `Content-Type: text/markdown`, `Content-Type: text/plain`
// 上传文件时必须携带 Content-Type，请求体为文件本身，不能超过 512kb
// 服务端会自动处理字符编码。
func (a *Scraping) Convert(ctx *gear.Context) error {
	var mtype string
	var err error
	if mtype = ctx.GetHeader(gear.HeaderContentType); mtype == "" {
		mtype = gear.MIMEOctetStream
	}
	mtype, _, err = mime.ParseMediaType(mtype)
	if err != nil {
		return gear.ErrUnsupportedMediaType.From(err)
	}

	reader := http.MaxBytesReader(ctx.Res, ctx.Req.Body, 2<<18) // 512kb
	buf, err := io.ReadAll(reader)
	if err != nil {
		reader.Close()
		return gear.ErrRequestEntityTooLarge.From(err)
	}
	reader.Close()
	buf, mtype, err = util.NormalizeFileEncodingAndType(buf, mtype)
	if err != nil {
		return err
	}

	util.HeaderFromCtx(ctx).Set(gear.HeaderContentType, mtype)
	output, err := a.blls.Webscraper.Convert(ctx, buf, mtype)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bll.ScrapingOutput]{Result: *output})
}
