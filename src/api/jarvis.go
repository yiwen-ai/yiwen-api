package api

import (
	"sync"
	"time"

	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Jarvis struct {
	blls *bll.Blls
}

func (a *Jarvis) ListLanguages(ctx *gear.Context) error {
	output, err := a.blls.Jarvis.ListLanguages(ctx)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[[][]string]{Result: output})
}

func (a *Jarvis) Search(ctx *gear.Context) error {
	input := &bll.SearchInput{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	rid := ctx.GetHeader("X-Request-Id")
	output := bll.SearchOutput{}
	var wg sync.WaitGroup
	wg.Add(2)

	now := time.Now()
	semanticElapsed := int64(0)
	literalElapsed := int64(0)
	go logging.Run(func() logging.Log {
		defer wg.Done()

		semanticInput := &bll.EmbeddingSearchInput{
			Input:  input.Q,
			Public: true,
			GID:    input.GID,
		}

		if input.Language != "" {
			semanticInput.Language = &input.Language
		}

		semanticOutput, err := a.blls.Jarvis.EmbeddingSearch(ctx, semanticInput)
		if err != nil {
			semanticElapsed = int64(time.Since(now)) / 1e6
			return logging.Log{
				"action":   "semantic_search",
				"rid":      rid,
				"gid":      input.GID.String(),
				"language": input.Language,
				"elapsed":  semanticElapsed,
				"error":    err.Error(),
			}
		}

		for _, item := range semanticOutput {
			if doc, err := a.blls.Writing.GetPublication(ctx, &bll.QueryPublication{
				GID:      item.GID,
				CID:      item.CID,
				Language: item.Language,
				Version:  item.Version,
				Fields:   "title,summary",
			}); err == nil {
				output.Hits = append(output.Hits, bll.SearchDocument{
					GID:      doc.GID,
					CID:      doc.CID,
					Language: doc.Language,
					Version:  doc.Version,
					Kind:     1,
					Title:    *doc.Title,
					Summary:  *doc.Summary,
				})
			}
		}

		semanticElapsed = int64(time.Since(now)) / 1e6
		return nil
	})

	var literalOutput bll.SearchOutput
	go logging.Run(func() logging.Log {
		defer wg.Done()

		literalOutput = a.blls.Writing.Search(ctx, input)
		literalElapsed = int64(time.Since(now)) / 1e6
		return nil
	})

	wg.Wait()
	output.Hits = append(output.Hits, literalOutput.Hits...)
	output.Languages = literalOutput.Languages
	(&output).LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	logging.SetTo(ctx, "semanticElapsed", semanticElapsed)
	logging.SetTo(ctx, "literalElapsed", literalElapsed)
	return ctx.OkSend(bll.SuccessResponse[bll.SearchOutput]{Result: output})
}

func (a *Jarvis) GroupSearch(ctx *gear.Context) error {
	input := &bll.SearchInput{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if input.GID == nil {
		return gear.ErrBadRequest.WithMsg("missing gid")
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, *input.GID)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}
	if role < -1 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	output := bll.SearchOutput{}
	var wg sync.WaitGroup
	wg.Add(2)

	now := time.Now()
	semanticElapsed := int64(0)
	literalElapsed := int64(0)
	go logging.Run(func() logging.Log {
		defer wg.Done()

		semanticInput := &bll.EmbeddingSearchInput{
			Input:  input.Q,
			Public: false,
			GID:    input.GID,
		}

		if input.Language != "" {
			semanticInput.Language = &input.Language
		}

		semanticOutput, err := a.blls.Jarvis.EmbeddingSearch(ctx, semanticInput)
		if err != nil {
			semanticElapsed = int64(time.Since(now)) / 1e6
			return logging.Log{
				"action":   "semantic_search",
				"rid":      sess.RID,
				"uid":      sess.UserID,
				"gid":      input.GID.String(),
				"language": input.Language,
				"elapsed":  semanticElapsed,
				"error":    err.Error(),
			}
		}

		for _, item := range semanticOutput {
			if doc, err := a.blls.Writing.GetPublication(ctx, &bll.QueryPublication{
				GID:      item.GID,
				CID:      item.CID,
				Language: item.Language,
				Version:  item.Version,
				Fields:   "title,summary",
			}); err == nil {
				output.Hits = append(output.Hits, bll.SearchDocument{
					GID:      doc.GID,
					CID:      doc.CID,
					Language: doc.Language,
					Version:  doc.Version,
					Kind:     1,
					Title:    *doc.Title,
					Summary:  *doc.Summary,
				})
			}
		}

		semanticElapsed = int64(time.Since(now)) / 1e6
		return nil
	})

	var literalOutput bll.SearchOutput
	go logging.Run(func() logging.Log {
		defer wg.Done()

		literalOutput = a.blls.Writing.GroupSearch(ctx, input)
		literalElapsed = int64(time.Since(now)) / 1e6
		return nil
	})

	wg.Wait()
	output.Hits = append(output.Hits, literalOutput.Hits...)
	output.Languages = literalOutput.Languages
	(&output).LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	logging.SetTo(ctx, "semanticElapsed", semanticElapsed)
	logging.SetTo(ctx, "literalElapsed", literalElapsed)
	return ctx.OkSend(bll.SuccessResponse[bll.SearchOutput]{Result: output})
}

func (a *Jarvis) OriginalSearch(ctx *gear.Context) error {
	input := &bll.ScrapingInput{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	sess := gear.CtxValue[middleware.Session](ctx)
	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, input.GID)
	if err != nil {
		return gear.ErrForbidden.From(err)
	}
	if role < -1 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	scraper, err := a.blls.Webscraper.Search(ctx, input.Url)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	// a unique Url generated by webscraper from input.Url
	input.Url = scraper.Url
	output := a.blls.Writing.OriginalSearch(ctx, input)
	(&output).LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})
	return ctx.OkSend(bll.SuccessResponse[bll.SearchOutput]{Result: output})
}
