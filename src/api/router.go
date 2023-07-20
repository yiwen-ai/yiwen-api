package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

func init() {
	util.DigProvide(newAPIs)
	util.DigProvide(newRouters)
}

// APIs ..
type APIs struct {
	Healthz     *Healthz
	Creation    *Creation
	Jarvis      *Jarvis
	Publication *Publication
	Scraping    *Scraping
}

func newAPIs(blls *bll.Blls) *APIs {
	return &APIs{
		Healthz:     &Healthz{blls},
		Creation:    &Creation{blls},
		Jarvis:      &Jarvis{blls},
		Publication: &Publication{blls},
		Scraping:    &Scraping{blls},
	}
}

func todo(ctx *gear.Context) error {
	return gear.ErrNotImplemented.WithMsgf("TODO: %s %s", ctx.Method, ctx.Path)
}

func newRouters(apis *APIs) []*gear.Router {

	router := gear.NewRouter(gear.RouterOptions{
		Root:                  "/v1",
		IgnoreCase:            false,
		FixedPathRedirect:     false,
		TrailingSlashRedirect: false,
	})

	// /v1/xxx 都需要认证
	router.Use(middleware.AuthToken(true).Auth)

	router.Get("/scraping", apis.Scraping.Create)
	router.Get("/search/in_group", apis.Jarvis.GroupSearch)
	router.Get("/search/by_original_url", apis.Jarvis.OriginalSearch)

	router.Post("/creation", apis.Creation.Create)
	router.Get("/creation", apis.Creation.Get)
	router.Patch("/creation", apis.Creation.Update)
	router.Delete("/creation", apis.Creation.Delete)

	router.Post("/creation/list", apis.Creation.List)
	router.Patch("/creation/archive", apis.Creation.Archive)
	router.Patch("/creation/redraft", apis.Creation.Redraft)
	router.Patch("/creation/review", todo)
	router.Patch("/creation/approve", todo)
	router.Patch("/creation/release", apis.Creation.Release)
	router.Put("/creation/update_content", apis.Creation.UpdateContent)
	router.Patch("/creation/update_content", todo)
	router.Post("/creation/assist", todo)

	router.Post("/publication", todo)
	router.Get("/publication", todo)
	router.Patch("/publication", todo)
	router.Delete("/publication", todo)

	router.Post("/publication/list", todo)
	router.Patch("/publication/archive", todo)
	router.Patch("/publication/redraft", todo)
	router.Patch("/publication/approve", todo)
	router.Patch("/publication/publish", todo)
	router.Put("/publication/update_content", todo)
	router.Post("/publication/assist", todo)

	// 以下 API 不需要认证
	rx := gear.NewRouter()
	rx.Get("/healthz", apis.Healthz.Get)
	rx.Get("/languages", apis.Jarvis.ListLanguages)
	// 搜索公开内容
	rx.Get("/search", apis.Jarvis.Search)

	return []*gear.Router{router, rx}
}
