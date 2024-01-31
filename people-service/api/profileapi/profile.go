package profileapi

import (
	"encoding/json"
	"net/http"
	"people-service/app"
	"people-service/consts"
	"people-service/model"
	"people-service/util"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func (a *api) GetProfiles(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	userIDStr := ctx.Vars["userID"]
	userID, _ := strconv.Atoi(userIDStr)
	ret, err := a.App.ProfileService.GetProfilesByUserID(userID)

	if err == nil {
		json.NewEncoder(w).Encode(ret)
		return nil
	}
	return err
}

func (a *api) GetProfileCount(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	userIDStr := ctx.Vars["userID"]
	userID, _ := strconv.Atoi(userIDStr)
	ret, err := a.App.ProfileService.GetProfileCountByUserID(userID)

	if err == nil {
		json.NewEncoder(w).Encode(ret)
		return nil
	}
	return err
}

func (a *api) SetProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {

	//GRPC METHOD FOR FetchBoards
	//GRPC METHOD FOR GetBoardPermissionByProfile

	// boardsRes, err := a.boardService.FetchBoards(ctx.Profile, true, "", "")
	// boards := boardsRes["data"].([]*model.Board)

	// var boardPermissions *model.BoardPermission
	// if err != nil {
	// 	boardPermissions, _ = a.boardService.GetBoardPermissionByProfile(boards, ctx.Profile)
	// }

	// cache board permissions as per profile
	// cacheKey := fmt.Sprintf("boards:%s", strconv.Itoa(ctx.Profile))
	// a.cache.SetValue(cacheKey, boardPermissions.ToJSON())

	res := util.SetResponse(nil, 1, "All processes complete")
	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) AddProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	r.ParseMultipartForm(12 << 20)
	err := r.ParseForm()
	if err != nil {
		return errors.Wrap(err, "error parsing form")
	}
	profileData := r.FormValue("profile")
	var payload model.Profile
	err = json.Unmarshal([]byte(profileData), &payload)
	if err != nil {
		return errors.Wrap(err, "Error Parsing Profile metadata")
	}
	if payload.ConnectCodeExpiration == "" {
		payload.ConnectCodeExpiration = "1w"
	}
	res, err := a.App.ProfileService.AddProfile(payload, ctx.User.ID)
	if err != nil {
		if errors.Is(err, consts.ProfileLimitError) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, consts.ProfileLimitError.Error()))
			return nil
		}
		return err
	}
	profileIDInt := int(res["data"].(map[string]interface{})["id"].(int64))

	// default things board
	var defThingsBoard model.Board
	defThingsBoard.Title = "Default Board"
	defThingsBoard.CreateDate = time.Now()
	defThingsBoard.Type = "BOARD"
	defThingsBoard.Owner = strconv.Itoa(profileIDInt)
	defThingsBoard.IsDefaultBoard = true
	defThingsBoard.Description = "This is the default board. This is cannot be deleted"

	//GRPC METHOD FOR AddBoard
	//GRPC METHOD FOR UpdateDefaultThingsBoard

	// defThingsBoardRes, err := a.boardService.AddBoard(defThingsBoard, ctx.Profile)
	// if err != nil {
	// 	return errors.Wrap(err, "unable to add default Things Board")
	// }

	// profileRes, err := a.App.ProfileService.GetProfileInfo(profileIDInt)
	// if err != nil {
	// 	return errors.Wrap(err, "unable to fetch profile")
	// }

	// add default things board ID to profile
	// err = a.profileService.UpdateDefaultThingsBoard(profileRes["data"].(model.Profile).ID, defThingsBoardRes["data"].(model.Board).Id.Hex())
	// if err != nil {
	// 	return errors.Wrap(err, "unable to update default board ID")
	// }

	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

// EditProfile
func (a *api) EditProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	r.ParseMultipartForm(12 << 20)
	err := r.ParseForm()
	if err != nil {
		return errors.Wrap(err, "error parsing form")
	}
	profileData := r.FormValue("profile")
	var payload model.Profile
	err = json.Unmarshal([]byte(profileData), &payload)
	if err != nil {
		return err
	}
	payload.ID = ctx.Profile
	query := r.URL.Query()
	if query.Get("profileID") != "" {
		payload.ID, _ = strconv.Atoi(query.Get("profileID"))
	}
	_, err = a.App.ProfileService.EditProfile(payload)
	if err != nil {
		return errors.Wrap(err, "unable to update profile in db")
	}

	profileMapInfo, err := a.App.ProfileService.GetProfileInfo(payload.ID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch profile in db")
	}

	profileInfo := profileMapInfo["data"].(model.Profile)
	json.NewEncoder(w).Encode(util.SetResponse(profileInfo, 1, "Profiles updated successfully"))
	return nil
}

