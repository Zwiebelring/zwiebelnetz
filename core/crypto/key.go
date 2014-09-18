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

package crypto

import (
	"../../logger"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base32"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

func PubKeyEq(key1 rsa.PublicKey, key2 rsa.PublicKey) bool {
	return key1.N.Cmp(key2.N) == 0 && key1.E == key2.E
}

func MarshalPKCS1PublicKey(key *rsa.PublicKey) []byte {
	b, _ := asn1.Marshal(*key)
	return b
}
func UnmarshalPKCS1PublicKey(bytes []byte) (rsa.PublicKey, error) {
	var pubKey rsa.PublicKey
	rest, err := asn1.Unmarshal(bytes, &pubKey)
	if err != nil || len(rest) > 0 {
		return pubKey, errors.New("could not decode pkcs1 public key")
	}
	return pubKey, nil
}

func ReadKey(path string) *rsa.PrivateKey {
	var pemKey [2048]byte
	file, _ := os.Open(path)
	n, err := file.Read(pemKey[:])
	file.Close()
	logger.ConditionalError(err, "Failed to read private key")

	// decode pem
	pkcsKey, _ := pem.Decode(pemKey[:n])

	// decode pkcs1
	key, err := x509.ParsePKCS1PrivateKey(pkcsKey.Bytes)
	logger.ConditionalError(err, "Failed to parse PKCS1 private key")

	return key
}

func Pem2Key(pemKey []byte) *rsa.PrivateKey {
	// decode pem
	pkcsKey, _ := pem.Decode(pemKey)

	// decode pkcs1
	key, err := x509.ParsePKCS1PrivateKey(pkcsKey.Bytes)
	logger.ConditionalError(err, "Failed to parse PKCS1 private key")

	return key
}

func GetOnionAddress(key *rsa.PublicKey) string {
	// encode public key tp pkcs1
	pub_pkcs1 := MarshalPKCS1PublicKey(key)

	sha := sha1.Sum(pub_pkcs1)
	onion := base32.StdEncoding.EncodeToString(sha[:])
	return fmt.Sprintf("%s.onion", strings.ToLower(onion)[0:16])
}
