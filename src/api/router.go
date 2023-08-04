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
	Group       *Group
	Jarvis      *Jarvis
	Publication *Publication
	Scraping    *Scraping
}

func newAPIs(blls *bll.Blls) *APIs {
	return &APIs{
		Healthz:     &Healthz{blls},
		Creation:    &Creation{blls},
		Group:       &Group{blls},
		Jarvis:      &Jarvis{blls},
		Publication: &Publication{blls},
		Scraping:    &Scraping{blls},
	}
}

func todo(ctx *gear.Context) error {
	return gear.ErrNotImplemented.WithMsgf("TODO: %s %s", ctx.Method, ctx.Path)
}

func newRouters(apis *APIs) []*gear.Router {

	router := gear.NewRouter()
	router.Get("/healthz", apis.Healthz.Get)

	// 允许匿名访问
	router.Get("/languages", middleware.AuthAllowAnon.Auth, apis.Jarvis.ListLanguages)
	router.Get("/search", middleware.AuthAllowAnon.Auth, apis.Jarvis.Search)
	router.Get("/v1/publication", middleware.AuthAllowAnon.Auth, apis.Publication.Get)
	router.Get("/v1/publication/publish", middleware.AuthAllowAnon.Auth, apis.Publication.GetPublishList)
	router.Post("/v1/publication/list_published", middleware.AuthAllowAnon.Auth, apis.Publication.ListPublished)

	router.Get("/v1/search", middleware.AuthToken.Auth, apis.Jarvis.Search)
	router.Get("/v1/search/in_group", middleware.AuthToken.Auth, apis.Jarvis.GroupSearch)
	router.Get("/v1/search/by_original_url", middleware.AuthToken.Auth, apis.Jarvis.OriginalSearch)

	router.Get("/v1/scraping", middleware.AuthToken.Auth, apis.Scraping.Create)

	router.Post("/v1/creation", middleware.AuthToken.Auth, apis.Creation.Create)
	router.Get("/v1/creation", middleware.AuthToken.Auth, apis.Creation.Get)
	router.Patch("/v1/creation", middleware.AuthToken.Auth, apis.Creation.Update)
	router.Delete("/v1/creation", middleware.AuthToken.Auth, apis.Creation.Delete)

	router.Post("/v1/creation/list", middleware.AuthToken.Auth, apis.Creation.List)
	router.Post("/v1/creation/list_archived", middleware.AuthToken.Auth, apis.Creation.ListArchived)
	router.Patch("/v1/creation/archive", middleware.AuthToken.Auth, apis.Creation.Archive)
	router.Patch("/v1/creation/redraft", middleware.AuthToken.Auth, apis.Creation.Redraft)
	router.Patch("/v1/creation/review", middleware.AuthToken.Auth, todo)  // 暂不实现
	router.Patch("/v1/creation/approve", middleware.AuthToken.Auth, todo) // 暂不实现
	router.Post("/v1/creation/release", middleware.AuthToken.Auth, apis.Creation.Release)
	router.Put("/v1/creation/update_content", middleware.AuthToken.Auth, apis.Creation.UpdateContent)
	router.Patch("/v1/creation/update_content", middleware.AuthToken.Auth, todo) // 暂不实现
	router.Post("/v1/creation/assist", middleware.AuthToken.Auth, todo)          // 暂不实现

	router.Post("/v1/publication", middleware.AuthToken.Auth, apis.Publication.Create)
	router.Patch("/v1/publication", middleware.AuthToken.Auth, apis.Publication.Update)
	router.Delete("/v1/publication", middleware.AuthToken.Auth, apis.Publication.Delete)

	router.Get("/v1/publication/by_job", middleware.AuthToken.Auth, apis.Publication.GetByJob)
	router.Get("/v1/publication/list_job", middleware.AuthToken.Auth, apis.Publication.ListJob)
	router.Post("/v1/publication/list", middleware.AuthToken.Auth, apis.Publication.List)
	router.Post("/v1/publication/list_by_following", middleware.AuthToken.Auth, apis.Publication.ListByFollowing)
	router.Post("/v1/publication/list_archived", middleware.AuthToken.Auth, apis.Publication.ListArchived)
	router.Patch("/v1/publication/archive", middleware.AuthToken.Auth, apis.Publication.Archive)
	router.Patch("/v1/publication/redraft", middleware.AuthToken.Auth, apis.Publication.Redraft)
	router.Patch("/v1/publication/publish", middleware.AuthToken.Auth, apis.Publication.Publish)
	router.Put("/v1/publication/update_content", middleware.AuthToken.Auth, apis.Publication.UpdateContent)
	router.Post("/v1/publication/assist", middleware.AuthToken.Auth, todo) // 暂不实现

	router.Patch("/v1/group/follow", middleware.AuthToken.Auth, apis.Group.Follow)
	router.Patch("/v1/group/unfollow", middleware.AuthToken.Auth, apis.Group.UnFollow)
	router.Post("/v1/group/list_my", middleware.AuthToken.Auth, apis.Group.ListMy)
	router.Post("/v1/group/list_following", middleware.AuthToken.Auth, apis.Group.ListFollowing)
	router.Post("/v1/group/list_subscribing", middleware.AuthToken.Auth, todo) // 暂不实现

	return []*gear.Router{router}
}
