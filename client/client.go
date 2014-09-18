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

package client

import (
	"../core/crypto/auth"
	"../core/db"
	"../external"
	"../logger"
	"../sync/protocol"
	"crypto/rsa"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type OnionConnection struct {
	*net.TCPConn
	Onion       string
	Established bool
}

func ConnectToOnion(onion string) (OnionConnection, error) {
	// step 1: connect to TOR proxy
	laddr, _ := net.ResolveTCPAddr("tcp", "localhost")
	raddr, _ := net.ResolveTCPAddr("tcp", "localhost:9050")
	conn, err := net.DialTCP("tcp", laddr, raddr)
	//	_ = conn.SetDeadline(time.Now().Add(time.Second*5))
	if err != nil {
		return OnionConnection{conn, onion, false}, err
	}
	// step 2: tell TOR proxy to connect to onion address
	err = socks.Connect(conn, onion)
	if err != nil {
		conn.Close()
		return OnionConnection{conn, onion, false}, err
	}
	// success
	return OnionConnection{conn, onion, true}, nil
}

func (conn OnionConnection) Auth(key *rsa.PrivateKey) error {
	// 1. send auth request
	//logger.Debug("...sending auth request")
	pkg := protocol.EncodeAuth(key.PublicKey)
	conn.Write(pkg)

	// 2 read and check challenge
	//logger.Debug("...waiting for challenge")
	header, err := protocol.ReadHeader(conn)
	if err != nil {
		return errors.New("error while reading challenge header: " + err.Error())
	}
	if header.PacketType != protocol.CHALLENGE {
		return errors.New("wrong package type waiting for challenge")
	}
	buffer := make([]byte, 16384)
	_ = protocol.ReadPayload(conn, buffer[:header.PacketLength])
	challenge, err := protocol.DecodeChallenge(buffer[:header.PacketLength])
	if err != nil {
		return errors.New("could not decode challenge" + err.Error())
	}

	// 3 build and send response
	//logger.Debug("...generating response")
	response, err := auth.GenerateResponse(challenge,
		key,
		conn.Onion)
	if err != nil {
		return errors.New("could not build resonionsponse: " + err.Error())
	}
	//logger.Debug("...sending response")
	pkg = protocol.EncodeResponse(&response)
	conn.Write(pkg)

	// 4. await success notification
	if header, err = protocol.ReadHeader(conn); err != nil {
		return errors.New("could not receive success notification: " + err.Error())
	}
	if header.PacketType != protocol.SUCCESS {
		return fmt.Errorf("expected success packet, but got %s instead",
			header.PacketType)
	}
	return nil
}

func (conn OnionConnection) Pull(timestamp int64) ([]db.Post, []db.Profile, error) {
	posts := []db.Post{}
	profiles := []db.Profile{}

	//logger.Debug(fmt.Sprint("sending PULL with timestamp ", timestamp))
	conn.Write(protocol.EncodePull(timestamp))

	// default length of post: 64 Kilobyte, relocation is implemented
	length := uint32(65536)
	buffer := make([]byte, length)
	for {
		header, err := protocol.ReadHeader(conn)
		if err != nil {
			return posts,
				profiles,
				errors.New("error while receiving posts: " + err.Error())
		}

		if 16777216 < header.PacketLength { // if payload greater than 16 Megabyte
			return posts,
				profiles,
				errors.New("Received payload is greater than 16 Megabyte")
		} else if length < header.PacketLength { // relocate buffer if required
			buffer = nil // garbage collection help
			length = header.PacketLength
			buffer = make([]byte, length)
		}

		if header.PacketType == protocol.SUCCESS {
			// finished, no more replies
			return posts, profiles, nil
		} else if header.PacketType == protocol.PUSH_POST {

			_ = protocol.ReadPayload(conn, buffer[:header.PacketLength])
			post, err := protocol.DecodePushPost(
				buffer[:header.PacketLength],
				conn.Onion)
			if err != nil {
				return posts,
					profiles,
					errors.New("decode of post failed: " + err.Error())
			}
			posts = append(posts, post)

		} else if header.PacketType == protocol.PUSH_PROFILE {

			_ = protocol.ReadPayload(conn, buffer[:header.PacketLength])
			profile, err := protocol.DecodePushProfile(
				buffer[:header.PacketLength],
				conn.Onion)
			if err != nil {
				return posts,
					profiles,
					errors.New("decode of post failed: " + err.Error())
			}
			profiles = append(profiles, profile)

		} else {
			return posts,
				profiles,
				fmt.Errorf("expected push post, but got %s\n", header.PacketType)
		}
	}
}

func (conn OnionConnection) Trigger() error {
	//logger.Debug("sending TRIGGER")
	conn.Write(protocol.EncodeTrigger())
	header, err := protocol.ReadHeader(conn)
	if err != nil {
		return errors.New("error while waiting for SUCCESS: " + err.Error())
	} else if header.PacketType != protocol.SUCCESS {
		return errors.New("expected SUCCESS, but got " + string(header.PacketType))
	}
	return nil
}

func (conn OnionConnection) ContactRequest(cr protocol.ContactRequest) error {
	//logger.Debug("sending contact request")
	conn.Write(protocol.EncodeContactRequest(cr))
	header, err := protocol.ReadHeader(conn)
	if err != nil {
		return errors.New("error while waiting for SUCCESS message " + err.Error())
	} else if header.PacketType != protocol.SUCCESS {
		return errors.New("expected success message but received " + string(header.PacketType))
	}
	return nil
}

func TriggerHandling(dbconn *db.SSNDB, onions []db.Onion) {
	key := dbconn.GetKey()
	for _, onion := range onions {
		onionconn, err := ConnectToOnion(onion.Onion)
		/*if err != nil {
		logger.Warning(fmt.Sprintf("could not conect to %s addr (TRIGGER)", onion.Onion))
		*/
		if err == nil {
			err = onionconn.Auth(key)
			if err != nil {
				logger.Warning(fmt.Sprintf("authentication fail: to %s addr (TRIGGER)", onion.Onion))
			} else {
				err = onionconn.Trigger()
				if err != nil {
					logger.Warning(fmt.Sprintf("could not send a TRIGGER to %s addr", onion.Onion))
				}
			}
			onionconn.Close()
		}
	}
}

func ContactRequestHandling(contact *db.Contact, myOnion *db.Onion) error {
	if contact == nil {
		return errors.New("nil argument")
	}
	logger.Debug(fmt.Sprint("SEND CONTACT REQUEST (", contact.Alias, ")"))

	onionconn, err := ConnectToOnion(contact.Onion.Onion)
	if err != nil { //logger.ConditionalWarning(err, fmt.Sprintf("could not conect to %s addr", contact.Onion.Onion)) {
		return err
	}
	defer onionconn.Close()

	cr := protocol.ContactRequest{
		Message: contact.RequestMessage,
		Onion:   myOnion.Onion,
	}

	return onionconn.ContactRequest(cr)
}

func PullHandling(dbconn *db.SSNDB, lastActivity int64, contact *db.Contact, key *rsa.PrivateKey) ([]db.Post, []db.Profile, error) {
	posts := []db.Post{}
	profiles := []db.Profile{}
	if contact == nil || key == nil {
		return posts, profiles, errors.New("nil argument")
	}
	//logger.Debug(fmt.Sprintf("SEND PULL REQUEST (%s): timestamp %d\n", contact.Alias, lastActivity))

	onionconn, err := ConnectToOnion(contact.Onion.Onion)
	if err != nil { //logger.ConditionalWarning(err, fmt.Sprintf("could not conect to %s addr", contact.Onion.Onion)) {
		return posts, profiles, err
	}
	defer onionconn.Close()

	err = onionconn.Auth(key)
	if logger.ConditionalWarning(err, "(authentication fail, trying to PULL without AUTH..)") {
		onionconn, err = ConnectToOnion(contact.Onion.Onion) // needed for PULL request
		if logger.ConditionalWarning(err, "could not conect to onion addr") {
			return posts, profiles, err
		}
	} else {
		// auth successful, set contact's status to SUCCESS
		if contact.Status != db.SUCCESS {
			SetContactToSuccess(contact, dbconn)
		}
	}

	posts, profiles, err = onionconn.Pull(lastActivity)
	if logger.ConditionalWarning(err, "client could not PULL") {
		return posts, profiles, err
	}

	logger.Debug(fmt.Sprint("RECEIVED(", len(posts), " POSTS, ", len(profiles), " PROFILES) from ", contact.Alias))

	return posts, profiles, nil
}

func SyncAllContacts(key *rsa.PrivateKey) {

	dbconn := db.SSNDB{}
	dbconn.Init()
	//dbconn.LogMode(true)
	contacts := []db.Contact{}
	dbconn.Where(db.Contact{Status: db.SUCCESS}).Or(db.Contact{Status: db.PENDING}).Or(db.Contact{Status: db.FOLLOWING}).Find(&contacts)
	myOnion := dbconn.GetSelfOnion()
	// A WaitGroup waits for a collection of goroutines to finish.
	var wg sync.WaitGroup
	wg.Add(len(contacts)) // set the WaitGroup counter.

	for _, contact := range contacts {
		go func(dbconn *db.SSNDB, key *rsa.PrivateKey, contact db.Contact, wg *sync.WaitGroup) {
			dbconn.Model(&contact).Related(&contact.Onion, "OnionId")
			lastActivity := dbconn.GetContactsLastActivity(&contact)
			posts, profiles, err := PullHandling(dbconn, lastActivity, &contact, key)
			if err == nil {
				dbconn.AddOrUpdateProfiles(profiles)
				dbconn.AddOrUpdatePosts(posts)

				for _, post := range posts {
					TriggerOnReceivingComment(dbconn, &post)
				}

			}
			if db.PENDING == contact.Status {
				ContactRequestHandling(&contact, &myOnion)
				/*if err != nil {
					logger.Warning(fmt.Sprint(err))
				} */
			}
			wg.Done() // decrements the WaitGroup counter.
		}(&dbconn, key, contact, &wg)
	}

	wg.Wait() // blocks until the WaitGroup counter is zero.

	dbconn.Close()
}

func TriggerOnReceivingComment(dbconn *db.SSNDB, comment *db.Post) {
	if comment.ParentId == 0 {
		return
	}

	var parentPost db.Post
	dbconn.First(&parentPost, comment.ParentId)

	if parentPost.OriginatorId != dbconn.GetSelfOnion().Id {
		return
	}

	comment.PublishedAt = time.Now()
	comment.Published = true
	dbconn.Save(comment)

	circles := dbconn.GetPostCircles(comment)
	TriggerCircles(dbconn, circles)
}

func SetContactToSuccess(contact *db.Contact, dbconn *db.SSNDB) {
	// set contact status to success
	logger.Info("Updating status of contact " + contact.Alias + " to \"success\"")
	contact.Status = db.SUCCESS
	dbconn.Save(contact)

	// add circle for new contact
	circle := db.Circle{Name: contact.Alias, Creator: db.CREATOR_APP}
	dbconn.Find(&circle, circle)
	if circle.Id != 0 {
		logger.Warning("circle \"" + circle.Name + "\" already exists")
		return
	}
	logger.Debug("adding circle \"" + circle.Name + "\"")
	dbconn.Create(&circle)

	// add contact to circle
	logger.Debug("adding user " + contact.Alias + " (" + contact.Nickname + ") to circle \"" + circle.Name + "\"\n")
	dbconn.Model(&circle).Association("Contacts").Append(*contact)

	var p db.Pending
	dbconn.Find(&p, 1)
	p.Contacts = true
	dbconn.Save(&p)
}

func TriggerCircles(dbconn *db.SSNDB, circles []db.Circle) {
	// sync trigger
	// we abuse map type as set here, since golang does not have sets...
	// we are only interested in the KEY, we ignore  the value
	self := dbconn.GetSelfOnion()
	onionMap := map[db.Onion]bool{}
	for _, circle := range circles {
		// we need all onions in circle for trigger
		var contacts []db.Contact
		if circle.Name == "Public" {
			dbconn.Find(&contacts)
		} else {
			dbconn.Model(&circle).Related(&circle.Contacts, "Contacts")
			contacts = circle.Contacts
		}
		for _, contact := range contacts {
			var onion db.Onion
			dbconn.Model(&contact).Related(&onion, "Onion")
			if onion.Id != self.Id {
				onionMap[onion] = true
			}
		}
	}

	var onionsToTrigger []db.Onion
	for onionKey, _ := range onionMap {
		onionsToTrigger = append(onionsToTrigger, onionKey)
	}

	logger.Debug("I am going to trigger the following onions in a goroutine now: ")
	logger.Debug(fmt.Sprint(onionsToTrigger))
	go TriggerHandling(dbconn, onionsToTrigger)
}
