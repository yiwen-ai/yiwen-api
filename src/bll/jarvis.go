package bll

import (
	"context"
	"errors"
	"time"

	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

const DefaultModel = "gpt3.5"

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
	GID      util.ID `json:"gid" cbor:"gid"`
	CID      util.ID `json:"cid" cbor:"cid"`
	Language string  `json:"language" cbor:"language"`
	Version  uint16  `json:"version" cbor:"version"`
	Model    string  `json:"model" cbor:"model"`
	Tokens   uint32  `json:"tokens" cbor:"tokens"`
	Summary  string  `json:"summary" cbor:"summary"`
	Error    string  `json:"error" cbor:"error"`
}

func (b *Jarvis) Summarize(ctx context.Context, input *TEInput) (*SummarizingOutput, error) {
	// output := SuccessResponse[string]{}
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
	output := SuccessResponse[*SummarizingOutput]{}
	i := 0
	for {
		time.Sleep(time.Second * 3)
		i += 1
		err := b.svc.Post(ctx, "/v1/summarizing/get", getInput, &output)
		if err != nil && !errors.Is(err, util.ErrNotFound) {
			return nil, err
		}

		if i > 100 {
			return nil, errors.New("summarizing timeout")
		}

		if err == nil {
			break
		}
	}

	if output.Result.Error != "" {
		return nil, errors.New(output.Result.Error)
	}

	return output.Result, nil
}

type TranslatingOutput struct {
	GID      util.ID    `json:"gid" cbor:"gid"`
	CID      util.ID    `json:"cid" cbor:"cid"`
	Language string     `json:"language" cbor:"language"`
	Version  uint16     `json:"version" cbor:"version"`
	Model    string     `json:"model" cbor:"model"`
	Tokens   uint32     `json:"tokens" cbor:"tokens"`
	Content  util.Bytes `json:"content" cbor:"content"`
	Error    string     `json:"error" cbor:"error"`
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
	output := SuccessResponse[*TranslatingOutput]{}
	i := 0
	for {
		time.Sleep(time.Second * 3)
		i += 1
		err := b.svc.Post(ctx, "/v1/translating/get", getInput, &output)
		if err != nil && !errors.Is(err, util.ErrNotFound) {
			return nil, err
		}

		if i > 100 {
			return nil, errors.New("translating timeout")
		}

		if err == nil {
			break
		}
	}

	if output.Result.Error != "" {
		return nil, errors.New(output.Result.Error)
	}

	return output.Result, nil
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
