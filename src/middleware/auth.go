package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/teambition/gear"
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
}

func (s *Session) HasToken() bool {
	return s.AppID != util.ZeroID
}

func (s *Session) HasScope(scope string) bool {
	return util.StringSliceHas(s.AppScope, scope)
}

type AuthToken bool

func (m AuthToken) Auth(ctx *gear.Context) error {
	sess, err := extractAuth(ctx)
	if err != nil {
		return gear.ErrUnauthorized.From(err)
	}

	if bool(m) && !sess.HasToken() {
		return gear.ErrUnauthorized.WithMsg("invalid token")
	}

	ctxHeader := make(http.Header)
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

func extractAuth(ctx *gear.Context) (*Session, error) {
	var err error
	sess := &Session{}
	sess.UserID, _ = util.ParseID(ctx.GetHeader("x-auth-user"))
	if sess.UserID == util.ZeroID {
		return nil, gear.ErrUnauthorized.WithMsg("invalid session")
	}

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
