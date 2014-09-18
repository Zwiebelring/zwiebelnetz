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

package db

type RelationStatus uint8

const (
	BLOCKED   RelationStatus = 0
	OPEN                     = 1
	PENDING                  = 2
	SUCCESS                  = 3
	FOLLOWING                = 4
)

type Contact struct {
	Id             int64          `json:"id"`
	Onion          Onion          `json:"-"`
	OnionId        int64          `json:"onion" sql:"not null;unique"`
	Nickname       string         `json:"nickname"`
	Alias          string         `json:"alias" sql:"not null;unique"`
	Trust          int            `json:"trust" sql:"not null;default:0"`
	Status         RelationStatus `json:"status" sql:"not null"`
	RequestMessage string         `json:"request_message"`
	Circles        []Circle       `json:"-" gorm:"many2many:circle_contacts;"`
}

type EmberContactResponse struct {
	Contact
	ProfilePictureId int64   `json:"profilePictureId"`
	EmberCircles     []int64 `json:"circles"`
}

type EmberContactRequest struct {
	OnionId        string         `json:"onion"`
	Nickname       string         `json:"nickname"`
	Alias          string         `json:"alias"`
	Trust          int            `json:"trust" sql:"not null;default:0"`
	Status         RelationStatus `json:"status" sql:"not null"`
	RequestMessage string         `json:"request_message"`
	EmberCircles   []string       `json:"circles"`
}

func StatusString(rs RelationStatus) string {
	switch rs {
	case BLOCKED:
		return "blocked"
	case OPEN:
		return "open"
	case PENDING:
		return "pending"
	case SUCCESS:
		return "success"
	}
	return ""
}
