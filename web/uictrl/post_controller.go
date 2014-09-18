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
	"time"

	"../../client"
	"../../core/db"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

type GetAllPostsWrapper struct {
	Posts []PostResponse `json:"posts"`
}

type GetPostWrapper struct {
	Post PostResponse `json:"posts"`
}

type PostWrapper struct {
	Post CreatePostRequest `json:"post"`
}

type CreatePostRequest struct {
	Id               int64     `json:"id"`
	Message          string    `json:"message" sql:"type:text;not null"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	DeletedAt        time.Time `json:"deletedAt"`
	PostedAt         time.Time `json:"postedAt"`
	TTL              uint8     `json:"ttl"`
	OriginatorId     string    `json:"originator" sql:"not null"`
	AuthorId         string    `json:"author" sql:"not null"`
	ProfilePictureId int64     `json:"profilePictureId"`
	CircleIds        []string  `json:"circles" sql:"not null"`
	CommentIds       []string  `json:"comments" sql:"not null"`
}

type PostResponse struct {
	Id               int64     `json:"id"`
	Message          string    `json:"message"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	DeletedAt        time.Time `json:"deletedAt"`
	PostedAt         time.Time `json:"postedAt"`
	TTL              uint8     `json:"ttl"`
	OriginatorId     int64     `json:"originator"`
	AuthorId         int64     `json:"author"`
	ProfilePictureId int64     `json:"profilePictureId"`
	CircleIds        []int64   `json:"circles,omitempty"`
	CommentIds       []int64   `json:"comments,omitempty"`
}

func (api *Api) GetAllPosts(w rest.ResponseWriter, r *rest.Request) {
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

	posts := []db.Post{}

	if len(requestedIds) > 0 {
		err = api.Where("parent_id = 0").Where(requestedIds).Find(&posts).Error
	} else {
		err = api.Where("parent_id = 0").Find(&posts).Error
	}
	if err != nil {
		if err != gorm.RecordNotFound {
			log.Println(gormLoadError("posts"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	postResponses := []PostResponse{}

	for _, post := range posts {
		postResponse := PostResponse{
			Id:               post.Id,
			Message:          post.Message,
			CreatedAt:        post.CreatedAt,
			UpdatedAt:        post.UpdatedAt,
			DeletedAt:        post.DeletedAt,
			PostedAt:         post.PostedAt,
			TTL:              post.TTL,
			OriginatorId:     post.OriginatorId,
			AuthorId:         post.AuthorId,
			ProfilePictureId: api.GetProfilePictureId(post.AuthorId),
		}

		// Get Circles

		var circleIds []int64
		var id int64

		// todo translate to gorm
		rows, err := api.Raw("select circle_id from circle_posts where post_id = ?", post.Id).Rows() // (*sql.Rows, error)
		if err != nil {
			log.Println(gormLoadError("circle ids"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			rows.Scan(&id)
			circleIds = append(circleIds, id)
		}

		postResponse.CircleIds = circleIds

		// Get Comments

		var commentIds []int64
		var comments []db.Post

		if err = api.Where(db.Post{ParentId: post.Id}).Find(&comments).Error; err != nil {
			if err != gorm.RecordNotFound {
				log.Println(gormLoadError("comment ids"), err)
				rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
				return
			}
		}

		for _, comment := range comments {
			commentIds = append(commentIds, comment.Id)
		}

		postResponse.CommentIds = commentIds
		postResponses = append(postResponses, postResponse)

	}

	w.WriteJson(
		&GetAllPostsWrapper{
			Posts: postResponses,
		},
	)
}

func (api *Api) CreatePost(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)
	if err != nil {
		rest.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	postRequest := PostWrapper{}

	if err = r.DecodeJsonPayload(&postRequest); err != nil {
		log.Println(jsonDecodeError("create post request"), err)
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	post := postRequest.Post

	err, authorId := ToId(post.AuthorId)
	if err != nil {
		log.Println("Cannot convert author id to int")
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	err, originatorId := ToId(post.OriginatorId)
	if err != nil {
		log.Println("Cannot convert originator id to int")
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	err, circleIds := ToIds(post.CircleIds)
	if err != nil {
		log.Println("Cannot convert circle ids to int")
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	author := db.Onion{}
	if err = api.Find(&author, db.Onion{Id: authorId}).Error; err != nil {
		if err == gorm.RecordNotFound {
			rest.NotFound(w, r)
			return
		} else {
			log.Println(gormLoadError("onion"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	originator := db.Onion{}
	if err = api.Find(&originator, db.Onion{Id: originatorId}).Error; err != nil {
		if err == gorm.RecordNotFound {
			rest.NotFound(w, r)
			return
		} else {
			log.Println(gormLoadError("onion"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
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

	newPost := db.Post{}

	newPost.Message = post.Message
	newPost.TTL = post.TTL
	newPost.Originator = originator
	newPost.PostedAt = time.Now()
	newPost.PublishedAt = time.Now()
	newPost.Author = author
	newPost.Published = true

	newPost.CalcHash()

	if api.Create(&newPost).Error != nil {
		log.Println(gormSaveError("post"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	//TODO: Transactions?

	for _, circle := range circles {
		if err = api.Model(&circle).Association("Posts").Append(&newPost).Error; err != nil {
			log.Println(gormSaveError("circle"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
			return
		}
	}

	postRequest.Post.Id = newPost.Id
	postRequest.Post.CommentIds = []string{}
	postRequest.Post.ProfilePictureId = api.GetProfilePictureId(newPost.Author.Id)
	w.WriteJson(&postRequest)

	client.TriggerCircles(&api.SSNDB, circles)
}

func (api *Api) GetPost(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	requestedId, err := strconv.ParseInt(_id, 10, 64)

	post := db.Post{}

	if err = api.First(&post, requestedId).Error; err != nil {
		if err == gorm.RecordNotFound {
			rest.NotFound(w, r)
			return
		} else {
			log.Println(gormLoadError("posts"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	postResponse := PostResponse{
		Id:               post.Id,
		Message:          post.Message,
		CreatedAt:        post.CreatedAt,
		UpdatedAt:        post.UpdatedAt,
		DeletedAt:        post.DeletedAt,
		PostedAt:         post.PostedAt,
		TTL:              post.TTL,
		OriginatorId:     post.OriginatorId,
		AuthorId:         post.AuthorId,
		ProfilePictureId: api.GetProfilePictureId(post.AuthorId),
	}

	// Get Circles

	var circleIds []int64
	var id int64

	// todo translate to gorm
	rows, err := api.Raw("select circle_id from circle_posts where post_id = ?", post.Id).Rows() // (*sql.Rows, error)
	if err != nil {
		log.Println(gormLoadError("circle ids"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&id)
		circleIds = append(circleIds, id)
	}

	postResponse.CircleIds = circleIds

	// Get Comments

	var commentIds []int64
	var comments []db.Post

	if err = api.Where(db.Post{ParentId: post.Id}).Find(&comments).Error; err != nil {
		log.Println(gormLoadError("comment ids"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}
	for _, comment := range comments {
		commentIds = append(commentIds, comment.Id)
	}

	postResponse.CommentIds = commentIds

	w.WriteJson(
		&GetPostWrapper{
			Post: postResponse,
		},
	)
}

func (api *Api) DeletePost(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)
	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	post := db.Post{}

	if err = api.First(&post, id).Error; err != nil {
		if err == gorm.RecordNotFound {
			rest.NotFound(w, r)
			return
		} else {
			log.Println(gormLoadError("post"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
			return
		}
	}

	if err := api.Delete(&post).Error; err != nil {
		log.Println(gormDeleteError("post"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