func (a *api) GetProfileInfo(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		// Profile not authorized to perform this action
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	query := r.URL.Query()
	if query.Get("profileID") != "" {
		id, _ = strconv.Atoi(query.Get("profileID"))
	}

	// Update any pending profile tags update

	// GRPC METHOD FOR GetContentProfileTags based on profileID based on [board / post / note /task / file / collections] tags
	err := a.App.ProfileService.UpdateProfileTagsNew(strconv.Itoa(id))
	if err != nil {
		return err
	}

	res, err := a.App.ProfileService.GetProfileInfo(id)
	if err != nil {
		return errors.Wrap(err, "unable to get profile info")
	}

	//GRPC METHOD FOR GetNotificationDisplayCount

	// totalNotifications, err := a.notificationService.GetNotificationDisplayCount(fmt.Sprint(ctx.Profile))
	// if err != nil {
	// 	return errors.Wrap(err, "unable to fetch notification count")
	// }

	// if profile, ok := res["data"].(model.Profile); ok {
	// 	profile.TotalNotifications = totalNotifications
	// 	res["data"] = profile
	// }

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) GetProfileTags(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		// Profile not authorized to perform this action
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	res, err := a.App.ProfileService.FetchTags(id)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}
	return err
}

// DeleteProfile
func (a *api) DeleteProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	callerUserID := ctx.Vars["userID"]

	var payload struct {
		ProfileIDs []string `json:"profileIDs"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	res, err := a.App.ProfileService.DeleteProfile(callerUserID, payload.ProfileIDs)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) GenerateCode(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	CallerprofileID := ctx.Profile
	if CallerprofileID == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	var payload model.ConnectionRequest
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode json body")
	}
	res, err := a.App.ProfileService.GenerateCode(ctx.User.ID, CallerprofileID, payload)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}
	return err
}

func (a *api) DeleteCode(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	var payload map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode json")
	}

	res, err := a.App.ProfileService.DeleteCode(ctx.User.ID, payload)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

// UpdateProfileSettings
func (a *api) UpdateProfileSettings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	var payload model.UpdateProfileSettings
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	res, err := a.App.ProfileService.UpdateProfileSettings(payload, ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) UpdateShareableSettings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var shareableSettings model.ShareableSettings
	err := json.NewDecoder(r.Body).Decode(&shareableSettings)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	response, err := a.App.ProfileService.UpdateShareableSettings(ctx.Profile, shareableSettings)
	if err == nil {
		json.NewEncoder(w).Encode(response)
	}

	return err
}

func (a *api) SendCoManagerRequest(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var connReq map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&connReq)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}

	res, err := a.App.ProfileService.SendCoManagerRequest(id, connReq)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) AcceptCoManagerRequest(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}

	res, err := a.App.ProfileService.AcceptCoManagerRequest(ctx.User.ID, id, payload["code"].(string))
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) ListProfilesWithCoManager(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	search := r.URL.Query().Get("search")
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	allProfilesRes, err := a.App.ProfileService.GetProfilesWithInfoByUserID(ctx.User.ID)
	if err != nil {
		return err
	}

	// fetch profiles along with their co-managers
	res, err := a.App.ProfileService.FetchProfilesWithCoManager(ctx.Profile, allProfilesRes["data"].([]model.Profile), search, page, limit)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) ListExternalProfiles(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	search := r.URL.Query().Get("search")
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	res, err := a.App.ProfileService.FetchExternalProfiles(ctx.User.ID, search, page, limit)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) GetPeopleInfo(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload map[string]interface{}
	var res map[string]interface{}
	var err error

	query := r.URL.Query()
	limit := query.Get("limit")
	page := query.Get("page")

	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}

	if len(payload) < 1 || len(payload) > 3 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Invalid body found in request"))
		return nil
	}

	if len(payload) > 1 {
		// only one of the paramter exists so setting other to empty
		if len(payload) == 2 {
			for key := range payload {
				if key == "search" {
					payload["boardID"] = ""
					break
				} else if key == "boardID" {
					payload["search"] = ""
					break
				}
			}
		}
	} else {
		// no parameters found in body set default value to empty
		payload["search"] = ""
		payload["boardID"] = ""
	}

	payload["search"] = strings.Replace(payload["search"].(string), "%20", " ", -1)
	res, err = a.App.ProfileService.GetPeopleInfo(ctx.Profile, int(payload["type"].(float64)), limit, page, payload["search"].(string), payload["boardID"].(string))
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}

	return err
}

func (a *api) FetchBoards(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.App.ProfileService.FetchBoards(id, ctx.Vars["type"])
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}
	return err
}

func (a *api) ListCoManagerCandidates(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	ret, err := a.App.ProfileService.ListAllOpenProfiles()
	if err == nil {
		json.NewEncoder(w).Encode(ret)
	}
	return err
}

func (a *api) MoveConnection(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	switch reflect.TypeOf(payload).Kind() {
	case reflect.Map:
		res, err := a.App.ProfileService.MoveConnection(payload, id)
		if err == nil {
			json.NewEncoder(w).Encode(res)
			return nil
		} else {
			return err
		}
	default:
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Invalid request body"))
		return nil
	}
}
