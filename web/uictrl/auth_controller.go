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
	"../../core/db"
	"errors"
	"github.com/ant0ine/go-json-rest/rest"
	_ "log"
	"net/http"
)

type AuthTokenResponse struct {
	AuthToken string `json:"auth_token"`
}

type AuthorizeRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (api *Api) validateAuthHeader(r *http.Request) (user db.User, err error) {
	username := r.Header.Get("Auth-User")
	token := r.Header.Get("Auth-Token")

	if len(username) == 0 || len(token) == 0 {
		return db.User{}, errors.New("Auth Header missing")
	}

	user = db.User{}

	if err = api.Find(&user, db.User{Username: username}).Error; err != nil {
		return db.User{}, errors.New("User not found")
	}

	if token != user.AuthToken {
		return db.User{}, errors.New("Auth token invalid")
	}

	return user, nil
}

func (api *Api) Authorize(w rest.ResponseWriter, r *rest.Request) {
	request := AuthorizeRequest{}

	if err := r.DecodeJsonPayload(&request); err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(request.Username) == 0 || len(request.Password) == 0 {
		rest.Error(w, "Username or Password empty!", http.StatusBadRequest)
		return
	}

	user := db.User{}
	if api.Find(&user, db.User{Username: request.Username}).Error != nil {
		rest.Error(w, "User does not exist or Password wrong", http.StatusUnauthorized)
		return
	}

	err := user.CheckPassword(request.Password)
	if err != nil {
		rest.Error(w, "User does not exist or Password wrong", http.StatusUnauthorized)
		return
	}

	err = user.GenerateAuthToken()
	if err != nil {
		rest.Error(w, "Auth token generation failed", http.StatusInternalServerError)
		return
	}

	if api.Save(&user).Error != nil {
		rest.Error(w, "Updating User failed", http.StatusInternalServerError)
		return
	}

	w.WriteJson(
		&AuthTokenResponse{
			AuthToken: user.AuthToken,
		},
	)
}
