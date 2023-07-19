package bll

import (
	"context"

	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

func init() {
	util.DigProvide(NewBlls)
}

// Blls ...
type Blls struct {
	Jarvis     *Jarvis
	Userbase   *Userbase
	Webscraper *Webscraper
	Writing    *Writing
}

// NewBlls ...
func NewBlls(oss *service.OSS) *Blls {
	cfg := conf.Config.Base
	return &Blls{
		Jarvis:     &Jarvis{svc: service.APIHost(cfg.Jarvis)},
		Userbase:   &Userbase{svc: service.APIHost(cfg.Userbase), oss: oss},
		Webscraper: &Webscraper{svc: service.APIHost(cfg.Webscraper)},
		Writing:    &Writing{svc: service.APIHost(cfg.Writing)},
	}
}

func (b *Blls) Stats(ctx context.Context) (res map[string]any, err error) {
	return b.Userbase.svc.Stats(ctx)
}

type SuccessResponse[T any] struct {
	Retry  int `json:"retry" cbor:"retry"`
	Result T   `json:"result" cbor:"result"`
}
