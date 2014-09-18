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
	"github.com/ant0ine/go-json-rest/rest"
	"net/http"
	"time"
)

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (api *Api) CreateUser(w rest.ResponseWriter, r *rest.Request) {

	request := CreateUserRequest{}

	if err := r.DecodeJsonPayload(&request); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(request.Username) == 0 || len(request.Password) == 0 {
		rest.Error(w, "Username or Password empty!", http.StatusBadRequest)
		return
	}

	user := db.User{
		Username:  request.Username,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := user.SetPassword(request.Password)
	if err != nil {
		rest.Error(w, "Setting password failed", http.StatusInternalServerError)
		return
	}

	if api.Create(&user).Error != nil {
		rest.Error(w, "Inserting user into db failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
