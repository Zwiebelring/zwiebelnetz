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

package auth

import "crypto/rand"
import "crypto/rsa"
import "crypto/sha256"
import "log"
import _ "fmt"
import "errors"
import "../../crypto"

/* This is a direct implementation of chapter 10.3.3 of the handbook of applied cryptography:
   (i) Challenge-response based on public-key decryption
       Identification based on PK decryption and witness. Consider the following protocol:
       A <- B: h(r),B,P_A(r,B)        (1)
       A <- B: r                      (2)

   (Before (2), A verifies that r' = r and B' = B)

   Note: We use send P_A(r,h(B)) instead of P_A(r,B) due to limits
         on the size of the to-be encrypted data with PKCS1v15.
*/

// TODO: check if h(B) is okay, crypto stackexchange post done

type Challenge struct {
	HR     [32]byte  // witness of the challenge r (sha256 hash)
	PubKey [140]byte // the hash of the public key of B
	Enc    []byte    // encrypted blob containing Enc(r,B)
}

type Response struct {
	R [32]byte
}

func GenerateResponse(Challenge Challenge, A_PrivK *rsa.PrivateKey, B_Onion string) (Response, error) {
	var response Response
	// variables
	var hb_ [32]byte // B'
	var r_ [32]byte  // R'

	// check that onion of B == the host which we connected to in the first place!
	challenge_b, err := crypto.UnmarshalPKCS1PublicKey(Challenge.PubKey[:])
	if err != nil {
		return Response{[32]byte{0}}, errors.New("invalid B given by remote!")
	}
	if B_Onion != crypto.GetOnionAddress(&challenge_b) {
		return Response{[32]byte{0}}, errors.New("wrong B given by remote!")
	}

	{ // decrypt (r,h(B))
		dec, err := rsa.DecryptOAEP(
			sha256.New224(), /* hash function  */
			rand.Reader,     /* random source  */
			A_PrivK,         /* decryption key */
			Challenge.Enc,   /* ciphertext     */
			[]byte{})        /* label          */
		if err != nil {
			return response, errors.New("Auth.GenerateResponse: Could not decrypt challenge: " + err.Error())
		}
		copy(r_[:], dec[0:32])
		copy(hb_[:], dec[32:64])
		if err != nil {
			return response, errors.New("Auth.GenerateResponse: Could not decode b_")
		}
	}
	// check that h(B) == h(B') => B == B'
	if sha256.Sum256(Challenge.PubKey[:]) != hb_ {
		return response, errors.New("B and B' differ!")
	}

	// check that h(r) == h(r_)
	if sha256.Sum256(r_[:]) != Challenge.HR {
		return response, errors.New("h(r) and h(r') differ!")
	}

	return Response{r_}, nil
}

func GenerateChallenge(
	/* to verify */ A *rsa.PublicKey,
	/* ours*/ B *rsa.PublicKey) (Challenge, [32]byte, error) {

	// variables
	var challenge Challenge
	var r [32]byte
	var err error

	// encode b
	copy(challenge.PubKey[:], crypto.MarshalPKCS1PublicKey(B))
	if err != nil {
		log.Fatalln("Auth.GenerateChallenge: cannot marshall public key b, error: " + err.Error())
	}

	// generate r
	_, err = rand.Read(r[:])
	if err != nil {
		log.Fatalln("Auth.GenerateChallenge: cannot get secure random number, error: " + err.Error())
	}
	//fmt.Printf("r: %.x\n", r)

	// get h(r)
	challenge.HR = sha256.Sum256(r[:])

	{ // get P_A(r,B)
		buf := make([]byte, 64)
		copy(buf[0:32], r[:])
		hb := sha256.Sum256(challenge.PubKey[:])
		copy(buf[32:64], hb[:])

		// encrypt (r,h(B))
		enc, err := rsa.EncryptOAEP(
			sha256.New224(), /* hash function  */
			rand.Reader,     /* random source  */
			A,               /* encryption key */
			buf,             /* plaintext      */
			[]byte{})        /* label          */
		if err != nil {
			return challenge, r, errors.New("Auth.GenerateChallenge: Could not encrypt (r,B), due to " + err.Error())
		}
		challenge.Enc = enc
	}
	// fill challenge struct
	return challenge, r, nil
}
