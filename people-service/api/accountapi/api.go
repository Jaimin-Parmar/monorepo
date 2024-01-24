package accountapi

import (
	"people-service/api/common"
	"people-service/app/account"
	"people-service/cache"
	"people-service/model"
)

// API sidekiq api
type api struct {
	config         *common.Config
	cache          *cache.Cache
	accountService account.Service
}

// New creates a new api
func New(conf *common.Config, repos *model.Repos, accountService account.Service) *api {
	return &api{
		config:         conf,
		cache:          repos.Cache,
		accountService: accountService,
	}
}
