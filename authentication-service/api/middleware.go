package api

import (
	"authentication-service/api/common"
	"authentication-service/app"
	"authentication-service/model"
	"authentication-service/util"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func (a *API) handler(f common.HandlerFuncWithCTX, auth ...bool) http.HandlerFunc {
	checkAuth := auth[0]
	onlyUserAuth := false
	if len(auth) > 1 {
		onlyUserAuth = auth[1]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("API -", r.URL.Path)
		r.Body = http.MaxBytesReader(w, r.Body, a.Config.MaxContentSize*1024*1024)
		beginTime := time.Now()
		hijacker, _ := w.(http.Hijacker)
		ctx := a.App.NewContext().WithRemoteAddress(a.IPAddressForRequest(r))
		ctx = ctx.WithLogger(ctx.Logger.WithField("request_id", base64.RawURLEncoding.EncodeToString(util.NewID())))
		ctx.Vars = mux.Vars(r)

		w = &common.StatusCodeRecorder{
			ResponseWriter: w,
			Hijacker:       hijacker,
		}
		if checkAuth {
			authResp := checkForUserProfileAuth(a.Config, ctx, r)
			if authResp.Error != nil {
				ctx.Logger.WithError(authResp.Error).Error(authResp.ErrMsg)
				if authResp.ErrCode == http.StatusUnauthorized {
					http.Error(w, authResp.ErrMsg, authResp.ErrCode)
				} else {
					http.Error(w, "error from checkForUserAuth", http.StatusForbidden)
				}
				return
			}
			ctx = ctx.WithUserProfile(authResp.User, authResp.Profile)
		}

		if len(auth) > 1 && onlyUserAuth {
			authResp := checkForUserAuth(a.Config, ctx, r)
			if authResp.Error != nil {
				ctx.Logger.WithError(authResp.Error).Error(authResp.ErrMsg)
				if authResp.ErrCode == http.StatusUnauthorized {
					// http.Error(w, "Token has expired!", errCode)
					http.Error(w, authResp.ErrMsg, authResp.ErrCode)
				} else {
					http.Error(w, "error from checkForUserAuth", http.StatusForbidden)
				}
				return
			}
			ctx = ctx.WithUser(authResp.User)
		}

		defer func() {
			statusCode := w.(*common.StatusCodeRecorder).StatusCode
			if statusCode == 0 {
				statusCode = 200
			}
			duration := time.Since(beginTime)

			logger := ctx.Logger.WithFields(logrus.Fields{
				"duration":    duration,
				"status_code": statusCode,
				"remote":      ctx.RemoteAddress,
			})
			logger.Info(r.Method + " " + r.URL.RequestURI())
		}()

		defer func() {
			if localRecover := recover(); localRecover != nil {
				ctx.Logger.Error(fmt.Errorf("recovered from panic\n %v: %s", localRecover, debug.Stack()))
				json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "server failed to process request"))
			}
		}()

		w.Header().Set("Content-Type", "application/json")

		if err := f(ctx, w, r); err != nil {
			if verr, ok := err.(*app.ValidationError); ok {
				data, err := json.Marshal(verr)
				if err == nil {
					w.WriteHeader(http.StatusBadRequest)
					_, err = w.Write(data)
				}

				if err != nil {
					ctx.Logger.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
					// http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			} else if uerr, ok := err.(*app.UserError); ok {
				data, err := json.Marshal(uerr)
				if err == nil {
					w.WriteHeader(uerr.StatusCode)
					_, err = w.Write(data)
				}

				if err != nil {
					ctx.Logger.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
					// http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			} else {
				ctx.Logger.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
				// http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}
	}
}

func checkForUserProfileAuth(config *common.Config, ctx *app.Context, r *http.Request) model.AuthResponse {
	token := r.Header.Get(config.AuthCookieName)
	if token == "" {
		c, err := r.Cookie(config.AuthCookieName)
		if c.Value == "" || err != nil {
			return model.AuthResponse{
				User: nil, Profile: 0, ErrCode: http.StatusUnauthorized,
				ErrMsg: "Token is not present", Error: errors.New("Token is not present"),
			}
		}
		token = c.Value
	}
	profile := r.Header.Get("Profile")
	var profileID int
	if profile == "" {
		c, err := r.Cookie("Profile")
		if err != nil {
			return model.AuthResponse{
				User: nil, Profile: 0, ErrCode: http.StatusUnauthorized,
				ErrMsg: "Profile is not present", Error: err,
			}
		}
		profile = c.Value
	}

	profileID, _ = strconv.Atoi(profile)
	// cachedUser, err := ctx.UserService.ValidateJWTToken(token)
	// if err != nil {
	// 	if err == jwt.ErrSignatureInvalid {
	// 		return model.AuthResponse{
	// 			User: nil, Profile: 0, ErrCode: http.StatusUnauthorized,
	// 			ErrMsg: "Invalid JWT token", Error: ctx.AuthorizationError(true),
	// 		}
	// 	}
	// 	return model.AuthResponse{
	// 		User: nil, Profile: 0, ErrCode: http.StatusUnauthorized,
	// 		ErrMsg: "Error in ValidateJWTToken", Error: err,
	// 	}
	// }
	// err = ctx.ProfileService.ValidateProfile(profileID, cachedUser.ID)
	// if err != nil {
	// 	if err.Error() == "Profile is not present" {
	// 		return model.AuthResponse{
	// 			User: nil, Profile: -1, ErrCode: http.StatusUnauthorized,
	// 			ErrMsg: "Profile is not present", Error: err,
	// 		}
	// 	}
	// 	return model.AuthResponse{
	// 		User: nil, Profile: 0, ErrCode: http.StatusUnauthorized,
	// 		ErrMsg: "Invalid Profile", Error: err,
	// 	}

	// }
	// return model.AuthResponse{User: cachedUser, Profile: profileID, ErrCode: 0, ErrMsg: "", Error: nil}
	return model.AuthResponse{Profile: profileID, ErrCode: 0, ErrMsg: "", Error: nil}
}

func checkForUserAuth(config *common.Config, ctx *app.Context, r *http.Request) model.AuthResponse {
	token := r.Header.Get(config.AuthCookieName)
	if token == "" {
		c, err := r.Cookie(config.AuthCookieName)
		if err != nil || c.Value == "" {
			return model.AuthResponse{
				User: nil, ErrMsg: "Token is not present",
				ErrCode: http.StatusUnauthorized, Error: errors.New("Token is not present"),
			}
		}
		token = c.Value
	}

	// cachedUser, err := ctx.UserService.ValidateJWTToken(token)
	// if err != nil {
	// 	fmt.Println("error from Validate_JWT_Token", err)
	// 	if err == jwt.ErrSignatureInvalid {
	// 		return model.AuthResponse{
	// 			User: nil, ErrMsg: "Invalid JWT token",
	// 			ErrCode: http.StatusUnauthorized, Error: ctx.AuthorizationError(true),
	// 		}
	// 	}
	// 	return model.AuthResponse{User: nil, ErrMsg: "unable to validate JWT token", ErrCode: 0, Error: err}
	// }

	// return model.AuthResponse{User: cachedUser, ErrMsg: "", ErrCode: 0, Error: nil}
	return model.AuthResponse{ErrMsg: "", ErrCode: 0, Error: nil}
}

func checkForPreLoginAuth(config *common.Config, ctx *app.Context, r *http.Request) (*model.Account, error) {
	token := r.Header.Get(config.SignUpAuthName)
	if token == "" {
		c, err := r.Cookie(config.SignUpAuthName)
		if err != nil {
			return nil, err
		}
		token = c.Value
	}

	// cachedUser, err := ctx.UserService.ValidateJWTToken(token)
	// if err != nil {
	// 	if err == jwt.ErrSignatureInvalid {
	// 		return nil, ctx.AuthorizationError(true)
	// 	}
	// 	return nil, err
	// }
	// return cachedUser, nil

	return nil, nil
}

// IPAddressForRequest determines IP address for request
func (a *API) IPAddressForRequest(r *http.Request) string {
	addr := r.RemoteAddr
	if a.Config.ProxyCount > 0 {
		h := r.Header.Get("X-Forwarded-For")
		if h != "" {
			clients := strings.Split(h, ",")
			if a.Config.ProxyCount > len(clients) {
				addr = clients[0]
			} else {
				addr = clients[len(clients)-a.Config.ProxyCount]
			}
		}
	}
	return strings.Split(strings.TrimSpace(addr), ":")[0]
}
