package bll

import (
	"context"

	"github.com/teambition/gear"
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
		Userbase:   &Userbase{svc: service.APIHost(cfg.Userbase)},
		Webscraper: &Webscraper{svc: service.APIHost(cfg.Webscraper)},
		Writing:    &Writing{svc: service.APIHost(cfg.Writing), oss: oss},
	}
}

func (b *Blls) Stats(ctx context.Context) (res map[string]any, err error) {
	return b.Userbase.svc.Stats(ctx)
}

type SuccessResponse[T any] struct {
	Retry         *int        `json:"retry,omitempty" cbor:"retry,omitempty"`
	TotalSize     *int        `json:"total_size,omitempty" cbor:"total_size,omitempty"`
	NextPageToken *util.Bytes `json:"next_page_token,omitempty" cbor:"next_page_token,omitempty"`
	Result        T           `json:"result" cbor:"result"`
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
	Status int8    `json:"status" cbor:"status"`
	MyRole *int8   `json:"_role,omitempty" cbor:"_role,omitempty"`
}

type Pagination struct {
	PageToken *util.Bytes `json:"page_token,omitempty" cbor:"page_token,omitempty"`
	PageSize  *int16      `json:"page_size,omitempty" cbor:"page_size,omitempty" validate:"omitempty,gte=5,lte=100"`
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
	PageSize  *int16      `json:"page_size,omitempty" cbor:"page_size,omitempty" validate:"omitempty,gte=5,lte=100"`
	Status    *int8       `json:"status,omitempty" cbor:"status,omitempty"`
	Fields    *[]string   `json:"fields,omitempty" cbor:"fields,omitempty"`
}

func (i *GIDPagination) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

func Int8Ptr(i int8) *int8 {
	return &i
}

func Int16Ptr(i int16) *int16 {
	return &i
}

func IntPtr(i int) *int {
	return &i
}

func StringPtr(str string) *string {
	return &str
}
