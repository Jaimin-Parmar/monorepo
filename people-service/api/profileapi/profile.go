package profileapi

import (
	"encoding/json"
	"net/http"
	"people-service/app"
	"people-service/consts"
	"people-service/model"
	"people-service/util"
	"strconv"
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
