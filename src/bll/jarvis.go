package bll

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Jarvis struct {
	svc        service.APIHost
	tokensRate map[string]float32
	Languages  [][]string
}

func (b *Jarvis) InitApp(ctx context.Context, app *gear.App) error {
	output, err := b.ListLanguages(context.Background())
	if err != nil {
		return err
	}
	b.Languages = output
	b.tokensRate = make(map[string]float32, len(conf.Config.TokensRate))
	for _, vv := range output {
		if f, ok := conf.Config.TokensRate[vv[1]]; ok {
			b.tokensRate[vv[0]] = f
		}
	}

	return nil
}

func (b *Jarvis) getTokensRate(lang string) float32 {
	if v, ok := b.tokensRate[strings.ToLower(lang)]; ok {
		return v
	}
	return 1.0
}

func (b *Jarvis) EstimateTranslatingTokens(text, srcLang, dstLang string) uint32 {
	tokens := util.Tiktokens(text) + 100
	return tokens + uint32(float32(tokens)*b.getTokensRate(dstLang)/b.getTokensRate(srcLang))
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
	Content  util.Bytes `json:"content" cbor:"content" validate:"required"`
}

func (i *DetectLangInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}

	return nil
}

type TEInput struct {
	GID          util.ID     `json:"gid" cbor:"gid"`
	CID          util.ID     `json:"cid" cbor:"cid"`
	Language     string      `json:"language" cbor:"language"`
	Version      uint16      `json:"version" cbor:"version"`
	FromLanguage *string     `json:"from_language,omitempty" cbor:"from_language,omitempty"`
	Context      *string     `json:"context,omitempty" cbor:"context,omitempty"`
	Model        *string     `json:"model,omitempty" cbor:"model,omitempty"`
	Content      *util.Bytes `json:"content,omitempty" cbor:"content,omitempty"`
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
	GID       util.ID  `json:"gid" cbor:"gid"`
	CID       util.ID  `json:"cid" cbor:"cid"`
	Language  string   `json:"language" cbor:"language"`
	Version   uint16   `json:"version" cbor:"version"`
	Model     string   `json:"model" cbor:"model"`
	Tokens    uint32   `json:"tokens" cbor:"tokens"`
	Progress  int8     `json:"progress" cbor:"progress"`
	UpdatedAt int64    `json:"updated_at" cbor:"updated_at"`
	Summary   string   `json:"summary" cbor:"summary"`
	Keywords  []string `json:"keywords" cbor:"keywords"`
	Error     string   `json:"error" cbor:"error"`
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
		if i > 1200 {
			return nil, errors.New("summarizing timeout")
		}

		time.Sleep(time.Second * 3)
		output, err := b.GetSummary(ctx, getInput)
		if err != nil {
			return nil, err
		}

		if output.Progress == 100 && output.Summary != "" {
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
		if i > 1200 {
			return nil, errors.New("translating timeout")
		}

		time.Sleep(time.Second * 3)
		output, err := b.GetTranslation(ctx, getInput)
		if err != nil {
			return nil, err
		}

		if output.Progress == 100 && len(output.Content) > 0 {
			return output, nil
		}
	}
}

type TMInput struct {
	ID           util.ID     `json:"id" cbor:"id"`
	Language     string      `json:"language" cbor:"language"`
	Version      uint16      `json:"version" cbor:"version"`
	FromLanguage *string     `json:"from_language,omitempty" cbor:"from_language,omitempty"`
	Context      *string     `json:"context,omitempty" cbor:"context,omitempty"`
	Model        *string     `json:"model,omitempty" cbor:"model,omitempty"`
	Content      *util.Bytes `json:"content,omitempty" cbor:"content,omitempty"`
}

type TMOutput struct {
	Model    string     `json:"model" cbor:"model"`
	Progress int8       `json:"progress" cbor:"progress"`
	Tokens   uint32     `json:"tokens" cbor:"tokens"`
	Content  util.Bytes `json:"content" cbor:"content"`
	Error    string     `json:"error" cbor:"error"`
}

func (b *Jarvis) TranslateMessage(ctx context.Context, input *TMInput) (*TMOutput, error) {
	o0 := SuccessResponse[TMOutput]{}
	if err := b.svc.Post(ctx, "/v1/message/translating", input, &o0); err != nil {
		return nil, err
	}

	getInput := &TMInput{
		ID:       input.ID,
		Language: input.Language,
		Version:  input.Version,
	}

	i := 0
	for {
		i += 1
		if i > 1200 {
			return nil, errors.New("translating timeout")
		}

		time.Sleep(time.Second * 3)
		output, err := b.GetMessageTranslation(ctx, getInput)
		if err != nil {
			return nil, err
		}

		if output.Progress == 100 && len(output.Content) > 0 {
			return output, nil
		}
	}
}

func (b *Jarvis) GetMessageTranslation(ctx context.Context, input *TMInput) (*TMOutput, error) {
	output := SuccessResponse[TMOutput]{}

	err := b.svc.Post(ctx, "/v1/message/translating/get", input, &output)
	if err != nil {
		return nil, err
	}
	if output.Result.Error != "" {
		return nil, errors.New(output.Result.Error)
	}

	return &output.Result, nil
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

// func (b *Jarvis) Embedding(ctx context.Context, input *TEInput) (*TEOutput, error) {
// 	output := SuccessResponse[TEOutput]{}
// 	if err := b.svc.Post(ctx, "/v1/embedding", input, &output); err != nil {
// 		return nil, err
// 	}

// 	return &output.Result, nil
// }

// func (b *Jarvis) EmbeddingPublic(ctx context.Context, input *TEInput) {
// 	input.Content = nil
// 	output := SuccessResponse[any]{}
// 	if err := b.svc.Post(ctx, "/v1/embedding/public", input, &output); err != nil {
// 		logging.Warningf("Jarvis.EmbeddingPublic error: %v", err)
// 	}
// }

// type EmbeddingSearchInput struct {
// 	Input    string   `json:"input" cbor:"input"`
// 	Public   bool     `json:"public" cbor:"public"`
// 	GID      *util.ID `json:"gid,omitempty" cbor:"gid,omitempty"`
// 	Language *string  `json:"language,omitempty" cbor:"language,omitempty"`
// 	CID      *util.ID `json:"cid,omitempty" cbor:"cid,omitempty"`
// }

// type EmbeddingSearchOutput struct {
// 	GID      util.ID    `json:"gid" cbor:"gid"`
// 	CID      util.ID    `json:"cid" cbor:"cid"`
// 	Language string     `json:"language" cbor:"language"`
// 	Version  uint16     `json:"version" cbor:"version"`
// 	IDs      string     `json:"ids" cbor:"ids"`
// 	Content  util.Bytes `json:"content" cbor:"content"`
// }

// func (b *Jarvis) EmbeddingSearch(ctx context.Context, input *EmbeddingSearchInput) []*EmbeddingSearchOutput {
// 	output := SuccessResponse[[]*EmbeddingSearchOutput]{}
// 	b.svc.Post(ctx, "/v1/embedding/search", input, &output)
// 	return output.Result
// }
