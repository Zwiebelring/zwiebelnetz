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
	_ "container/list"
	"crypto/rsa"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"../client"
	"../core/crypto"
	"../core/crypto/auth"
	"../core/db"
	"../logger"
	"./protocol"
	_ "github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

func containsState(states []protocol.PacketType, state protocol.PacketType) bool {
	for _, elm := range states {
		if elm == state {
			return true
		}
	}
	return false
}

func connectionHandling(netconn net.Conn, dbconn db.SSNDB, key *rsa.PrivateKey) {
	//logger.Security(fmt.Sprint("net.Conn open :", netconn.RemoteAddr(), " on ", netconn.LocalAddr()))
	defer netconn.Close()

	var challengeR [32]byte
	var contact *db.Contact = nil
	head := make([]byte, protocol.HEADER_SIZE)
	buffer := make([]byte, 4096) // max length

	nextPossibleStates := []protocol.PacketType{
		protocol.AUTH,
		protocol.PULL,
		protocol.CONTACT_REQUEST}

	for {

		err := protocol.ReadPayload(netconn, head)
		if logger.ConditionalWarning(err, "could not read header from socket") {
			return
		}

		header := protocol.DecodeHeader(head)
		//logger.Info(fmt.Sprintf("Header(%c, %d)", header.PacketType, header.PacketLength))

		// protection for memory exhaustion attacks
		// increase maximum if connection authorized
		if 4096 < header.PacketLength {
			logger.Warning("corrupted network package (got more than 4096 bytes)")
			return
		}

		payload := buffer[0:header.PacketLength]
		if 0 < header.PacketLength { // if there is something to read
			err = protocol.ReadPayload(netconn, payload)
			if logger.ConditionalWarning(err, "could not read payload from socket") {
				return
			}
		}

		switch header.PacketType {
		case protocol.AUTH:

			if !containsState(nextPossibleStates, protocol.AUTH) {
				logger.Warning("impossible protocol state condition")
				return
			}

			pubKey, err := protocol.DecodeAuth(payload)
			if logger.ConditionalWarning(err, "could not decode public key blob") {
				return
			}

			onionstr := crypto.GetOnionAddress(&pubKey)

			contact = dbconn.GetFriendlyContactByOnion(onionstr)
			if contact == nil {
				return
			}

			var challenge auth.Challenge
			challenge, challengeR, err = auth.GenerateChallenge(&pubKey, &key.PublicKey)
			if logger.ConditionalWarning(err, "could not generate a challenge") {
				return
			}

			err = protocol.WritePacket(netconn, protocol.EncodeChallenge(&challenge))
			if logger.ConditionalWarning(err, "sending challenge packet failed!") {
				return
			}

			nextPossibleStates = []protocol.PacketType{protocol.RESPONSE}

		case protocol.PULL:

			if !containsState(nextPossibleStates, protocol.PULL) {
				logger.Security("impossible protocol state condition")
				return
			}

			timestamp, err := protocol.DecodePull(payload)
			if logger.ConditionalWarning(err, "could not decode timestamp") {
				return
			}
			/*
				if contact == nil {
					logger.Debug(fmt.Sprint("GET PULL REQUEST (unknown person) timestamp ", timestamp))
				} else {
					logger.Debug(fmt.Sprint("GET PULL REQUEST (", contact.Alias, ") timestamp ", timestamp))
				} */

			posts := dbconn.GetPosts(contact, timestamp)
			profiles := dbconn.GetProfiles(contact, timestamp)

			if contact == nil {
				logger.Debug(fmt.Sprint("SEND(", posts.Len(), " POSTS, ", profiles.Len(), " PROFILES) to (unknown person)"))
			} else {
				logger.Debug(fmt.Sprint("SEND(", posts.Len(), " POSTS, ", profiles.Len(), " PROFILES) to ", contact.Alias))
			}

			// reply posts
			for itr := posts.Front(); itr != nil; itr = itr.Next() {
				post := itr.Value.(*db.Post)
				reply := protocol.EncodePushPost(post)
				err := protocol.WritePacket(netconn, reply)
				if logger.ConditionalWarning(err, "sending post back failed!") {
					return
				}
			}
			// reply profile items
			for itr := profiles.Front(); itr != nil; itr = itr.Next() {
				profile := itr.Value.(*db.Profile)
				reply := protocol.EncodePushProfile(profile)
				err := protocol.WritePacket(netconn, reply)
				if logger.ConditionalWarning(err, "sending profile back failed!") {
					return
				}
			}

			err = protocol.WritePacket(netconn, protocol.EncodeSuccess())
			if err != nil {
				logger.Warning(fmt.Sprint(err, " sending success packet failed!"))
			}

			//logger.Debug("DONE!")

			return // no next possible states

		case protocol.TRIGGER:

			if !containsState(nextPossibleStates, protocol.TRIGGER) {
				logger.Security("impossible protocol state condition")
				return
			}

			err := protocol.WritePacket(netconn, protocol.EncodeSuccess())
			if err != nil {
				logger.Warning(fmt.Sprint(err, " sending success packet failed!"))
			}

			lastActivity := dbconn.GetContactsLastActivity(contact)

			posts, profiles, err := client.PullHandling(&dbconn, lastActivity, contact, key)
			if err != nil {
				return
			}

			dbconn.AddOrUpdateProfiles(profiles)
			dbconn.AddOrUpdatePosts(posts)

			for _, post := range posts {
				client.TriggerOnReceivingComment(&dbconn, &post)
			}

			//logger.Debug("DONE!")

			return

		case protocol.CHALLENGE:

			//logger.Debug("CHALLENGE")
			logger.Security("impossible protocol state condition")
			return

		case protocol.RESPONSE:

			//logger.Debug("RESPONSE")
			if !containsState(nextPossibleStates, protocol.RESPONSE) {
				logger.Security("impossible protocol state condition")
				return
			}

			response, err := protocol.DecodeResponse(payload)
			if logger.ConditionalWarning(err, "could not decode response") {
				return
			}

			if response.R != challengeR {
				logger.Security(fmt.Sprintf("invalid response from %s", contact.Onion.Onion))
				return
			}

			err = protocol.WritePacket(netconn, protocol.EncodeSuccess())
			if err != nil {
				logger.Warning(fmt.Sprint(err, " sending success packet failed!"))
			}
			//logger.Security(fmt.Sprintf("contact successful AUTH [%s]", contact.Onion.Onion))

			nextPossibleStates = []protocol.PacketType{protocol.TRIGGER, protocol.PULL}

		case protocol.PUSH_POST:

			logger.Debug("PUSH_POST")
			logger.Security("impossible protocol state condition")
			return

		case protocol.SUCCESS:

			logger.Debug("SUCCESS")
			logger.Security("impossible protocol state condition")
			return

		case protocol.CONTACT_REQUEST:

			if !containsState(nextPossibleStates, protocol.CONTACT_REQUEST) {
				logger.Security("impossible protocol state condition")
				return
			}

			contactReq, err := protocol.DecodeContactRequest(payload)
			if logger.ConditionalWarning(err, "could not decode contact request") {
				return
			}

			contact = dbconn.GetContactByOnion(contactReq.Onion)
			if contact != nil { // contact exists already

				if contact.Status == db.PENDING {
					client.SetContactToSuccess(contact, &dbconn)
				} else {
					logger.Debug("contact exists already, doing nothing")
				}

			} else {

				if contact == nil {
					contact = new(db.Contact)
				}

				contact.Status = db.OPEN
				contact.RequestMessage = contactReq.Message
				contact.Onion = dbconn.GetOnion(contactReq.Onion)
				contact.Alias = contactReq.Onion

				if contact.Onion.Id == 0 {
					logger.Debug("creating new contact and onion")
					contact.Onion = db.Onion{0, contactReq.Onion}
					dbconn.Create(contact)
				} else {
					logger.Debug("onion exists already, creating new contact")
					dbconn.Create(contact)
				}
			}

			err = protocol.WritePacket(netconn, protocol.EncodeSuccess())
			if err != nil {
				logger.Warning(fmt.Sprint(err, " sending success packet failed!"))
			}

			var p db.Pending
			dbconn.Find(&p, 1)
			p.Contacts = true
			dbconn.Save(&p)

			return

		case protocol.INVALID:

			logger.Debug("INVALID")
			logger.Security("corrupted network package")
			return

		default:

			logger.Security("undefined network package")
			return

		}
	}
}

func main() {
	logger.Init(os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	
	dbconn := db.SSNDB{}
	dbconn.Init()

	key := dbconn.GetKey()

	t := time.Minute * 5
	deadline := client.NewDeadline(key, t, client.SyncAllContacts, 0)
	deadline.Start()

	ln, err := net.Listen("tcp", "localhost:3141")
	if err != nil {
		log.Fatalln("could not listen, error: %s", err)
		dbconn.Close()
	}

	for {
		netconn, err := ln.Accept()
		if err != nil {
			log.Println("could ont accept connection: %s", err)
		}
		go connectionHandling(netconn, dbconn, key)
	}
}
