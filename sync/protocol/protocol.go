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

package protocol

import (
	"../../core/crypto/auth"
	"../../core/db"
	"crypto/rsa"
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"time"
)

/* Packet Format
 *
 * 1 byte              -- type
 * 4 bytes             -- payload len
 * <payload len> bytes -- payload
 */

type PacketType uint8

const (
	AUTH            PacketType = 'A'
	PULL                       = 'P'
	TRIGGER                    = 'T'
	CHALLENGE                  = 'C'
	RESPONSE                   = 'R'
	PUSH_POST                  = 'Q'
	SUCCESS                    = 'S'
	CONTACT_REQUEST            = 'B'
	PUSH_PROFILE               = 'U'
	INVALID                    = 0
)

const (
	HEADER_SIZE          int = 5
	MAX_PACKET_BYTE_SIZE     = 1024 // one Kilobyte
)

const (
	TIMEOUT time.Duration = 60
)

type Header struct {
	PacketType   PacketType
	PacketLength uint32
}

/* Socket Timeouts */
func DefaultTimeout() (t time.Time) {
	var dur time.Duration = TIMEOUT * time.Second
	t = time.Now().Add(dur)
	return t
}

func TimeoutSecondsOffset(timeout time.Duration) (t time.Time) {
	var dur time.Duration = timeout * time.Second
	t = time.Now().Add(dur)
	return t
}

/* Package */

func EncodePacket(packetType PacketType, payload []byte) []byte {
	var header [HEADER_SIZE]byte
	header[0] = byte(packetType)
	binary.BigEndian.PutUint32(header[1:], uint32(len(payload)))
	return append(header[:], payload...)
}

func DecodeHeader(header []byte) Header {
	if len(header) < HEADER_SIZE {
		log.Print("syncer.protocol.DecodeHeader: len(header) < ", HEADER_SIZE)
		return Header{INVALID, 0}
	}

	return Header{
		PacketType(header[0]),
		binary.BigEndian.Uint32(header[1:HEADER_SIZE]),
	}
}

func WritePacket(conn net.Conn, reply []byte) error {
	var sum int = 0
	var length int = len(reply)
	var step int
	var packet []byte
	for sum < length {
		step = sum + MAX_PACKET_BYTE_SIZE
		if step < length {
			packet = reply[sum:step]
		} else {
			packet = reply[sum:length]
		}
		conn.SetWriteDeadline(DefaultTimeout())
		size, err := conn.Write(packet)
		if err != nil {
			return err
		}
		sum += size
	}
	return nil
}

func ReadPayload(conn net.Conn, buffer []byte) error {
	var sum int = 0
	var length int = len(buffer)
	for sum < length {
		conn.SetDeadline(DefaultTimeout())
		size, err := conn.Read(buffer[sum:length])
		if err != nil {
			return err
		}
		sum += size
	}
	return nil
}

func ReadHeader(conn net.Conn) (Header, error) {
	var header Header = Header{INVALID, 0}
	buffer := make([]byte, HEADER_SIZE)
	err := ReadPayload(conn, buffer)
	if err != nil {
		return header, err
	}
	header = DecodeHeader(buffer)
	return header, nil
}

/* Auth Payload */

func EncodeAuth(ourPublicKey rsa.PublicKey) []byte {
	return EncodePacket(AUTH, JsonOrDie(ourPublicKey))
}

func DecodeAuth(payload []byte) (rsa.PublicKey, error) {
	var pubKey rsa.PublicKey
	err := json.Unmarshal(payload, &pubKey)
	return pubKey, err
}

/* Challenge Payload */

func EncodeChallenge(challenge *auth.Challenge) []byte {
	return EncodePacket(CHALLENGE, JsonOrDie(*challenge))
}

func DecodeChallenge(payload []byte) (auth.Challenge, error) {
	var challenge auth.Challenge
	err := json.Unmarshal(payload, &challenge)
	return challenge, err
}

/* Response Payload */

func EncodeResponse(response *auth.Response) []byte {
	return EncodePacket(RESPONSE, JsonOrDie(*response))
}

func DecodeResponse(payload []byte) (auth.Response, error) {
	var response auth.Response
	err := json.Unmarshal(payload, &response)
	return response, err
}

/* Pull Payload */

func EncodePull(timestamp int64) []byte {
	return EncodePacket(PULL, JsonOrDie(timestamp))
}

func DecodePull(payload []byte) (int64, error) {
	var timestamp int64
	err := json.Unmarshal(payload, &timestamp)
	return timestamp, err
}

/* Trigger Payload */

func EncodeTrigger() []byte {
	return EncodePacket(TRIGGER, []byte{})
}

/* Success Payload */
func EncodeSuccess() []byte {
	return EncodePacket(SUCCESS, []byte{})
}

/* Post Payload */

type PushPost struct {
	Message     string
	PostedAt    int64
	PublishedAt int64
	TTL         byte
	Author      string
	Hash        string
	ParentHash  string
}

func EncodePushPost(post *db.Post) []byte {
	pullReply := PushPost{
		post.Message,
		post.PostedAt.Unix(),
		post.PublishedAt.Unix(),
		post.TTL,
		post.Author.Onion,
		post.Hash,
		post.ParentHash}
	json := JsonOrDie(pullReply)
	return EncodePacket(PUSH_POST, json)
}

func DecodePushPost(payload []byte, origin string) (db.Post, error) {
	var pub db.Post
	var pp PushPost
	err := json.Unmarshal(payload, &pp)
	if err != nil {
		return pub, err
	}
	pub.Message = pp.Message
	pub.PostedAt = time.Unix(pp.PostedAt, 0)
	pub.RemotePublishedAt = time.Unix(pp.PublishedAt, 0)
	pub.TTL = pp.TTL
	pub.Originator = db.Onion{0, origin}
	pub.Author = db.Onion{0, pp.Author}
	pub.Hash = pp.Hash
	pub.ParentHash = pp.ParentHash
	return pub, err
}

/* Contact Request Payload */
type ContactRequest struct {
	Message string
	Onion   string
}

func EncodeContactRequest(cr ContactRequest) []byte {
	return EncodePacket(CONTACT_REQUEST, JsonOrDie(cr))
}

func DecodeContactRequest(payload []byte) (ContactRequest, error) {
	var cr ContactRequest
	err := json.Unmarshal(payload, &cr)
	return cr, err
}

/* Profile Payload */
type PushProfile struct {
	Key       string
	Value     string
	ChangedAt int64
}

func EncodePushProfile(prof *db.Profile) []byte {
	pp := PushProfile{
		Key:       prof.Key,
		Value:     prof.Value,
		ChangedAt: prof.ChangedAt.Unix(),
	}

	json := JsonOrDie(pp)
	return EncodePacket(PUSH_PROFILE, json)
}

func DecodePushProfile(payload []byte, onion string) (db.Profile, error) {
	var pp PushProfile
	err := json.Unmarshal(payload, &pp)

	prof := db.Profile{
		Key:       pp.Key,
		Value:     pp.Value,
		ChangedAt: time.Unix(pp.ChangedAt, 0),
		Onion:     db.Onion{Id: 0, Onion: onion},
	}

	return prof, err
}

/* encodes to json or dies if it fails */

func JsonOrDie(x interface{}) []byte {
	json, err := json.Marshal(x)
	if err != nil {
		log.Fatalln("could not encode to json, error=", err, " value=", x)
	}
	return json
}
