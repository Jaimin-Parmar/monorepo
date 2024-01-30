package profileapi

import (
	"people-service/api/common"
	"people-service/app"
	"people-service/cache"
	"people-service/model"
)

// API sidekiq api
type api struct {
	config *common.Config
	cache  *cache.Cache
	App    *app.App
}

// New creates a new api
func New(conf *common.Config, repos *model.Repos, app *app.App) *api {
	return &api{
		config: conf,
		cache:  repos.Cache,
		App:    app,
	}
}
