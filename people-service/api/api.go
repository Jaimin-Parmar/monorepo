package api

import (
	"net/http"
	accountApipk "people-service/api/accountapi"
	"people-service/api/common"
	profileApipk "people-service/api/profileapi"

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

	/* ****************** ACCOUNT ****************** */
	accountAPI := accountApipk.New(a.Config, a.App.Repos, a.App)
	r.Handle("/ping", a.handler(accountAPI.Pong, false)).Methods(http.MethodGet)
	r.Handle("/user", a.handler(accountAPI.FetchAccountByID, true)).Methods(http.MethodGet)
	r.Handle("/contact", a.handler(accountAPI.FetchContacts, true)).Methods(http.MethodGet)
	r.Handle("/fetchAccounts", a.handler(accountAPI.FetchAccounts, false, true)).Methods(http.MethodGet)
	r.Handle("/createAccount", a.handler(accountAPI.CreateAccount, false)).Methods(http.MethodPost)
	r.Handle("/verifyVerificationCode", a.handler(accountAPI.VerifyVerificationCode, false)).Methods(http.MethodPost)
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

	/* ****************** PROFILE ****************** */
	profileAPI := profileApipk.New(a.Config, a.App.Repos, a.App)
	r.Handle("/user/{userID}/profiles", a.handler(profileAPI.GetProfiles, false, true)).Methods(http.MethodGet)
	r.Handle("/user/{userID}/profile/count", a.handler(profileAPI.GetProfileCount, false, true)).Methods(http.MethodGet)
	r.Handle("/user/{userID}/profile/set", a.handler(profileAPI.SetProfile, true)).Methods(http.MethodPost)
	r.Handle("/user/{userID}/profile/add", a.handler(profileAPI.AddProfile, false, true)).Methods(http.MethodPost)
	r.Handle("/user/{userID}/profile/edit", a.handler(profileAPI.EditProfile, true)).Methods(http.MethodPut)
	r.Handle("/user/{userID}/profile/info", a.handler(profileAPI.GetProfileInfo, true)).Methods(http.MethodGet)
	r.Handle("/user/{userID}/profile/tags", a.handler(profileAPI.GetProfileTags, true)).Methods(http.MethodGet)
	r.Handle("/user/{userID}/profile/delete", a.handler(profileAPI.DeleteProfile, true)).Methods(http.MethodDelete)

	r.Handle("/user/{userID}/profile/code/generate", a.handler(profileAPI.GenerateCode, true)).Methods(http.MethodPost)
	r.Handle("/user/{userID}/profile/code/delete", a.handler(profileAPI.DeleteCode, true)).Methods(http.MethodDelete)
	r.Handle("/user/{userID}/profile/settings", a.handler(profileAPI.UpdateProfileSettings, true)).Methods(http.MethodPut)
	r.Handle("/user/{userID}/profile/settings/shareable", a.handler(profileAPI.UpdateShareableSettings, true)).Methods(http.MethodPut)

	// CO-MANAGER
	r.Handle("/user/{userID}/profile/manage/send", a.handler(profileAPI.SendCoManagerRequest, true)).Methods(http.MethodPost)
	r.Handle("/user/{userID}/profile/manage/accept", a.handler(profileAPI.AcceptCoManagerRequest, true)).Methods(http.MethodPut)
	r.Handle("/user/{userID}/profile/manage/list", a.handler(profileAPI.ListProfilesWithCoManager, true)).Methods(http.MethodGet)
	r.Handle("/user/{userID}/profile/manage/external/list", a.handler(profileAPI.ListExternalProfiles, true)).Methods(http.MethodGet)
	r.Handle("/user/{userID}/profile/people/info", a.handler(profileAPI.GetPeopleInfo, true)).Methods(http.MethodPost)
	r.Handle("/user/{userID}/profile/boards/{type}", a.handler(profileAPI.FetchBoards, true)).Methods(http.MethodGet)
	r.Handle("/user/{userID}/profile/privacy/co-manager/list", a.handler(profileAPI.ListCoManagerCandidates, true)).Methods(http.MethodGet)

	// CONNECTIONS
	r.Handle("/user/{userID}/profile/connections", a.handler(profileAPI.MoveConnection, true)).Methods(http.MethodPut)
}
