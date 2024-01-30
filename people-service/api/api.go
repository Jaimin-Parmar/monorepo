package api

import (
	"net/http"
	accountApipk "people-service/api/accountapi"
	"people-service/api/common"

	"people-service/app"

	"people-service/cache"

	"github.com/gorilla/mux"
)

// API sidekiq api
type API struct {
	App    *app.App
	Config *common.Config
	Cache  *cache.Cache
}

// New creates a new api
func New(a *app.App) (api *API, err error) {
	api = &API{App: a}
	api.Config, err = common.InitConfig()
	if err != nil {
		return nil, err
	}
	return api, nil
}

func (a *API) Init(r *mux.Router) {
	accountAPI := accountApipk.New(a.Config, a.App.Repos, a.App.AccountService, a.App)
	r.Handle("/ping", a.handler(accountAPI.Pong, false)).Methods(http.MethodGet)
	r.Handle("/user", a.handler(accountAPI.FetchAccountByID, true)).Methods(http.MethodGet)
	r.Handle("/contact", a.handler(accountAPI.FetchContacts, true)).Methods(http.MethodGet)
	r.Handle("/fetchAccounts", a.handler(accountAPI.FetchAccounts, false, true)).Methods(http.MethodGet)
	r.Handle("/createAccount", a.handler(accountAPI.CreateAccount, false)).Methods(http.MethodPost)
	r.Handle("/getVerificationCode", a.handler(accountAPI.GetVerificationCode, false)).Methods(http.MethodPost)
	r.Handle("/verifyLink", a.handler(accountAPI.VerifyLink, false)).Methods(http.MethodPost)
	r.Handle("/forgotPassword", a.handler(accountAPI.ForgotPassword, false)).Methods(http.MethodPost)
	r.Handle("/resetPassword", a.handler(accountAPI.ResetPassword, false)).Methods(http.MethodPost)
	r.Handle("/setAccountType", a.handler(accountAPI.SetAccountType, false, true)).Methods(http.MethodPut)
	r.Handle("/account/service", a.handler(accountAPI.FetchAccountServices, true)).Methods(http.MethodGet)
	r.Handle("/pin/verify", a.handler(accountAPI.VerifyPin, false)).Methods(http.MethodPost)
	r.Handle("/account/info", a.handler(accountAPI.SetAccountInformation, false)).Methods(http.MethodPost)
	r.Handle("/account/info", a.handler(accountAPI.FetchAccountInformation, true)).Methods(http.MethodGet)
	r.Handle("/account/info", a.handler(accountAPI.UpdateAccountInfo, true)).Methods(http.MethodPut)
}
