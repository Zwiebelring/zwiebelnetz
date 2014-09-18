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

package main

import (
	"../logger"
	"./uictrl"
	"github.com/ant0ine/go-json-rest/rest"
	"log"
	"net/http"
	"os"
)

func main() {

	api := uictrl.Api{}
	api.InitDB()
	logger.Init(os.Stderr, os.Stderr, os.Stderr, os.Stderr, os.Stderr)

	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
		DisableXPoweredBy:        true,
	}

	handler.SetRoutes(
		//Contacts
		rest.RouteObjectMethod("GET", "/contacts", &api, "GetAllContacts"),
		rest.RouteObjectMethod("GET", "/contacts/:id", &api, "GetContact"),
		rest.RouteObjectMethod("PUT", "/contacts/:id", &api, "PutContact"),
		rest.RouteObjectMethod("POST", "/contacts", &api, "CreateContact"),
		rest.RouteObjectMethod("DELETE", "/contacts/:id", &api, "DeleteContact"),

		//Circles
		rest.RouteObjectMethod("GET", "/circles", &api, "GetAllCircles"),
		rest.RouteObjectMethod("GET", "/circles/:id", &api, "GetCircle"),
		rest.RouteObjectMethod("POST", "/circles", &api, "CreateCircle"),
		rest.RouteObjectMethod("DELETE", "/circles/:id", &api, "DeleteCircle"),

		//Posts
		rest.RouteObjectMethod("GET", "/posts", &api, "GetAllPosts"),
		rest.RouteObjectMethod("GET", "/posts/:id", &api, "GetPost"),
		rest.RouteObjectMethod("POST", "/posts", &api, "CreatePost"),
		rest.RouteObjectMethod("DELETE", "/posts/:id", &api, "DeletePost"),

		//Comments
		rest.RouteObjectMethod("GET", "/comments", &api, "GetAllComments"),
		rest.RouteObjectMethod("GET", "/comments/:id", &api, "GetComment"),
		rest.RouteObjectMethod("POST", "/comments", &api, "CreateComment"),
		rest.RouteObjectMethod("DELETE", "/comments/:id", &api, "DeleteComment"),

		//Profiles
		rest.RouteObjectMethod("GET", "/profiles", &api, "GetProfiles"),
		rest.RouteObjectMethod("GET", "/profiles/:id", &api, "GetProfile"),

		rest.RouteObjectMethod("DELETE", "/profiles/:id", &api, "DeleteProfile"),
		rest.RouteObjectMethod("POST", "/profiles", &api, "CreateProfile"),
		rest.RouteObjectMethod("PUT", "/profiles/:id", &api, "PutProfile"),
		rest.RouteObjectMethod("DELETE", "/profile_picture", &api, "DeleteProfilePicture"),

		//Onion
		rest.RouteObjectMethod("GET", "/onions", &api, "GetAllOnions"),
		rest.RouteObjectMethod("POST", "/onions", &api, "CreateOnion"),
		rest.RouteObjectMethod("GET", "/authors", &api, "GetAllAuthors"),
		rest.RouteObjectMethod("GET", "/originators", &api, "GetAllOriginators"),

		rest.RouteObjectMethod("GET", "/authors/:id", &api, "GetAuthor"),
		rest.RouteObjectMethod("GET", "/originators/:id", &api, "GetOriginator"),
		rest.RouteObjectMethod("GET", "/onions/:id", &api, "GetOnion"),

		//rest.RouteObjectMethod("POST",   "/authors",        &api, "CreateOnion"),
		//rest.RouteObjectMethod("POST",   "/originator",     &api, "CreateOnion"),

		//rest.RouteObjectMethod("DELETE", "/author/:id",     &api, "DeleteOnion"),
		//rest.RouteObjectMethod("DELETE", "/author/:id",     &api, "DeleteOnion"),

		////Users
		rest.RouteObjectMethod("POST", "/users", &api, "CreateUser"),
		rest.RouteObjectMethod("POST", "/authorize", &api, "Authorize"),
	)

	http.HandleFunc("/profile_picture", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Receiving profile picture ...\n")
		api.ProfilePictureHandler(w, r)
	})

	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		api.SyncHandler(w, r)
	})

	http.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		api.TriggerHandler(w, r)
	})

	http.HandleFunc("/pending_posts", func(w http.ResponseWriter, r *http.Request) {
		api.PendingPostsHandler(w, r)
	})

	http.HandleFunc("/pending_contacts", func(w http.ResponseWriter, r *http.Request) {
		api.PendingContactsHandler(w, r)
	})

	http.Handle("/api/", http.StripPrefix("/api", &handler))
	http.Handle("/", http.FileServer(http.Dir(".")))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
