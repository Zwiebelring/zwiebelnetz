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
	"fmt"
	"log"
	"net/http"
)

func (api *Api) TriggerHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Trigger request received\n")
	_, err := api.validateAuthHeader(r)
	if err != nil {
		http.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}
	var contact db.Contact
	err, contactId := ToId(r.FormValue("contactId"))
	if err != nil {
		http.Error(w, INVALIDCONTACT, http.StatusBadRequest)
		return
	}

	api.Find(&contact, contactId)
	if contact.Id == 0 {
		http.Error(w, INVALIDCONTACT, http.StatusBadRequest)
		return
	}

	var onion db.Onion
	api.Find(&onion, contact.OnionId)

	client.TriggerHandling(&api.SSNDB, []db.Onion{onion})
	w.WriteHeader(http.StatusOK)
}

func (api *Api) SyncHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Sync request received\n")
	_, err := api.validateAuthHeader(r)
	if err != nil {
		http.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}
	client.SyncAllContacts(api.GetKey())
	w.WriteHeader(http.StatusOK)
}

func (api *Api) PendingPostsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := api.validateAuthHeader(r)
	if err != nil {
		http.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	w.Header().Add("pending", fmt.Sprint(api.PostsPending()))
	w.Header().Write(w)
}

func (api *Api) PendingContactsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := api.validateAuthHeader(r)
	if err != nil {
		http.Error(w, INVALIDAUTH, http.StatusUnauthorized)
		return
	}

	w.Header().Add("pending", fmt.Sprint(api.ContactsPending()))
	w.Header().Write(w)
}
