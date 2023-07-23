package bll

import (
	"context"

	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Jarvis struct {
	svc service.APIHost
}

func (b *Jarvis) ListLanguages(ctx context.Context) ([][]string, error) {
	output := SuccessResponse[[][]string]{}
	if err := b.svc.Get(ctx, "/v1/translating/list_languages", &output); err != nil {
		return nil, err
	}

	return output.Result, nil
}

type DetectLangInput struct {
	GID      util.ID            `json:"gid" cbor:"gid" validate:"required"`
	Language string             `json:"language,omitempty" cbor:"language,omitempty"`
	Content  content.TEContents `json:"content" cbor:"content"`
}

type TEOutput struct {
	CID      util.ID `json:"cid" cbor:"cid"`
	Language string  `json:"detected_language" cbor:"detected_language"`
}

func (b *Jarvis) DetectLang(ctx context.Context, input DetectLangInput) (*TEOutput, error) {
	output := SuccessResponse[TEOutput]{}
	if err := b.svc.Post(ctx, "/v1/translating/detect_language", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}
