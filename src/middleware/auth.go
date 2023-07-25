package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Session struct {
	// cookie session 验证即可
	UserID util.ID

	// 以下字段需要 access token 验证
	AppID      util.ID
	UserStatus int
	UserRating int
	UserKind   int
	AppScope   []string
	RID        string // x-request-id
}

func (s *Session) HasToken() bool {
	return s.AppID != util.ZeroID
}

func (s *Session) HasScope(scope string) bool {
	return util.StringSliceHas(s.AppScope, scope)
}

type AuthLevel uint8

const (
	AuthAllowAnon AuthLevel = iota
	AuthSession             // cookie session
	AuthToken               // access token
)

func (m AuthLevel) Auth(ctx *gear.Context) error {
	l := uint8(m)
	sess, err := extractAuth(ctx)
	if err != nil {
		if l == 0 {
			return nil
		}

		return gear.ErrUnauthorized.From(err)
	}

	if l > 1 && !sess.HasToken() {
		return gear.ErrUnauthorized.WithMsg("invalid token")
	}

	ctxHeader := make(http.Header)
	// inject auth headers into context for base service
	util.CopyHeader(ctxHeader, ctx.Req.Header,
		"x-real-ip",
		"x-request-id",
		"x-auth-user",
		"x-auth-user-rating",
		"x-auth-app",
	)

	cctx := gear.CtxWith[Session](ctx.Context(), sess)
	cheader := util.ContextHTTPHeader(ctxHeader)
	ctx.WithContext(gear.CtxWith[util.ContextHTTPHeader](cctx, &cheader))
	return nil
}

func WithGlobalCtx(ctx *gear.Context) context.Context {
	gctx := conf.Config.GlobalCtx

	if sess := gear.CtxValue[Session](ctx); sess != nil {
		gctx = gear.CtxWith[Session](gctx, sess)
		gctx = gear.CtxWith[util.ContextHTTPHeader](gctx, gear.CtxValue[util.ContextHTTPHeader](ctx))
	}

	return gctx
}

func extractAuth(ctx *gear.Context) (*Session, error) {
	var err error
	sess := &Session{}
	sess.UserID, _ = util.ParseID(ctx.GetHeader("x-auth-user"))
	if sess.UserID == util.ZeroID {
		return nil, gear.ErrUnauthorized.WithMsg("invalid session")
	}

	sess.RID = ctx.GetHeader("x-request-id")
	sess.AppID, err = util.ParseID(ctx.GetHeader("x-auth-app"))
	if err == nil {
		if sess.UserStatus, err = strconv.Atoi(ctx.GetHeader("x-auth-user-status")); err != nil {
			return nil, gear.ErrUnauthorized.WithMsg("invalid user status")
		}
		if sess.UserRating, err = strconv.Atoi(ctx.GetHeader("x-auth-user-rating")); err != nil {
			return nil, gear.ErrUnauthorized.WithMsg("invalid user rating")
		}
		if sess.UserKind, err = strconv.Atoi(ctx.GetHeader("x-auth-user-kind")); err != nil {
			return nil, gear.ErrUnauthorized.WithMsg("invalid user kind")
		}
		if scope := ctx.GetHeader("x-auth-app-scope"); scope != "" {
			sess.AppScope = strings.Split(scope, ",")
		}
	}

	return sess, nil
}
