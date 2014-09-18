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
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

type GetAllCirclesWrapper struct {
	Circles []db.EmberCircleResponse `json:"circles"`
}

type GetCircleWrapper struct {
	Circle db.EmberCircleResponse `json:"circle"`
}

type CreateCircleWrapper struct {
	Circle db.EmberCircleRequest `json:"circle"`
}

func (api *Api) GetAllCircles(w rest.ResponseWriter, r *rest.Request) {
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

	circles := []db.Circle{}
	_circles := []db.EmberCircleResponse{}

	if len(requestedIds) > 0 {
		err = api.Where(requestedIds).Find(&circles).Error
	} else {
		err = api.Find(&circles).Error
	}
	if err != nil {
		if err != gorm.RecordNotFound {
			log.Println(gormLoadError("circles"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
			return
		}
	}

	for _, circle := range circles {
		eCircle := db.EmberCircleResponse{}

		eCircle.Id = circle.Id
		eCircle.Name = circle.Name
		eCircle.Creator = circle.Creator
		eCircle.Contacts = []int64{}
		eCircle.Posts = []int64{}

		circle_contacts := []db.Contact{}
		circle_posts := []db.Post{}

		if err = api.Model(&circle).Association("Contacts").Find(&circle_contacts).Error; err != nil {
			log.Println(gormLoadError("circle contacts"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
			return
		}
		if err = api.Model(&circle).Association("Posts").Find(&circle_posts).Error; err != nil {
			log.Println(gormLoadError("circle posts"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
			return
		}

		for _, circle_contact := range circle_contacts {
			eCircle.Contacts = append(eCircle.Contacts, circle_contact.Id)
		}

		for _, circle_post := range circle_posts {
			eCircle.Posts = append(eCircle.Posts, circle_post.Id)
		}

		_circles = append(_circles, eCircle)
	}

	w.WriteJson(
		&GetAllCirclesWrapper{
			Circles: _circles,
		},
	)
}

func (api *Api) GetCircle(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	circle := db.Circle{}

	if err = api.First(&circle, id).Error; err != nil {
		rest.NotFound(w, r)
		return
	}

	eCircle := db.EmberCircleResponse{}

	eCircle.Id = circle.Id
	eCircle.Name = circle.Name
	eCircle.Creator = circle.Creator
	eCircle.Contacts = []int64{}
	eCircle.Posts = []int64{}

	circle_contacts := []db.Contact{}
	circle_posts := []db.Post{}
	api.Model(&circle).Related(&circle_contacts, "Contacts")
	api.Model(&circle).Related(&circle_posts, "Posts")

	for _, circle_contact := range circle_contacts {
		eCircle.Contacts = append(eCircle.Contacts, circle_contact.Id)
	}

	for _, circle_post := range circle_posts {
		eCircle.Posts = append(eCircle.Posts, circle_post.Id)
	}

	w.WriteJson(
		&GetCircleWrapper{
			Circle: eCircle,
		},
	)
}

func (api *Api) CreateCircle(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)
	if err != nil {
		rest.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	postRequest := CreateCircleWrapper{}

	if err = r.DecodeJsonPayload(&postRequest); err != nil {
		log.Println(jsonDecodeError("create circle request"), err)
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	circle := db.Circle{
		Name:    postRequest.Circle.Name,
		Creator: db.CREATOR_USER,
	}

	if api.Create(&circle).Error != nil {
		log.Println(gormSaveError("circle"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	postRequest.Circle.Id = circle.Id
	w.WriteJson(&postRequest)

}

func (api *Api) DeleteCircle(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	circle := db.Circle{Id: id}

	if err = api.Find(&circle, circle).Error; err != nil {
		if err == gorm.RecordNotFound {
			rest.NotFound(w, r)
			return
		} else {
			log.Println(gormLoadError("circle"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	api.Model(&circle).Association("Contacts").Clear()
	api.Model(&circle).Association("Posts").Clear()
	api.Delete(&circle)

	w.WriteHeader(http.StatusOK)
}
