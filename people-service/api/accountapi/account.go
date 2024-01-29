package accountapi

import (
	"encoding/json"
	"net/http"
	"people-service/app"
	"people-service/model"
	"people-service/util"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// Pong Api
func (a *api) Pong(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	time.Sleep(5 * time.Minute)
	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "pong"))
	return nil
}

// GetAccounts
func (a *api) FetchAccounts(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	res, err := a.accountService.FetchAccounts()
	if err != nil {
		return err
	}
	json.NewEncoder(w).Encode(res)
	return nil
}

// FetchAccountByID - fetch account by ID
func (a *api) FetchAccountByID(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	user, err := a.accountService.FetchAccount(ctx.User.ID, false)
	if err != nil {
		return err
	}
	user.Token = ctx.User.Token
	user.WriteToJSON(w)
	return nil
}

func (a *api) FetchContacts(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	contacts, err := a.accountService.FetchContacts(ctx.User.ID)
	if err != nil {
		return err
	}
	json.NewEncoder(w).Encode(contacts)
	return nil
}

func (a *api) CreateAccount(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload model.AccountSignup
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}
	res, err := a.accountService.CreateAccount(payload)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) GetVerificationCode(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload model.RegistrationUser
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}

	if payload.Type == "email" {
		res, err := a.accountService.GetVerificationCode(payload.ID, payload.Email)
		if err != nil {
			return err
		}
		json.NewEncoder(w).Encode(res)
	}
	return nil
}

func (a *api) VerifyLink(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload struct {
		Token string `json:"token" db:"token"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}
	res, err := a.accountService.VerifyLink(payload.Token)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

// ForgotPassword - Here we send reset password link on the recipient email
func (a *api) ForgotPassword(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload struct {
		Email string `json:"email" db:"email"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}
	res, err := a.accountService.ForgotPassword(payload.Email)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

// ResetPassword
func (a *api) ResetPassword(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload *model.ResetPassword
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}
	res, err := a.accountService.ResetPassword(payload)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

// SetAccountType
func (a *api) SetAccountType(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload *model.SetAccountType
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}
	payload.AccountId = strconv.Itoa(ctx.User.ID)
	res, err := a.accountService.SetAccountType(payload)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) FetchAccountServices(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	var res map[string]interface{}

	accInfo, err := a.accountService.FetchAccountInformation(ctx.User.ID)
	if err == nil {
		res, err = a.accountService.FetchAccountServices(accInfo["data"].(model.Account))
	}
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) VerifyPin(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return errors.Wrap(err, "unable to parse input")
	}

	res, err := a.accountService.VerifyPin(payload)
	if err != nil {
		return err
	}
	json.NewEncoder(w).Encode(res)
	return nil
}
