package bll

import (
	"context"

	"github.com/pkoukk/tiktoken-go"
	"github.com/pkoukk/tiktoken-go-loader"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

func init() {
	util.DigProvide(NewBlls)
	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
}

// Blls ...
type Blls struct {
	tk         *tiktoken.Tiktoken
	Locker     *service.Locker
	Jarvis     *Jarvis
	Logbase    *Logbase
	Taskbase   *Taskbase
	Userbase   *Userbase
	Walletbase *Walletbase
	Webscraper *Webscraper
	Writing    *Writing
}

// NewBlls ...
func NewBlls(oss *service.OSS, locker *service.Locker) *Blls {
	cfg := conf.Config.Base
	tk, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		panic(err)
	}
	return &Blls{
		tk:         tk,
		Locker:     locker,
		Jarvis:     &Jarvis{svc: service.APIHost(cfg.Jarvis)},
		Logbase:    &Logbase{svc: service.APIHost(cfg.Logbase)},
		Taskbase:   &Taskbase{svc: service.APIHost(cfg.Taskbase)},
		Userbase:   &Userbase{svc: service.APIHost(cfg.Userbase)},
		Walletbase: &Walletbase{svc: service.APIHost(cfg.Walletbase)},
		Webscraper: &Webscraper{svc: service.APIHost(cfg.Webscraper)},
		Writing:    &Writing{svc: service.APIHost(cfg.Writing), oss: oss},
	}
}

func (b *Blls) Stats(ctx context.Context) (res map[string]any, err error) {
	return b.Userbase.svc.Stats(ctx)
}

func (b *Blls) Tiktokens(input string) uint32 {
	return uint32(len(b.tk.Encode(input, nil, nil)))
}

type SuccessResponse[T any] struct {
	Retry         int        `json:"retry,omitempty" cbor:"retry,omitempty"`
	TotalSize     int        `json:"total_size,omitempty" cbor:"total_size,omitempty"`
	NextPageToken util.Bytes `json:"next_page_token,omitempty" cbor:"next_page_token,omitempty"`
	Job           string     `json:"job,omitempty" cbor:"job,omitempty"`
	Result        T          `json:"result" cbor:"result"`
}

type UserInfo struct {
	ID      *util.ID `json:"id,omitempty" cbor:"id,omitempty"` // should clear this field when return to client
	CN      string   `json:"cn" cbor:"cn"`
	Name    string   `json:"name" cbor:"name"`
	Picture string   `json:"picture" cbor:"picture"`
	Status  int8     `json:"status" cbor:"status"`
	Kind    int8     `json:"kind" cbor:"kind"`
}

type GroupInfo struct {
	ID     util.ID `json:"id" cbor:"id"`
	CN     string  `json:"cn" cbor:"cn"`
	Name   string  `json:"name" cbor:"name"`
	Logo   string  `json:"logo" cbor:"logo"`
	Slogan string  `json:"slogan" cbor:"slogan"`
	Status int8    `json:"status" cbor:"status"`
	MyRole *int8   `json:"_role,omitempty" cbor:"_role,omitempty"`
}

type Pagination struct {
	PageToken *util.Bytes `json:"page_token,omitempty" cbor:"page_token,omitempty"`
	PageSize  *uint16     `json:"page_size,omitempty" cbor:"page_size,omitempty" validate:"omitempty,gte=5,lte=100"`
	Status    *int8       `json:"status,omitempty" cbor:"status,omitempty"`
	Fields    *[]string   `json:"fields,omitempty" cbor:"fields,omitempty"`
}

func (i *Pagination) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type GIDPagination struct {
	GID       util.ID     `json:"gid" cbor:"gid" validate:"required"`
	PageToken *util.Bytes `json:"page_token,omitempty" cbor:"page_token,omitempty"`
	PageSize  *uint16     `json:"page_size,omitempty" cbor:"page_size,omitempty" validate:"omitempty,gte=5,lte=100"`
	Status    *int8       `json:"status,omitempty" cbor:"status,omitempty"`
	Fields    *[]string   `json:"fields,omitempty" cbor:"fields,omitempty"`
}

func (i *GIDPagination) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type QueryIdCn struct {
	ID     *util.ID `json:"id,omitempty" cbor:"id,omitempty" query:"id"`
	CN     *string  `json:"cn,omitempty" cbor:"cn,omitempty" query:"cn"`
	Fields *string  `json:"fields,omitempty" cbor:"fields,omitempty" query:"fields"`
}

func (i *QueryIdCn) Validate() error {
	if i.ID == nil && i.CN == nil {
		return gear.ErrBadRequest.WithMsg("id or cn is required")
	}
	return nil
}
