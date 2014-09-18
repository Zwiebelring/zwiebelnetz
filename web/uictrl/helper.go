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
	"strconv"

	"../../core/db"
	"github.com/ant0ine/go-json-rest/rest"
)

type Api struct {
	db.SSNDB
}

type Idable interface {
	Id() int64
}

func (api *Api) InitDB() {
	api.Init()

	// for debug purpose
	//api.LogMode(true)
}

func appendIfMissing(array []db.Onion, element db.Onion) []db.Onion {
	for _, ele := range array {
		if ele.Id == element.Id {
			return array
		}
	}
	return append(array, element)
}

func ToIds(array []string) (err error, intArray []int64) {
	intArray = make([]int64, len(array))
	for index, value := range array {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err, []int64{}
		}
		intArray[index] = id
	}
	return nil, intArray
}

func ToId(value string) (err error, id int64) {
	id, err = strconv.ParseInt(value, 10, 64)
	return
}

func GetRequestedIds(r *rest.Request) (ids []int64, err error) {
	idMap := r.URL.Query()
	for _, queryId := range idMap["ids[]"] {
		id, err := strconv.ParseInt(queryId, 10, 64)
		if err != nil {
			return ids, err
		}
		ids = append(ids, int64(id))
	}
	return
}
