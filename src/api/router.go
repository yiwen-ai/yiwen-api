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
	Publication *Publication
	Scraping    *Scraping
}

func newAPIs(blls *bll.Blls) *APIs {
	return &APIs{
		Healthz:     &Healthz{blls},
		Creation:    &Creation{blls},
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

	router.Use(middleware.AuthToken(true).Auth)

	router.Get("/scraping", apis.Scraping.Create)

	router.Post("/creation", apis.Creation.Create)
	router.Get("/creation", apis.Creation.Create)
	router.Patch("/creation", apis.Creation.Create)
	router.Delete("/creation", apis.Creation.Create)

	router.Post("/creation/list", apis.Creation.Create)
	router.Patch("/creation/archive", apis.Creation.Create)
	router.Patch("/creation/redraft", apis.Creation.Create)
	router.Patch("/creation/review", todo)
	router.Patch("/creation/approve", todo)
	router.Patch("/creation/release", apis.Creation.Create)
	router.Put("/creation/update_content", apis.Creation.Create)
	router.Patch("/creation/update_content", todo)
	router.Post("/creation/assist", todo)

	router.Post("/publication", apis.Publication.Create)
	router.Get("/publication", apis.Publication.Create)
	router.Patch("/publication", apis.Publication.Create)
	router.Delete("/publication", apis.Publication.Create)

	router.Post("/publication/list", apis.Publication.Create)
	router.Patch("/publication/archive", apis.Publication.Create)
	router.Patch("/publication/redraft", apis.Publication.Create)
	router.Patch("/publication/approve", todo)
	router.Patch("/publication/publish", apis.Publication.Create)
	router.Put("/publication/update_content", apis.Publication.Create)
	router.Post("/publication/assist", todo)

	rx := gear.NewRouter()
	// health check
	rx.Get("/healthz", apis.Healthz.Get)

	return []*gear.Router{router, rx}
}
