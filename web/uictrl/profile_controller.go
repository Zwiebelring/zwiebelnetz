/*
Copyright (c) 2014
  Dario Brandes
  Thies Johannsen
  Paul Kröger
  Sergej Mann
  Roman Naumann
  Sebastian Thobe
All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
/* -*- Mode: Go; indent-tabs-mode: t; c-basic-offset: 4; tab-width: 4 -*- */

package uictrl

import (
	"../../client"
	"../../core/db"
	"encoding/base64"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type ProfileResponse struct {
	Id        int64   `json:"id"`
	Key       string  `json:"key"`
	Value     string  `json:"value"`
	OnionId   int64   `json:"onion"`
	CircleIds []int64 `json:"circles"`
}

type CreateProfileRequest struct {
	Id        int64    `json:"id"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	CircleIds []string `json:"circles"`
}

type GetProfilesWrapper struct {
	Profiles []ProfileResponse `json:"profile"`
}

type ProfileWrapper struct {
	Profile CreateProfileRequest
}

type ProfileResponseWrapper struct {
	Profile ProfileResponse
}

func (api *Api) GetProfiles(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	requestedIds, err := GetRequestedIds(r)
	if err != nil {
		log.Println("Parsing requested ids failed", err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	profiles := []db.Profile{}

	if len(requestedIds) > 0 {
		err = api.Where(requestedIds).Find(&profiles).Error
	} else {
		err = api.Find(&profiles).Error
	}
	if err != nil {
		if err != gorm.RecordNotFound {
			log.Println(gormLoadError("profiles"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	profileResponses := make([]ProfileResponse, len(profiles))
	for i, profile := range profiles {
		profileResponse := ProfileResponse{
			Id:      profile.Id,
			Key:     profile.Key,
			Value:   profile.Value,
			OnionId: profile.OnionId,
		}

		var circles []db.Circle
		api.Model(profile).Related(&circles, "Circles")

		profileResponse.CircleIds = make([]int64, len(circles))
		for j, circle := range circles {
			profileResponse.CircleIds[j] = circle.Id
		}

		profileResponses[i] = profileResponse
	}

	w.WriteJson(
		&GetProfilesWrapper{
			Profiles: profileResponses,
		},
	)
}

func (api *Api) GetProfile(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	var profile db.Profile
	err = api.Find(&profile, id).Error

	if err != nil {
		if err != gorm.RecordNotFound {
			log.Println(gormLoadError("profile"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	profileResponse := ProfileResponse{
		Id:      profile.Id,
		Key:     profile.Key,
		Value:   profile.Value,
		OnionId: profile.OnionId,
	}

	var circles []db.Circle
	api.Model(profile).Related(&circles, "Circles")

	profileResponse.CircleIds = make([]int64, len(circles))
	for j, circle := range circles {
		profileResponse.CircleIds[j] = circle.Id
	}

	w.WriteJson(
		&ProfileResponseWrapper{
			Profile: profileResponse,
		},
	)
}

func (api *Api) DeleteProfile(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)
	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	profile := db.Profile{}

	if err = api.Find(&profile, db.Profile{Id: id}).Error; err != nil {
		rest.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	self := api.GetSelfOnion()
	var circles []db.Circle
	api.Model(&profile).Related(&circles, "Circles")

	if err := api.Unscoped().Delete(&profile).Error; err != nil {
		rest.Error(w, "Deleting profile failed", http.StatusInternalServerError)
		return
	}

	if profile.OnionId == self.Id {
		api.ProfileDeleted(circles)
	}

	w.WriteHeader(http.StatusOK)
}

func (api *Api) CreateProfile(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)
	if err != nil {
		rest.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	profileRequest := ProfileWrapper{}

	if err = r.DecodeJsonPayload(&profileRequest); err != nil {
		log.Println(jsonDecodeError("create profile request"), err)
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	profile := profileRequest.Profile

	err, circleIds := ToIds(profile.CircleIds)
	if err != nil {
		log.Println("Cannot convert circle ids to int")
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	var circles []db.Circle
	for _, id := range circleIds {
		circle := db.Circle{}
		if err = api.Find(&circle, db.Circle{Id: id}).Error; err != nil {
			if err == gorm.RecordNotFound {
				rest.NotFound(w, r)
				return
			} else {
				log.Println(gormLoadError("circle"), err)
				rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
			}
		}
		circles = append(circles, circle)
	}

	newProfile := db.Profile{}

	// if a profile with this key already exits update it
	api.Where(db.Profile{OnionId: api.GetSelfOnion().Id, Key: profile.Key}).First(&newProfile)

	newProfile.Key = profile.Key
	newProfile.Value = profile.Value
	newProfile.Circles = circles
	newProfile.Onion = api.GetSelfOnion()
	newProfile.OnionId = newProfile.Onion.Id
	newProfile.ChangedAt = time.Now()

	if api.Save(&newProfile).Error != nil {
		log.Println(gormSaveError("profile"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	for _, circle := range circles {
		if err = api.Model(&circle).Association("Profiles").Append(&newProfile).Error; err != nil {
			log.Println(gormSaveError("circle"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
			return
		}
	}

	response := ProfileResponseWrapper{
		Profile: ProfileResponse{
			Id:        newProfile.Id,
			Key:       newProfile.Key,
			Value:     newProfile.Value,
			OnionId:   newProfile.OnionId,
			CircleIds: circleIds,
		},
	}
	w.WriteJson(&response)
	client.TriggerCircles(&api.SSNDB, circles)
}

func (api *Api) PutProfile(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	if err != nil {
		rest.NotFound(w, r)
		return
	}

	var pw ProfileWrapper
	if err = r.DecodeJsonPayload(&pw); err != nil {
		log.Println(jsonDecodeError("profile update request"), err)
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	// check for double (or more) circle IDs and error out if any are found
	for i, c1 := range pw.Profile.CircleIds {
		for j, c2 := range pw.Profile.CircleIds {
			if i != j && c1 == c2 {
				// found duplicate
				log.Println("pofile-to-circle request with duplicate circle")
				rest.Error(w, DUPLICATECIRC, http.StatusBadRequest)
				return
			}
		}
	}

	// marshal back into Profile struct
	var profile db.Profile
	profile.Id = id
	err = api.Find(&profile, profile.Id).Error
	if err != nil {
		log.Println("Could not find profile")
		rest.NotFound(w, r)
		return
	}

	err = api.Model(&profile).Related(&profile.Onion, "Onion").Error
	if err != nil {
		log.Println("Could not find onion for profile")
		rest.NotFound(w, r)
		return
	}
	// overwrite updated fields
	profile.Key = pw.Profile.Key
	profile.Value = pw.Profile.Value
	profile.ChangedAt = time.Now()
	circles := make([]db.Circle, len(pw.Profile.CircleIds))

	api.Model(&profile).Association("Circles").Clear()
	// fill in circles
	for idx, circIdStr := range pw.Profile.CircleIds {
		cid, err := strconv.ParseInt(circIdStr, 10, 64)
		if err != nil {
			log.Println(jsonDecodeError("a circle-id was not a number!"), err)
			rest.Error(w, INVALIDJSON, http.StatusBadRequest)
			return
		}
		err = api.First(&circles[idx], cid).Error
		if err != nil {
			rest.Error(w, INVALIDCIRCLE, http.StatusBadRequest)
			return
		}
		api.Model(&circles[idx]).Association("Profiles").Append(&profile)
	}
	profile.Circles = circles
	api.Save(&profile)
	client.TriggerCircles(&api.SSNDB, profile.Circles)
}

func (api *Api) ProfilePictureHandler(w http.ResponseWriter, r *http.Request) {
	_, err := api.validateAuthHeader(r)
	if err != nil {
		http.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "POST":
		reader, err := r.MultipartReader()

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if part.FormName() != "file" {
				part.Close()
				continue
			}

			self := api.GetSelfOnion()
			var profile db.Profile
			api.Where(&db.Profile{Key: "picture", OnionId: self.Id}).First(&profile)
			picture, err := ioutil.ReadAll(part)
			if err != nil {
				log.Println("Failed to read profile image: ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			log.Printf("Image size: %d bytes\n", len(picture))
			profile.Key = "picture"
			profile.Value = base64.StdEncoding.EncodeToString(picture)
			profile.Onion = self
			profile.OnionId = self.Id
			profile.ChangedAt = time.Now()
			api.Save(&profile)

			var pubcirc db.Circle
			api.Where(&db.Circle{Name: "Public"}).First(&pubcirc)
			api.Model(&pubcirc).Association("Profiles").Append(&profile)

			api.Save(&profile)
			log.Printf("Picture saved in db")
			part.Close()

			client.TriggerCircles(&api.SSNDB, []db.Circle{pubcirc})
		}
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *Api) DeleteProfilePicture(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)
	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	self := api.GetSelfOnion()
	id := api.GetProfilePictureId(self.Id)
	if id == 0 {
		rest.Error(w, "Profile picture not found", http.StatusNotFound)
		return
	}
	var profilePicture db.Profile
	err = api.First(&profilePicture, id).Error
	if err != nil {
		rest.Error(w, "Profile picture not found", http.StatusNotFound)
		return
	}

	var circles []db.Circle
	api.Model(&profilePicture).Related(&circles, "Circles")

	api.Unscoped().Delete(&profilePicture)
	api.ProfileDeleted(circles)

	w.WriteHeader(http.StatusOK)
}

func (api *Api) ProfileDeleted(circles []db.Circle) {
	self := api.GetSelfOnion()
	var dummy db.Profile
	api.Where(&db.Profile{OnionId: self.Id, Key: ""}).First(&dummy)

	var pubcirc db.Circle
	api.Where(&db.Circle{Name: "Public"}).First(&pubcirc)

	dummy.Value = "Deleted"
	dummy.Key = ""
	dummy.OnionId = self.Id
	dummy.Onion = self
	dummy.ChangedAt = time.Now()

	api.Save(&dummy)
	if err := api.Model(&pubcirc).Association("Profiles").Append(&dummy).Error; err != nil {
		log.Println(gormSaveError("dummy profile"), err)
	}

	client.TriggerCircles(&api.SSNDB, circles)
}
