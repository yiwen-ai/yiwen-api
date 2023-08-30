package bll

import (
	"context"
	"errors"
	"time"

	"github.com/yiwen-ai/yiwen-api/src/logging"
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
	GID      util.ID    `json:"gid" cbor:"gid" validate:"required"`
	Language string     `json:"language,omitempty" cbor:"language,omitempty"`
	Content  util.Bytes `json:"content" cbor:"content"`
}

type TEInput struct {
	GID      util.ID     `json:"gid" cbor:"gid"`
	CID      util.ID     `json:"cid" cbor:"cid"`
	Language string      `json:"language" cbor:"language"`
	Version  uint16      `json:"version" cbor:"version"`
	Model    *string     `json:"model,omitempty" cbor:"model,omitempty"`
	Content  *util.Bytes `json:"content,omitempty" cbor:"content,omitempty"`
}

type TEOutput struct {
	CID      util.ID `json:"cid" cbor:"cid"`
	Language string  `json:"detected_language" cbor:"detected_language"`
}

func (b *Jarvis) DetectLang(ctx context.Context, input *DetectLangInput) (*TEOutput, error) {
	output := SuccessResponse[TEOutput]{}
	if err := b.svc.Post(ctx, "/v1/translating/detect_language", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type SummarizingOutput struct {
	GID       util.ID `json:"gid" cbor:"gid"`
	CID       util.ID `json:"cid" cbor:"cid"`
	Language  string  `json:"language" cbor:"language"`
	Version   uint16  `json:"version" cbor:"version"`
	Model     string  `json:"model" cbor:"model"`
	Tokens    uint32  `json:"tokens" cbor:"tokens"`
	Progress  int8    `json:"progress" cbor:"progress"`
	UpdatedAt int64   `json:"updated_at" cbor:"updated_at"`
	Summary   string  `json:"summary" cbor:"summary"`
	Error     string  `json:"error" cbor:"error"`
}

func (b *Jarvis) Summarize(ctx context.Context, input *TEInput) (*SummarizingOutput, error) {
	o0 := SuccessResponse[TEOutput]{}
	if err := b.svc.Post(ctx, "/v1/summarizing", input, &o0); err != nil {
		return nil, err
	}

	getInput := &TEInput{
		GID:      input.GID,
		CID:      input.CID,
		Language: input.Language,
		Version:  input.Version,
	}

	i := 0
	for {
		i += 1
		if i > 400 {
			return nil, errors.New("summarizing timeout")
		}

		time.Sleep(time.Second * 3)
		output, err := b.GetSummary(ctx, getInput)
		if err != nil {
			return nil, err
		}

		if output.Summary != "" {
			return output, nil
		}
	}
}

func (b *Jarvis) GetSummary(ctx context.Context, input *TEInput) (*SummarizingOutput, error) {
	output := SuccessResponse[SummarizingOutput]{}

	err := b.svc.Post(ctx, "/v1/summarizing/get", input, &output)
	if err != nil {
		return nil, err
	}
	if output.Result.Error != "" {
		return nil, errors.New(output.Result.Error)
	}

	return &output.Result, nil
}

type TranslatingOutput struct {
	GID       util.ID    `json:"gid" cbor:"gid"`
	CID       util.ID    `json:"cid" cbor:"cid"`
	Language  string     `json:"language" cbor:"language"`
	Version   uint16     `json:"version" cbor:"version"`
	Model     string     `json:"model" cbor:"model"`
	Progress  int8       `json:"progress" cbor:"progress"`
	UpdatedAt int64      `json:"updated_at" cbor:"updated_at"`
	Tokens    uint32     `json:"tokens" cbor:"tokens"`
	Content   util.Bytes `json:"content" cbor:"content"`
	Error     string     `json:"error" cbor:"error"`
}

func (b *Jarvis) Translate(ctx context.Context, input *TEInput) (*TranslatingOutput, error) {
	o0 := SuccessResponse[TEOutput]{}
	if err := b.svc.Post(ctx, "/v1/translating", input, &o0); err != nil {
		return nil, err
	}

	getInput := &TEInput{
		GID:      input.GID,
		CID:      input.CID,
		Language: input.Language,
		Version:  input.Version,
	}

	i := 0
	for {
		i += 1
		if i > 400 {
			return nil, errors.New("translating timeout")
		}

		time.Sleep(time.Second * 3)
		output, err := b.GetTranslation(ctx, getInput)
		if err != nil {
			return nil, err
		}

		if len(output.Content) > 1 {
			return output, nil
		}
	}
}

func (b *Jarvis) GetTranslation(ctx context.Context, input *TEInput) (*TranslatingOutput, error) {
	output := SuccessResponse[TranslatingOutput]{}

	err := b.svc.Post(ctx, "/v1/translating/get", input, &output)
	if err != nil {
		return nil, err
	}
	if output.Result.Error != "" {
		return nil, errors.New(output.Result.Error)
	}

	return &output.Result, nil
}

func (b *Jarvis) Embedding(ctx context.Context, input *TEInput) (*TEOutput, error) {
	output := SuccessResponse[TEOutput]{}
	if err := b.svc.Post(ctx, "/v1/embedding", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Jarvis) EmbeddingPublic(ctx context.Context, input *TEInput) {
	input.Content = nil
	output := SuccessResponse[any]{}
	if err := b.svc.Post(ctx, "/v1/embedding/public", input, &output); err != nil {
		logging.Warningf("Jarvis.EmbeddingPublic error: %v", err)
	}
}

type EmbeddingSearchInput struct {
	Input    string   `json:"input" cbor:"input"`
	Public   bool     `json:"public" cbor:"public"`
	GID      *util.ID `json:"gid,omitempty" cbor:"gid,omitempty"`
	Language *string  `json:"language,omitempty" cbor:"language,omitempty"`
	CID      *util.ID `json:"cid,omitempty" cbor:"cid,omitempty"`
}

type EmbeddingSearchOutput struct {
	GID      util.ID    `json:"gid" cbor:"gid"`
	CID      util.ID    `json:"cid" cbor:"cid"`
	Language string     `json:"language" cbor:"language"`
	Version  uint16     `json:"version" cbor:"version"`
	IDs      string     `json:"ids" cbor:"ids"`
	Content  util.Bytes `json:"content" cbor:"content"`
}

func (b *Jarvis) EmbeddingSearch(ctx context.Context, input *EmbeddingSearchInput) ([]*EmbeddingSearchOutput, error) {
	output := SuccessResponse[[]*EmbeddingSearchOutput]{}
	if err := b.svc.Post(ctx, "/v1/embedding/search", input, &output); err != nil {
		return nil, err
	}

	return output.Result, nil
}
