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

	"../../client"
	"../../core/db"
	_ "fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

type GetAllContactsWrapper struct {
	Contacts []db.EmberContactResponse `json:"contacts"`
}

type GetContactWrapper struct {
	Contact db.EmberContactResponse `json:"contact"`
}

type PutContactWrapper struct {
	Contact db.EmberContactRequest `json:"contact"`
}

type PostContactWrapper struct {
	Contact db.EmberContactRequest `json:"contact"`
}

func (api *Api) CreateContact(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	var pc PostContactWrapper
	if err = r.DecodeJsonPayload(&pc); err != nil {
		log.Println(jsonDecodeError("contact post request"), err)
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	contact := db.Contact{}
	contact.OnionId, _ = strconv.ParseInt(pc.Contact.OnionId, 10, 64)
	if err = api.First(&contact.Onion, contact.OnionId).Error; err != nil {
		rest.Error(w, INVALIDONION, http.StatusBadRequest)
		return
	}
	contact.Alias = pc.Contact.Alias
	contact.Trust = pc.Contact.Trust
	contact.Status = pc.Contact.Status
	contact.RequestMessage = pc.Contact.RequestMessage

	if err = api.Save(&contact).Error; err != nil {
		rest.Error(w, "Could not save to database", http.StatusInternalServerError)
		return
	}

	reply := GetContactWrapper{
		db.EmberContactResponse{
			contact,
			api.GetProfilePictureId(contact.OnionId),
			[]int64{},
		},
	}
	w.WriteJson(reply)
	return
}

func (api *Api) PutContact(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	var ec PutContactWrapper
	if err = r.DecodeJsonPayload(&ec); err != nil {
		log.Println(jsonDecodeError("contact update request"), err)
		rest.Error(w, INVALIDJSON, http.StatusBadRequest)
		return
	}

	// check for double (or more) circle IDs and error out if any are found
	for i, c1 := range ec.Contact.EmberCircles {
		for j, c2 := range ec.Contact.EmberCircles {
			if i != j && c1 == c2 {
				// found duplicate
				log.Println("contact-to-circle request with duplicate circle")
				rest.Error(w, DUPLICATECIRC, http.StatusBadRequest)
				return
			}
		}
	}

	// marshal back into Contact struct
	var contact db.Contact
	api.First(&contact, id)
	if err != nil {
		log.Println("Could not find contact")
		rest.NotFound(w, r)
		return
	}
	err = api.Model(&contact).Related(&contact.Onion, "Onion").Error
	if err != nil {
		log.Println("Could not find onion for contact")
		rest.NotFound(w, r)
		return
	}
	if contact.Id == 1 {
		// do not allow adding ourself to circles
		rest.Error(w, INVALIDCONTACT, http.StatusBadRequest)
		return
	}
	// overwrite updated fields
	contact.Nickname = ec.Contact.Nickname
	contact.Alias = ec.Contact.Alias
	contact.Trust = ec.Contact.Trust
	contact.RequestMessage = ec.Contact.RequestMessage
	circles := make([]db.Circle, len(ec.Contact.EmberCircles))

	oldstatus := contact.Status
	contact.Status = ec.Contact.Status

	api.Model(&contact).Association("Circles").Clear()
	// fill in circles
	for idx, circIdStr := range ec.Contact.EmberCircles {
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
		if circles[idx].Name == "Public" {
			// do not allow adding ppl to public circle
			rest.Error(w, INVALIDCIRCLE, http.StatusBadRequest)
			return
		}
		api.Model(&circles[idx]).Association("Contacts").Append(&contact)
	}

	contact.Circles = circles
	api.Save(&contact)

	if oldstatus == db.OPEN && ec.Contact.Status == db.SUCCESS {
		client.SetContactToSuccess(&contact, &api.SSNDB)
	}

}

/// circles of set A\B  (as in math)
func circle_difference(A []db.Circle, B []db.Circle) []db.Circle {
	var res []db.Circle
	for _, a := range A {
		found := false
		for _, b := range B {
			if a.Id == b.Id {
				found = true
				break
			}
		}
		if found == false {
			res = append(res, a)
		}
	}
	return res
}

func (api *Api) GetAllContacts(w rest.ResponseWriter, r *rest.Request) {
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

	contacts := []db.Contact{}

	if len(requestedIds) > 0 {
		err = api.Where(requestedIds).Find(&contacts).Error
	} else {
		err = api.Find(&contacts).Error
	}

	if err != nil {
		if err != gorm.RecordNotFound {
			log.Fatalln("Loading posts failed:", err)
		}
	}
	emberContacts := make([]db.EmberContactResponse, len(contacts))

	for idx, c := range contacts {
		emberContacts[idx].Contact = c
		api.Model(&c).Related(&c.Circles, "Circles")
		emberContacts[idx].EmberCircles = make([]int64, len(c.Circles))
		for i, circ := range c.Circles {
			emberContacts[idx].ProfilePictureId = api.GetProfilePictureId(c.OnionId)
			emberContacts[idx].EmberCircles[i] = circ.Id
		}
	}

	w.WriteJson(
		&GetAllContactsWrapper{
			Contacts: emberContacts,
		},
	)
}

func (api *Api) DeleteContact(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	contact := db.Contact{Id: id}

	if err = api.Find(&contact, contact).Error; err != nil {
		if err == gorm.RecordNotFound {
			rest.NotFound(w, r)
			return
		} else {
			log.Println(gormLoadError("contact"), err)
			rest.Error(w, INTERNALERROR, http.StatusInternalServerError)
		}
	}

	api.Model(&contact).Association("Circles").Clear()
	api.Delete(&contact)

	// remove old contact circle
	ccirc := db.Circle{
		Name:    contact.Alias,
		Creator: db.CREATOR_APP,
	}
	api.Where(ccirc).First(&ccirc)
	if ccirc.Id != 0 {
		api.Delete(ccirc)
	}

	w.WriteHeader(http.StatusOK)
}

func (api *Api) GetContact(w rest.ResponseWriter, r *rest.Request) {
	_, err := api.validateAuthHeader(r.Request)

	if err != nil {
		rest.Error(w, "Authorization invalid", http.StatusUnauthorized)
		return
	}

	_id := r.PathParam("id")
	id, err := strconv.ParseInt(_id, 10, 64)

	contact := db.Contact{}
	if api.First(&contact, id).Error != nil {
		rest.NotFound(w, r)
		return
	}

	emberContact := db.EmberContactResponse{}
	emberContact.Contact = contact
	emberContact.ProfilePictureId = api.GetProfilePictureId(contact.OnionId)
	api.Model(&contact).Related(&contact.Circles, "Circles")
	emberContact.EmberCircles = make([]int64, len(contact.Circles))
	for i, circ := range contact.Circles {
		emberContact.EmberCircles[i] = circ.Id
	}

	w.WriteJson(
		&GetContactWrapper{
			Contact: emberContact,
		},
	)
}
