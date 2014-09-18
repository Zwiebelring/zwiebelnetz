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

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"
)

type User struct {
	Id        int64     `json:"id"`
	Username  string    `json:"username"        sql:"size:1024" `
	Password  string    `json:"-"               sql:"size:1024" `
	Salt      string    `json:"-"               sql:"size:1024" `
	AuthToken string    `json:"auth_token"      sql:"size:1024" `
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Onion     Onion     `json:"-"`
	OnionId   int64     `json:"contact_id"`
	PemKey    string    `json:"-"`
}

func (user *User) SetPassword(password string) (err error) {
	salt := make([]byte, 32)
	_, err = rand.Read(salt)

	hasher := sha256.New()
	hasher.Write(append([]byte(password), salt...))

	hash := hasher.Sum(nil)
	encodedPassword := base64.StdEncoding.EncodeToString(hash)
	encodedSalt := base64.StdEncoding.EncodeToString(salt)

	user.Password = encodedPassword
	user.Salt = encodedSalt

	return
}

func (user *User) CheckPassword(password string) (err error) {

	decodedPassword, err := base64.StdEncoding.DecodeString(user.Password)
	if err != nil {
		return
	}

	decodedSalt, err := base64.StdEncoding.DecodeString(user.Salt)
	if err != nil {
		return
	}

	hasher := sha256.New()
	hasher.Write(append([]byte(password), decodedSalt...))

	if !bytes.Equal(hasher.Sum(nil), decodedPassword) {
		return errors.New("Password missmatch")
	}

	return
}

func (user *User) GenerateAuthToken() (err error) {

	token := make([]byte, 255)
	_, err = rand.Read(token)
	if err != nil {
		return
	}

	encodedToken := base64.StdEncoding.EncodeToString(token)

	user.AuthToken = encodedToken
	return
}
