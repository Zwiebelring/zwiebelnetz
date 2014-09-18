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
	"log"
	"net/http"
	"strconv"

	"../../core/db"
	"github.com/ant0ine/go-json-rest/rest"
)

type GetAllOnionsWrapper struct {
	Onions []db.EmberOnion `json:"onions"`
}

type GetAllAuthorsWrapper struct {
	Authors []db.EmberOnion `json:"authors"`
}

type GetAllOriginatorsWrapper struct {
	Originators []db.EmberOnion `json:"originators"`
}

type GetOnionWrapper struct {
	Onion db.EmberOnion `json:"onion"`
}
type PostOnionWrapper struct {
	Onion db.Onion `json:"onion"`
}

type GetAuthorWrapper struct {
	Onion db.EmberOnion `json:"author"`
}

type GetOriginatorWrapper struct {
	Onion db.EmberOnion `json:"originator"`
}

func (api *Api) fetchAllOnions(filter []int64) (emberOnions []db.EmberOnion, err error) {

	emberOnions = []db.EmberOnion{}

	onions := []db.Onion{}
	if api.Find(&onions).Error != nil {
		return emberOnions, err
	}

	responses := []db.EmberOnion{}
	var emberOnion db.EmberOnion

	for _, onion := range onions {
		rows, err := api.Raw("select id from contacts where onion_id = ?", onion.Id).Rows()
		if err != nil {
			return emberOnions, err
		}

		defer rows.Close()

		var id int64

		// TODO: make sure len(rows) == 1

		rows.Next()
		rows.Scan(&id)

		emberOnion = db.EmberOnion{
			Id:        onion.Id,
			Onion:     onion.Onion,
			ContactId: id,
		}
		if len(filter) > 0 {
			for _, requestedId := range filter {
				if requestedId == onion.Id {
					responses = append(responses, emberOnion)
					break
				}
			}
		} else {
			responses = append(responses, emberOnion)
		}
	}

	return responses, nil
}

func (api *Api) fetchContact(id int64) (contactId int64, err error) {
	rows, err := api.Raw("select id from contacts where onion_id = ?", id).Rows()
	if err != nil {
		return 0, err
	}

	defer rows.Close()

	//TODO: same
	rows.Next()
	rows.Scan(&contactId)
	return
}

func (api *Api) GetAllOnions(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	requestedIds, err := GetRequestedIds(r)
	if err != nil {
		log.Println("Parsing requested ids failed", err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	resp, err := api.fetchAllOnions(requestedIds)
	if err != nil {
		log.Println(gormLoadError("onions"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	w.WriteJson(
		&GetAllOnionsWrapper{
			Onions: resp,
		},
	)
}

func (api *Api) GetAllAuthors(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	requestedIds, err := GetRequestedIds(r)
	if err != nil {
		log.Println("Parsing requested ids failed", err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	resp, err := api.fetchAllOnions(requestedIds)
	if err != nil {
		log.Println(gormLoadError("onions"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	w.WriteJson(
		&GetAllAuthorsWrapper{
			Authors: resp,
		},
	)
}

func (api *Api) GetAllOriginators(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	requestedIds, err := GetRequestedIds(r)
	if err != nil {
		log.Println("Parsing requested ids failed", err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	resp, err := api.fetchAllOnions(requestedIds)
	if err != nil {
		log.Println(gormLoadError("onions"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	w.WriteJson(
		&GetAllOriginatorsWrapper{
			Originators: resp,
		},
	)
}

func (api *Api) GetOnion(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	onion := db.Onion{}
	if api.First(&onion, id).Error != nil {
		rest.NotFound(w, r)
		return
	}

	contactId, err := api.fetchContact(onion.Id)
	if err != nil {
	}

	// get profiles
	var profileIds []int64
	var profiles []db.Profile

	api.Where(db.Profile{OnionId: onion.Id}).Find(&profiles)
	for _, profile := range profiles {
		profileIds = append(profileIds, profile.Id)
	}

	w.WriteJson(
		&GetOnionWrapper{
			Onion: db.EmberOnion{
				Id:        onion.Id,
				Onion:     onion.Onion,
				ContactId: contactId,
				ProfileIds: profileIds,
			},
		},
	)
}

func (api *Api) GetAuthor(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	onion := db.Onion{}
	if api.First(&onion, id).Error != nil {
		rest.NotFound(w, r)
		return
	}

	contactId, err := api.fetchContact(onion.Id)
	if err != nil {
	}

	w.WriteJson(
		&GetAuthorWrapper{
			Onion: db.EmberOnion{
				Id:        onion.Id,
				Onion:     onion.Onion,
				ContactId: contactId,
			},
		},
	)
}

func (api *Api) CreateOnion(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	onionWrapper := PostOnionWrapper{}
	if err = r.DecodeJsonPayload(&onionWrapper); err != nil {
		log.Println(jsonDecodeError("create onion request"), err)
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	if !db.IsValidOnion(onionWrapper.Onion.Onion) {
		rest.Error(w, INVALIDONION, http.StatusBadRequest)
		return
	}

	onion := api.SSNDB.GetOnion(onionWrapper.Onion.Onion)
	if onion.Id != 0 {
		onionWrapper.Onion.Id = onion.Id
		w.WriteJson(onionWrapper)
		return
	} else {
		if err := api.Save(&onionWrapper.Onion).Error; err != nil {
			rest.Error(w, "Could not create onion in db", http.StatusBadRequest)
			return
		}
		w.WriteJson(onionWrapper)
		return
	}
}

func (api *Api) GetOriginator(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	onion := db.Onion{}
	if api.First(&onion, id).Error != nil {
		rest.NotFound(w, r)
		return
	}

	contactId, err := api.fetchContact(onion.Id)
	if err != nil {
	}

	w.WriteJson(
		&GetOriginatorWrapper{
			Onion: db.EmberOnion{
				Id:        onion.Id,
				Onion:     onion.Onion,
				ContactId: contactId,
			},
		},
	)
}
