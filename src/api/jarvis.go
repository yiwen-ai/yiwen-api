package api

import (
	"strings"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/content"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Jarvis struct {
	blls *bll.Blls
}

func (a *Jarvis) ListLanguages(ctx *gear.Context) error {
	return ctx.OkSend(bll.SuccessResponse[[][]string]{Result: a.blls.Jarvis.Languages})
}

func (a *Jarvis) ListModels(ctx *gear.Context) error {
	return ctx.OkSend(bll.SuccessResponse[[]bll.AIModel]{Result: bll.AIModels})
}

func (a *Jarvis) Search(ctx *gear.Context) error {
	input := &bll.SearchInput{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	lang := ""
	if sess := gear.CtxValue[middleware.Session](ctx); sess != nil {
		lang = sess.Lang
	}

	output := bll.SearchOutput{Hits: []bll.SearchDocument{}, Languages: map[string]int{}}
	if strings.TrimSpace(input.Q) == "" {
		res, err := a.blls.Writing.ListLatestPublications(ctx, &bll.Pagination{
			Fields: util.Ptr([]string{"updated_at", "title", "summary"}),
		})
		if err != nil {
			return ctx.OkSend(bll.SuccessResponse[bll.SearchOutput]{Result: output})
		}

		output.Hits = make([]bll.SearchDocument, 0, len(res.Result))
		for _, doc := range res.Result {
			output.Hits = append(output.Hits, bll.SearchDocument{
				GID:       doc.GID,
				CID:       doc.CID,
				Language:  doc.Language,
				Version:   doc.Version,
				UpdatedAt: *doc.UpdatedAt,
				Kind:      1,
				Title:     *doc.Title,
				Summary:   *doc.Summary,
			})
		}
	} else {
		var wg sync.WaitGroup
		wg.Add(1)

		now := time.Now()
		// semanticElapsed := int64(0)
		literalElapsed := int64(0)

		// var semanticOutput []*bll.EmbeddingSearchOutput
		// go logging.Run(func() logging.Log {
		// 	defer wg.Done()

		// 	semanticInput := &bll.EmbeddingSearchInput{
		// 		Input:    input.Q,
		// 		Public:   true,
		// 		GID:      input.GID,
		// 		Language: input.Language,
		// 	}

		// 	semanticOutput = a.blls.Jarvis.EmbeddingSearch(ctx, semanticInput)
		// 	semanticElapsed = int64(time.Since(now)) / 1e6
		// 	return nil
		// })

		var literalOutput bll.SearchOutput
		go logging.Run(func() logging.Log {
			defer wg.Done()

			literalOutput = a.blls.Writing.Search(ctx, input)
			literalElapsed = int64(time.Since(now)) / 1e6
			return nil
		})

		wg.Wait()
		// logging.SetTo(ctx, "semanticResults", len(semanticOutput))
		logging.SetTo(ctx, "literalResults", len(literalOutput.Hits))
		// logging.SetTo(ctx, "semanticElapsed", semanticElapsed)
		logging.SetTo(ctx, "literalElapsed", literalElapsed)

		output.Hits = make([]bll.SearchDocument, 0, len(literalOutput.Hits))
		// append(output.Hits, literalOutput.Hits...)
		output.Languages = literalOutput.Languages
		resMap := make(map[util.ID]int, len(literalOutput.Hits))
		for i, item := range literalOutput.Hits {
			j, ok := resMap[item.CID]
			if ok && item.Language != lang {
				continue
			}

			v := literalOutput.Hits[i]
			if ok {
				output.Hits[j] = v
			} else {
				output.Hits = append(output.Hits, v)
				resMap[item.CID] = len(output.Hits) - 1
			}
		}

		// for _, item := range semanticOutput {
		// 	j, ok := resMap[item.CID]
		// 	if ok && item.Language != lang {
		// 		continue
		// 	}

		// 	if doc, err := a.blls.Writing.ImplicitGetPublication(ctx, &bll.ImplicitQueryPublication{
		// 		CID:      item.CID,
		// 		Language: item.Language,
		// 		Fields:   "status,updated_at,title,summary",
		// 	}, nil); err == nil && *doc.Status == 2 {
		// 		v := bll.SearchDocument{
		// 			GID:       doc.GID,
		// 			CID:       doc.CID,
		// 			Language:  doc.Language,
		// 			Version:   doc.Version,
		// 			UpdatedAt: *doc.UpdatedAt,
		// 			Kind:      1,
		// 			Title:     *doc.Title,
		// 			Summary:   *doc.Summary,
		// 		}

		// 		if ok {
		// 			output.Hits[j] = v
		// 		} else {
		// 			output.Hits = append(output.Hits, v)
		// 			resMap[item.CID] = len(output.Hits) - 1
		// 		}
		// 	}
		// }
	}

	(&output).LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

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

	lang := sess.Lang
	// if input.Language == nil && lang != "" {
	// 	input.Language = util.Ptr(lang)
	// }

	output := bll.SearchOutput{}

	now := time.Now()
	literalOutput := a.blls.Writing.GroupSearch(ctx, input)
	literalElapsed := int64(time.Since(now)) / 1e6
	logging.SetTo(ctx, "literalResults", len(literalOutput.Hits))
	logging.SetTo(ctx, "literalElapsed", literalElapsed)

	output.Hits = make([]bll.SearchDocument, 0, len(literalOutput.Hits))
	output.Languages = literalOutput.Languages
	resMap := make(map[util.ID]int, len(literalOutput.Hits))
	for i, item := range literalOutput.Hits {
		j, ok := resMap[item.CID]
		if ok && item.Language != lang {
			continue
		}

		v := literalOutput.Hits[i]
		if ok {
			output.Hits[j] = v
		} else {
			output.Hits = append(output.Hits, v)
			resMap[item.CID] = len(output.Hits) - 1
		}
	}

	(&output).LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

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

func (a *Jarvis) DetectLang(ctx *gear.Context) error {
	input := &bll.DetectLangInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	doc, err := content.ParseDocumentNode(input.Content)
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}
	teContents := doc.ToTEContents()
	if len(teContents) == 0 {
		return gear.ErrBadRequest.WithMsg("invalid content")
	}

	teData, err := cbor.Marshal(teContents)
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}

	output, err := a.blls.Jarvis.DetectLang(ctx, &bll.DetectLangInput{
		GID:      util.ANON,
		Language: input.Language,
		Content:  teData,
	})

	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.TEOutput]{Result: output})
}
