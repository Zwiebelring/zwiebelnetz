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
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"../client"
	"../core/crypto"
	"../core/db"
	"../logger"
	"../sync/protocol"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logger.Init(os.Stderr, os.Stderr, os.Stderr, os.Stderr, os.Stderr)

	logmode := flag.Bool("log", false, "enable log mode")
	command := flag.String("cmd", "", "the command to execute")
	onion := flag.String("onion", "", "the onion address to connect to")
	keyfile := flag.String("key", "", "the path to the rsa private key")
	nickname := flag.String("nickname", "", "the nickname for the contact")
	password := flag.String("password", "", "your password")
	circname := flag.String("circname", "", "the name of one or more circles, separated by comma")
	creator := flag.Int64("creator", 0, "creator of circle: 0 by app, 1 by user")
	post := flag.String("post", "", "the message of the post")
	status := flag.String("status", "", "contact status (blocked,open,pending,success)")
	var timestamp *int64 = flag.Int64("timestamp", 0, "the timestamp to send")
	var id *int64 = flag.Int64("id", 0, "ID of something")
	key := flag.String("prof_key", "", "Profile Key")
	value := flag.String("prof_value", "", "Profile Value")

	flag.Parse()

	logger.AssertError(len(*command) > 0, "please provide a command argument")

	var dbconn db.SSNDB
	var conn client.OnionConnection
	var err error

	dbconn.Init()
	dbconn.LogMode(*logmode)

	commands := strings.Split(*command, ";")

	for _, c := range commands {
		switch c {
		case "conn":
			logger.AssertError(len(*onion) > 0, "please provide a valid onion address")
			logger.Debug("...connecting to onion")
			conn, err = client.ConnectToOnion(*onion)
			logger.ConditionalError(err, "could not connect to onion")
			fmt.Printf("successfully connected to %s\n", *onion)
			defer conn.Close()

		case "trigger":
			logger.AssertError(conn.Established, "no connection established, use command \"conn\"")
			err := conn.Trigger()
			logger.ConditionalError(err, "could not run trigger proc")
			fmt.Printf("successfully sent trigger\n")

		case "list-onions":
			onions := []db.Onion{}
			dbconn.Find(&onions)
			for _, o := range onions {
				fmt.Println(o.Onion)
			}

		case "list-contacts":
			contacts := []db.Contact{}
			dbconn.Find(&contacts)
			for _, c := range contacts {
				// fill in onion address
				dbconn.Model(&c).Related(&c.Onion)
				fmt.Printf("%s ------\n", c.Onion.Onion)
				fmt.Printf("  alias:      %s\n", c.Alias)
				fmt.Printf("  nick:       %s\n", c.Nickname)
				fmt.Printf("  trust:      %d\n", c.Trust)
				fmt.Printf("  status:     %s\n", db.StatusString(c.Status))

			}

		case "add-contact":
			logger.AssertError(len(*onion) > 0, "please provide a valid onion address")
			logger.AssertError(len(*nickname) > 0, "please provide a nickname")
			contact := db.Contact{
				Alias:  *nickname,
				Trust:  0,
				Status: db.PENDING,
			}

			if dbconn.GetContactByOnion(*onion) != nil {
				fmt.Printf("contact exists already, doing nothing\n")
			} else if dbconn.GetOnion(*onion).Id != 0 {
				fmt.Printf("onion exists already, reusing it\n")
				contact.Onion = dbconn.GetOnion(*onion)
				dbconn.Create(&contact)
			} else {
				fmt.Printf("creating new contact and onion\n")
				contact.Onion = db.Onion{0, *onion}
				dbconn.Create(&contact)
			}

		case "contact-req":
			logger.AssertError(len(*onion) > 0, "please provide a valid onion address")
			logger.AssertError(len(*post) > 0, "please provide a message for the contact request (using post field)")
			contact := dbconn.GetContactByOnion(*onion)
			logger.AssertError(contact.Id > 0, "unknown contact")
			cr := protocol.ContactRequest{
				Onion:   dbconn.GetSelfOnion().Onion,
				Message: *post,
			}
			err := conn.ContactRequest(cr)
			if err != nil {
				logger.Info(fmt.Sprint("Failed to send contact request: ", err))
			}

		case "init":
			logger.AssertError(len(*onion) > 0, "please provide your onion address")
			logger.AssertError(len(*nickname) > 0, "please provide your nick name")
			logger.AssertError(len(*password) > 0, "please provide your nick name")
			logger.AssertError(len(*keyfile) > 0, "please the path to the key for initialization")

			// Read key
			var pemKeyBuf [2048]byte
			file, err := os.Open(*keyfile)
			logger.ConditionalError(err, "could not open key file")
			n, err := file.Read(pemKeyBuf[:])
			logger.ConditionalError(err, "could not read from key file")
			file.Close()

			// Frontend User
			myOnion := db.Onion{Onion: *onion}
			dbconn.FirstOrCreate(&myOnion, myOnion)
			user := db.User{Username: *nickname, Onion: myOnion}
			user.SetPassword(*password)
			user.PemKey = string(pemKeyBuf[0:n])
			dbconn.Save(&user)
			myContact := db.Contact{}
			dbconn.Where(db.Contact{OnionId: myOnion.Id}).Attrs(db.Contact{Nickname: *nickname}).FirstOrCreate(&myContact)
			fmt.Println("added self user with contact: ")
			dbconn.Model(&myContact).Related(&myContact.Onion)
			fmt.Println(myContact)

		case "list-posts":
			posts := []db.Post{}
			dbconn.Find(&posts)
			for _, post := range posts {
				dbconn.Model(&post).Related(&post.Originator, "OriginatorId")
				dbconn.Model(&post).Related(&post.Author, "AuthorId")
				circs := []db.Circle{}
				dbconn.Model(&post).Association("Circles").Find(&circs)
				fmt.Printf("post from unix time %d:\n", post.PostedAt.Unix())
				fmt.Printf("  ID:         %d\n", post.Id)
				fmt.Printf("  Message:    %s\n", post.Message)
				fmt.Printf("  Author:     %s\n", post.Author.Onion)
				fmt.Printf("  Origin:     %s\n", post.Originator.Onion)
				fmt.Printf("  Timestamp:  %s\n", post.PostedAt)
				fmt.Printf("  Hash:       %s\n", post.Hash)
				fmt.Printf("  ParentId:   %d\n", post.ParentId)
				fmt.Printf("  Circles: ")
				for _, c := range circs {
					fmt.Printf("%s ", c.Name)
				}
				fmt.Printf("\n")
			}

		case "list-comments":
			posts := []db.Post{}
			dbconn.Where("parent_id=?", *id).Find(&posts)
			for _, post := range posts {
				dbconn.Model(&post).Related(&post.Originator, "OriginatorId")
				dbconn.Model(&post).Related(&post.Author, "AuthorId")
				circs := []db.Circle{}
				dbconn.Model(&post).Association("Circles").Find(&circs)
				fmt.Printf("post from unix time %d:\n", post.PostedAt.Unix())
				fmt.Printf("  ID:         %d\n", post.Id)
				fmt.Printf("  Message:    %s\n", post.Message)
				fmt.Printf("  Author:     %s\n", post.Author.Onion)
				fmt.Printf("  Origin:     %s\n", post.Originator.Onion)
				fmt.Printf("  Timestamp:  %s\n", post.PostedAt)
				fmt.Printf("  Hash:       %s\n", post.Hash)
				fmt.Printf("  ParentId:   %d\n", post.ParentId)
				fmt.Printf("  Circles: ")
				for _, c := range circs {
					fmt.Printf("%s ", c.Name)
				}
				fmt.Printf("\n")
			}

		case "post":
			logger.AssertError(len(*circname) > 0, "please provide one or more circle names separated by comma")
			logger.AssertError(len(*post) > 0, "please provide a message for the post")
			circs := strings.Split(*circname, ",")
			selfonion := dbconn.GetSelfOnion()
			post := db.Post{
				Message:      *post,
				TTL:          99,
				AuthorId:     selfonion.Id,
				OriginatorId: selfonion.Id,
				PostedAt:     time.Now(),
				PublishedAt:  time.Now(),
				ParentId:     *id,
				Published:    true,
			}
			for _, c := range circs {
				circle := db.Circle{Name: c}
				dbconn.Find(&circle, circle)
				if circle.Id == 0 {
					log.Fatalf("circle \"%s\" does not exist.", circle.Name)
				}
			}
			fmt.Printf("adding post to circles: ")
			dbconn.AddOrUpdatePost(&post)

			for _, c := range circs {
				circle := db.Circle{Name: c}
				dbconn.Find(&circle, circle)
				dbconn.Model(&circle).Association("Posts").Append(&post)
				fmt.Printf(".")
			}
			fmt.Printf("\n")

		case "comment":
			logger.AssertError(*id > 0, "please provide a parent id")
			logger.AssertError(len(*post) > 0, "please provide a message for the comment")
			var parentPost db.Post
			dbconn.Find(&parentPost, db.Post{Id: *id})
			if parentPost.Id == 0 {
				log.Fatalf("post with id %d does not exist.", *id)
			}
			selfonion := dbconn.GetSelfOnion()
			post := db.Post{
				Message:      *post,
				TTL:          99, // TODO
				AuthorId:     selfonion.Id,
				Author:       selfonion,
				OriginatorId: parentPost.OriginatorId,
				PostedAt:     time.Now(),
				PublishedAt:  time.Now(),
				ParentId:     *id,
				Published:    true,
			}
			post.CalcHash()
			dbconn.Save(&post)
			dbconn.RedirectComment(&post)

		case "add-circle":
			logger.AssertError(len(*circname) > 0, "please provide a circle name")
			circle := db.Circle{Name: *circname, Creator: db.CREATOR_APP}
			if *creator == 1 {
				circle.Creator = db.CREATOR_USER
			}
			dbconn.Find(&circle, circle)
			if circle.Id != 0 {
				log.Fatalf("circle \"%s\" already exists\n", *circname)
			}
			fmt.Printf("adding circle \"%s\"\n", *circname)
			dbconn.Create(&circle)

		case "delete-circle":
			logger.AssertError(len(*circname) > 0, "please provide a circle name")
			circle := db.Circle{Name: *circname}
			dbconn.Find(&circle, circle)
			if circle.Id == 0 {
				log.Fatalf("circle \"%s\" does not exist\n", *circname)
			}
			dbconn.Model(&circle).Association("Contacts").Clear()
			dbconn.Model(&circle).Association("Posts").Clear()
			dbconn.Delete(&circle)

		case "list-circles":
			circles := []db.Circle{}
			dbconn.Find(&circles)
			for _, c := range circles {
				fmt.Printf("circle: %s\n", c.Name)
				dbconn.Model(&c).Related(&c.Contacts, "Contacts")
				for _, u := range c.Contacts {
					fmt.Printf("  %s (%s)\n", u.Alias, u.Nickname)
				}
			}

		case "add-to-circle":
			logger.AssertError(len(*circname) > 0, "please provide a circle name")
			logger.AssertError(len(*onion) > 0, "please provide a valid onion address")
			circle := db.Circle{Name: *circname}
			dbconn.Find(&circle, circle)
			if circle.Id == 0 {
				log.Fatalf("Circle \"%s\" does not exist, exiting.\n", *circname)
			}
			if circle.Name == "Public" {
				log.Fatalf("You must not add users to special circle \"Public\"\n")
			}
			user := dbconn.GetContactByOnion(*onion)
			if user == nil {
				log.Fatalf("User with onion-id %s does not exist!\n", *onion)
			}
			fmt.Printf("adding user %s (%s) to circle \"%s\"\n", user.Alias, user.Nickname, circle.Name)
			dbconn.Model(&circle).Association("Contacts").Append(*user)

		case "key2onion":
			logger.AssertError(len(*keyfile) > 0, "please provide a valid onion address")
			key := crypto.ReadKey(*keyfile)
			onion := crypto.GetOnionAddress(&key.PublicKey)
			fmt.Println(onion)

		case "auth":
			logger.AssertError(len(*onion) > 0, "please provide a valid onion address")
			logger.AssertError(conn.Established, "no connection established, use command \"conn\"")

			privKey := dbconn.GetKey()
			if err := conn.Auth(privKey); err != nil {
				fmt.Printf("error with auth command: %s\n", err.Error())
				return
			}
			fmt.Printf("successfully authenticated to %s\n", *onion)

			contact := dbconn.GetContactByOnion(*onion)
			logger.AssertError(contact != nil, "Contact doesn't exists!")
			if contact.Status == db.PENDING {
				contact.Status = db.SUCCESS
				dbconn.Save(contact)
			}

		case "pull":
			logger.AssertError(conn.Established, "no connection established, use command \"conn\"")
			logger.AssertError(len(*onion) > 0, "please provide a valid onion address")

			posts, profiles, err := conn.Pull(*timestamp)
			if err != nil {
				fmt.Println("error while receiving posts: " + err.Error())
				return
			}

			for _, post := range posts {
				fmt.Println("------received publication------")
				fmt.Printf("Message: %s\n", post.Message)
				fmt.Printf("Author:  %s\n", post.Author.Onion)
				fmt.Printf("Origin:  %s\n", post.Originator.Onion)
			}

			for _, profile := range profiles {
				fmt.Println("------received profile------")
				fmt.Printf("Onion:  %s\n", profile.Onion.Onion)
				fmt.Printf("Key: %s\n", profile.Key)
				fmt.Printf("Val:  %s\n", profile.Value)
			}

		case "set-status":
			logger.AssertError(len(*onion) > 0, "please provide a valid onion address")
			logger.AssertError(len(*status) > 0, "please provide a valid contact status")
			contact := dbconn.GetContactByOnion(*onion)
			logger.AssertError(contact != nil, "Contact does not exists!")
			switch *status {
			case "blocked":
				contact.Status = db.BLOCKED
			case "open":
				contact.Status = db.OPEN
			case "pending":
				contact.Status = db.PENDING
			case "success":
				contact.Status = db.SUCCESS
			default:
				logger.Error("Unknown status : " + *status)
			}
			dbconn.Save(contact)

		case "add-profile":
			logger.AssertError(len(*circname) > 0, "please provide a circle name")
			logger.AssertError(len(*key) > 0, "please provide a key")
			logger.AssertError(len(*value) > 0, "please provide a value")
			circs := strings.Split(*circname, ",")
			prof := db.Profile{
				Id:        0,
				Key:       *key,
				Value:     *value,
				ChangedAt: time.Now(),
				Onion:     dbconn.GetSelfOnion(),
			}

			for _, c := range circs {
				circle := db.Circle{Name: c}
				dbconn.Find(&circle, circle)
				if circle.Id == 0 {
					log.Fatalf("circle \"%s\" does not exist.", circle.Name)
				}
			}

			fmt.Printf("adding profile to circles: ")
			dbconn.AddOrUpdateProfile(&prof)

			for _, c := range circs {
				circle := db.Circle{Name: c}
				dbconn.Find(&circle, circle)
				dbconn.Model(&circle).Association("Profiles").Append(&prof)
				fmt.Printf(".")
			}
			fmt.Printf("\n")

		case "list-profiles":
			profs := []db.Profile{}
			dbconn.Find(&profs)
			for _, prof := range profs {
				dbconn.Model(&prof).Related(&prof.Onion, "OnionId")
				circs := []db.Circle{}
				dbconn.Model(&prof).Association("Circles").Find(&circs)
				fmt.Printf("profile from unix time %d:\n", prof.ChangedAt.Unix())
				fmt.Printf("  ID:     %d\n", prof.Id)
				fmt.Printf("  Key:    %s\n", prof.Key)
				fmt.Printf("  Value:  %s\n", prof.Value)
				fmt.Printf("  Onion:  %s\n", prof.Onion.Onion)
				fmt.Printf("  Circles: ")
				for _, c := range circs {
					fmt.Printf("%s ", c.Name)
				}
				fmt.Printf("\n")
			}

		case "publish":
			logger.AssertError(*id > 0, "please provide a post id")
			post := db.Post{}
			dbconn.Find(&post, *id)
			if post.Published {
				logger.Info("Post already published.")
			} else {
				post.Published = true
				post.PublishedAt = time.Now()
				dbconn.Save(&post)
				logger.Info("Post published.")
			}

		default:
			logger.Error("Unknown command!")

		}
	}
}
