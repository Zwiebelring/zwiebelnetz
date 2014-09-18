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

type GetAllCommentsWrapper struct {
	Comments []CommentResponse `json:"comments"`
}

type GetCommentWrapper struct {
	Comment CommentResponse `json:"comment"`
}

type CommentWrapper struct {
	Comment CreateCommentRequest `json:"comment"`
}

type CreateCommentRequest struct {
	Id               int64     `json:"id"`
	Message          string    `json:"message" sql:"type:text;not null"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	DeletedAt        time.Time `json:"deletedAt"`
	PostedAt         time.Time `json:"postedAt"`
	TTL              uint8     `json:"ttl"`
	OriginatorId     string    `json:"originator" sql:"not null"`
	AuthorId         string    `json:"author" sql:"not null"`
	ParentId         string    `json:"post" sql:"not null"`
	ProfilePictureId int64     `json:"profilePictureId"`
}

type CommentResponse struct {
	Id                int64     `json:"id"`
	Message           string    `json:"message"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	DeletedAt         time.Time `json:"deletedAt"`
	PostedAt          time.Time `json:"postedAt"`
	IsRemotePublished bool      `json:"isRemotePublished"`
	TTL               uint8     `json:"ttl"`
	OriginatorId      int64     `json:"originator"`
	AuthorId          int64     `json:"author"`
	ParentId          int64     `json:"post"`
	ProfilePictureId  int64     `json:"profilePictureId"`
}

func (api *Api) GetAllComments(w rest.ResponseWriter, r *rest.Request) {
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
		err = api.Where("parent_id != 0").Where(requestedIds).Find(&posts).Error
	} else {
		err = api.Where("parent_id != 0").Find(&posts).Error
	}
	if err != nil {
		if err != gorm.RecordNotFound {
			log.Println(gormLoadError("posts"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	commentResponses := []CommentResponse{}

	for _, post := range posts {
		commentResponse := CommentResponse{
			Id:                post.Id,
			Message:           post.Message,
			CreatedAt:         post.CreatedAt,
			UpdatedAt:         post.UpdatedAt,
			DeletedAt:         post.DeletedAt,
			PostedAt:          post.PostedAt,
			IsRemotePublished: post.RemotePublishedAt.Unix() > 0,
			TTL:               post.TTL,
			OriginatorId:      post.OriginatorId,
			AuthorId:          post.AuthorId,
			ParentId:          post.ParentId,
			ProfilePictureId:  api.GetProfilePictureId(post.AuthorId),
		}

		commentResponses = append(commentResponses, commentResponse)

	}

	w.WriteJson(
		&GetAllCommentsWrapper{
			Comments: commentResponses,
		},
	)
}

func (api *Api) CreateComment(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)
	if err != nil {
		rest.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	commentRequest := CommentWrapper{}

	if err = r.DecodeJsonPayload(&commentRequest); err != nil {
		log.Println(jsonDecodeError("create post request"), err)
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	comment := commentRequest.Comment

	var parentId int64
	err, parentId = ToId(comment.ParentId)
	if err != nil {
		log.Println("Cannot convert parent id to int")
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	author := api.GetSelfOnion()

	newComment := db.Post{}

	newComment.Message = comment.Message
	newComment.TTL = comment.TTL
	newComment.PublishedAt = time.Now()
	newComment.PostedAt = time.Now()
	newComment.Author = author
	newComment.AuthorId = author.Id
	newComment.Published = true
	newComment.ParentId = parentId

	var parentPost db.Post
	if err = api.First(&parentPost, parentId).Error; err != nil {
		if err == gorm.RecordNotFound {
			log.Println("parent post for comment not found")
			rest.NotFound(w, r)
			return
		} else {
			log.Println(gormLoadError("parent post"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	// derive originator from parent post orignator

	newComment.OriginatorId = parentPost.OriginatorId
	if err = api.First(&newComment.Originator,
		newComment.OriginatorId).Error; err != nil {
		if err == gorm.RecordNotFound {
			log.Println("onion for parent post not found")
			rest.NotFound(w, r)
			return
		} else {
			log.Println(gormLoadError("parent post onion"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	newComment.CalcHash()

	if api.Create(&newComment).Error != nil {
		log.Println(gormSaveError("comment"), err)
		rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		return
	}

	// enter comment into circles if we commented our own post
	api.RedirectComment(&newComment)

	// trigger all contacts to whom it may concern
	circles := api.GetPostCircles(&newComment)
	client.TriggerCircles(&api.SSNDB, circles)

	var onionsToTrigger []db.Onion
	if newComment.OriginatorId != newComment.AuthorId {
		// not our post (originator != us)
		onionsToTrigger = append(onionsToTrigger, newComment.Originator)
	} else {
		// our post, set remote published (since we are the "remote")
		newComment.RemotePublishedAt = time.Now()
		api.Save(&newComment)
	}

	log.Printf("I am going to trigger the following onions in a goroutine now: ")
	log.Println(onionsToTrigger)
	go client.TriggerHandling(&api.SSNDB, onionsToTrigger)

	// todo send comment response
	resp := CommentResponse{
		Id:                newComment.Id,
		Message:           newComment.Message,
		CreatedAt:         newComment.CreatedAt,
		UpdatedAt:         newComment.UpdatedAt,
		DeletedAt:         newComment.DeletedAt,
		PostedAt:          newComment.PostedAt,
		IsRemotePublished: newComment.RemotePublishedAt.Unix() > 0,
		TTL:               newComment.TTL,
		OriginatorId:      newComment.OriginatorId,
		AuthorId:          newComment.AuthorId,
		ParentId:          newComment.ParentId,
		ProfilePictureId:  api.GetProfilePictureId(newComment.AuthorId),
	}
	w.WriteJson(GetCommentWrapper{resp})
}

func (api *Api) GetComment(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	requestedId, err := strconv.ParseInt(_id, 10, 64)

	comment := db.Post{}

	if err = api.First(&comment, requestedId).Error; err != nil {
		if err == gorm.RecordNotFound {
			rest.NotFound(w, r)
			return
		} else {
			log.Println(gormLoadError("comment"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	commentResponse := CommentResponse{
		Id:                comment.Id,
		Message:           comment.Message,
		CreatedAt:         comment.CreatedAt,
		UpdatedAt:         comment.UpdatedAt,
		DeletedAt:         comment.DeletedAt,
		PostedAt:          comment.PostedAt,
		IsRemotePublished: comment.RemotePublishedAt.Unix() > 0,
		TTL:               comment.TTL,
		OriginatorId:      comment.OriginatorId,
		AuthorId:          comment.AuthorId,
		ParentId:          comment.ParentId,
		ProfilePictureId:  api.GetProfilePictureId(comment.AuthorId),
	}

	w.WriteJson(
		&GetCommentWrapper{
			Comment: commentResponse,
		},
	)
}

func (api *Api) DeleteComment(w rest.ResponseWriter, r *rest.Request) {
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
