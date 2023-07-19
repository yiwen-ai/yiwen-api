package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
)

type Publication struct {
	blls *bll.Blls
}

func (a *Publication) Create(ctx *gear.Context) error {
	return nil
}
