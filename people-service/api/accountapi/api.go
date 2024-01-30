package accountapi

import (
	"people-service/api/common"
	"people-service/app"
	"people-service/app/account"
	"people-service/cache"
	"people-service/model"
)

// API sidekiq api
type api struct {
	config         *common.Config
	cache          *cache.Cache
	accountService account.Service
	App            *app.App
}

// New creates a new api
func New(conf *common.Config, repos *model.Repos, accountService account.Service, app *app.App) *api {
	return &api{
		config:         conf,
		cache:          repos.Cache,
		accountService: accountService,
		App:            app,
	}
}
