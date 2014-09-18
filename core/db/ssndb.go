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
	"../../logger"
	"../crypto"
	"container/list"
	"crypto/rsa"
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"regexp"
	"time"
)

type SSNDB struct {
	// gorm db connection
	gorm.DB

	// prepared statements:
	getPostsStmt            *sql.Stmt // prepared statemtn to get all post for a given user newer than a specified timestamp
	getCommentsStmt         *sql.Stmt
	getProfileStmt          *sql.Stmt // prepared statemtn to get all profile (key, value) pairs for a given user newer than a specified timestamp
	getPublicPostsStmt      *sql.Stmt // get all public posts
	getPublicProfileStmt    *sql.Stmt // get public profile (key, value) pairs
	getContactsLastActivity *sql.Stmt
	// getContactByOnionStmt     *sql.Stmt // get contact by onion address
}

/* writes PEMKey into the database for the main user */
func (this *SSNDB) InitPEMKey(pemKey string) {
	user := this.GetUser()
	user.PemKey = pemKey
	this.Save(user)
}

/* gets the main db user */
func (this *SSNDB) GetUser() User {
	var user User
	this.First(&user)
	logger.AssertError(user.Id != 0, "main user does not exist in db!")
	return user
}

func (this *SSNDB) GetKey() *rsa.PrivateKey {
	user := this.GetUser()
	key := crypto.Pem2Key([]byte(user.PemKey))
	logger.AssertError(key != nil, "CRITICAL: Could not get user Key!")
	return key
}

func GetDBName() string {
	homeDir := os.Getenv("HOME")
	if len(homeDir) == 0 {
		logger.Error("$HOME unset!")
	}

	dbpath := homeDir + "/.ssn/"

	_, err := os.Stat(dbpath)
	if os.IsNotExist(err) {
		os.Mkdir(dbpath, 0700)
	}

	dbname := "ssn.db"
	return dbpath + dbname
}

func (this *SSNDB) Init() {
	var err error
	dbname := GetDBName()
	this.DB, err = gorm.Open("sqlite3", dbname)
	logger.ConditionalError(err, "Could not open database '"+dbname+"'")

	//this.DB.LogMode(true) // enable for debugging
	this.Exec("PRAGMA foreign_keys = ON;")

	this.AutoMigrate(Onion{})
	this.AutoMigrate(Contact{})
	this.AutoMigrate(Post{})
	this.AutoMigrate(Circle{})
	this.AutoMigrate(User{})
	this.AutoMigrate(Profile{})
	this.AutoMigrate(Pending{})

	var p Pending
	this.Find(&p, 1)
	if p.Id == 0 {
		p.Posts = false
		p.Contacts = false
		this.Save(&p)
	}

	// add public circle
	pubCircle := Circle{Name: "Public"}
	this.Find(&pubCircle, pubCircle)
	if pubCircle.Id == 0 {
		logger.Info("circle \"Public\" does not exist, creating circle now")
		this.Create(&pubCircle)
	}

	// Prepared Statements
	this.view()
	this.prepare()
}

func (this *SSNDB) prepare() {
	var err error
	postColumns := "P.id, P.message, P.created_at, P.updated_at, P.deleted_at, P.t_t_l-1, P.published, " +
		"P.originator_id, P.author_id, P.posted_at, P.published_at, P.remote_published_at, P.hash, P.parent_id "
	this.getPostsStmt, err = this.DB.DB().Prepare(
		"SELECT DISTINCT " + postColumns +
			"FROM posts AS P JOIN circle_posts ON P.id = circle_posts.post_id " +
			"JOIN circles ON circle_posts.circle_id = circles.id              " +
			"JOIN circle_contacts ON circle_contacts.circle_id = circles.id   " +
			"WHERE circle_contacts.contact_id = ? AND P.published_at > ? AND p.t_t_l > 0 AND P.published = 1 " +
			"AND p.deleted_at = datetime('0001-01-01 00:00:00') ORDER BY P.id")

	logger.ConditionalError(err, "Failed to create prepared statement")

	// this statement selects all comments which were written by us and which refer to a post by originator
	this.getCommentsStmt, err = this.DB.DB().Prepare(
		"SELECT " + postColumns +
			"FROM posts AS P " +
			"WHERE P.originator_id = ? AND P.author_id = ? AND p.parent_id != 0 AND p.published_at > ? AND P.published = 1 " +
			"AND p.deleted_at = datetime('0001-01-01 00:00:00') AND P.t_t_l > 0 ORDER BY P.id")

	logger.ConditionalError(err, "Failed to create prepared statement")

	this.getProfileStmt, err = this.DB.DB().Prepare(
		"SELECT DISTINCT P.id, P.key, P.value, P.changed_at, P.onion_id " +
			"FROM profiles AS P JOIN circle_profiles ON P.id = circle_profiles.profile_id " +
			"JOIN circles ON circle_profiles.circle_id = circles.id              " +
			"JOIN circle_contacts ON circle_contacts.circle_id = circles.id   " +
			"WHERE circle_contacts.contact_id = ? AND P.onion_id = ?  AND p.deleted_at = datetime('0001-01-01 00:00:00') ORDER BY P.id ")

	logger.ConditionalError(err, "Failed to create prepared statement")

	this.getPublicPostsStmt, err = this.DB.DB().Prepare(
		"SELECT DISTINCT " + postColumns +
			"FROM posts AS P JOIN circle_posts ON P.id = circle_posts.post_id " +
			"JOIN circles ON circle_posts.circle_id = circles.id              " +
			"WHERE circles.name = 'Public' AND P.published_at > ? AND P.published = 1 AND p.deleted_at = datetime('0001-01-01 00:00:00') ORDER BY P.id  ")

	this.getPublicProfileStmt, err = this.DB.DB().Prepare(
		"SELECT DISTINCT  P.id, P.key, P.value, P.changed_at, P.onion_id " +
			"FROM profiles AS P JOIN circle_profiles ON P.id = circle_profiles.profile_id " +
			"JOIN circles ON circle_profiles.circle_id = circles.id              " +
			"WHERE circles.name = 'Public' AND P.onion_id=? AND p.deleted_at = datetime('0001-01-01 00:00:00') ORDER BY P.id  ")

	logger.ConditionalError(err, "Failed to create prepared statement")

	this.getContactsLastActivity, err = this.DB.DB().Prepare(
		"SELECT MAX(" +
			"(SELECT ifnull(" +
			"(SELECT strftime('%s', MAX(remote_published_at)) FROM posts WHERE originator_id = ?)" +
			",0))" +
			",(SELECT ifnull(" +
			"(SELECT strftime('%s', MAX(changed_at)) FROM profiles WHERE onion_id=?)" +
			",0)));")

	logger.ConditionalError(err, "Failed to create prepared statement")
}

func (this *SSNDB) view() {
	this.DB.Exec("CREATE VIEW IF NOT EXISTS persons AS " +
		"SELECT ifnull(O.id, 0) AS onion_id," +
		"ifnull(O.onion, '') AS onion_addr," +
		"ifnull(C.id, 0) AS contact_id," +
		"ifnull(C.nickname, '') AS nickname," +
		"ifnull(C.alias, '') AS alias," +
		"ifnull(C.trust, 0) AS trust," +
		"ifnull(C.status, 0) AS status," +
		"ifnull(C.request_message, '') AS request_message " +
		"FROM onions AS O LEFT JOIN contacts AS C ON O.id = C.onion_id;")
}

func (this *SSNDB) getOnionById(id int64) Onion {
	var addr Onion
	this.DB.Where(&Onion{Id: id}).First(&addr)
	return addr
}

func IsValidOnion(onion string) bool {
	b, _ := regexp.MatchString("^[a-z2-7]{16}\\.onion$", onion)
	return b
}

func (this *SSNDB) scanPosts(posts *list.List, rows *sql.Rows) {
	for rows.Next() {
		post := new(Post)

		rows.Scan(
			&post.Id,
			&post.Message,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.DeletedAt,
			&post.TTL,
			&post.Published,
			&post.OriginatorId,
			&post.AuthorId,
			&post.PostedAt,
			&post.PublishedAt,
			&post.RemotePublishedAt,
			&post.Hash,
			&post.ParentId)

		post.ParentHash, _ = this.GetPostHashById(post.ParentId)
		post.Originator = this.getOnionById(post.OriginatorId)
		post.Author = this.getOnionById(post.AuthorId)
		posts.PushBack(post)
	}
}

func (this *SSNDB) GetPosts(contact *Contact, timestamp int64) *list.List {
	posts := new(list.List)

	rows, err := this.getPublicPostsStmt.Query(time.Unix(timestamp+1, 0))
	if logger.ConditionalWarning(err, "Failed to execute pepared statement: getPublicPostsStmt") {
		return nil
	}

	this.scanPosts(posts, rows)
	rows.Close()

	if contact != nil {
		rows, err := this.getPostsStmt.Query(contact.Id, time.Unix(timestamp+1, 0))
		if logger.ConditionalWarning(err, "Failed to execute pepared statement: getPostsStmt") {
			return nil
		}

		this.scanPosts(posts, rows)
		rows.Close()

		selfOnion := this.GetSelfOnion()
		rows, err = this.getCommentsStmt.Query(contact.OnionId, selfOnion.Id, time.Unix(timestamp+1, 0))
		this.scanPosts(posts, rows)
		rows.Close()
	}

	return posts
}

func (this *SSNDB) scanProfile(profs *list.List, rows *sql.Rows) {
	for rows.Next() {
		prof := new(Profile)

		rows.Scan(
			&prof.Id,
			&prof.Key,
			&prof.Value,
			&prof.ChangedAt,
			&prof.OnionId)
		profs.PushBack(prof)
	}
}

func (this *SSNDB) GetProfiles(contact *Contact, timestamp int64) *list.List {
	profs := new(list.List)
	self := this.GetSelfOnion()

	rows, err := this.getPublicProfileStmt.Query(self.Id)
	if logger.ConditionalWarning(err, "Failed to execute pepared statement: getPublicProfileStmt") {
		return nil
	}

	this.scanProfile(profs, rows)
	rows.Close()

	if contact != nil {
		rows, err := this.getProfileStmt.Query(contact.Id, self.Id)
		if logger.ConditionalWarning(err, "Failed to execute pepared statement: getProfileStmt") {
			return nil
		}

		this.scanProfile(profs, rows)
		rows.Close()
	}

	for prof := profs.Front(); prof != nil; prof = prof.Next() {
		p := prof.Value.(*Profile)
		if p.ChangedAt.Unix() > timestamp {
			return profs
		}
	}

	return new(list.List)
}

func (this *SSNDB) GetSelf() Contact {
	var user User
	this.DB.First(&user)
	logger.AssertError(user.Id != 0, "No \"self\" user in database!")
	var contact Contact
	this.DB.Where(&contact, Contact{OnionId: user.OnionId})
	logger.AssertError(contact.Id != 0, "No \"self\" contact in database!")
	return contact
}
func (this *SSNDB) GetSelfOnion() Onion {
	var user User
	this.DB.First(&user)
	logger.AssertError(user.Id != 0, "No \"self\" user in database!")
	var onion Onion
	this.DB.First(&onion, user.OnionId)
	logger.AssertError(onion.Id != 0, "No \"self\" onion in database!")
	return onion
}

func (this *SSNDB) GetContactByOnion(onionstr string) *Contact {
	var onion Onion
	contact := new(Contact)

	this.Where(&Onion{Onion: onionstr}).First(&onion)
	if 0 == onion.Id {
		return nil
	}

	this.Where(&Contact{OnionId: onion.Id}).First(contact)
	if contact.Id == 0 {
		return nil
	}
	contact.Onion = onion
	return contact
}

// only gets you SUCCESSFUL contacts (most likely friends)
func (this *SSNDB) GetFriendlyContactByOnion(onionstr string) *Contact {
	var onion Onion
	contact := new(Contact)

	this.Where(&Onion{Onion: onionstr}).First(&onion)
	if 0 == onion.Id {
		logger.Security(fmt.Sprintf("unknown onion address: %s", onionstr))
		return nil
	}

	this.Where(&Contact{OnionId: onion.Id}).Where("Status >= ?", PENDING).First(contact)
	if contact.Id == 0 {
		logger.Security(fmt.Sprintf("can't find unblocked contact for onion address: %s", onionstr))
		return nil
	}
	contact.Onion = onion
	return contact
}

func (this *SSNDB) GetOnion(onion string) Onion {
	var addr Onion
	this.DB.Where(&Onion{Onion: onion}).First(&addr)
	return addr
}

func (this *SSNDB) GetOrCreateOnion(onion string) Onion {
	var addr Onion
	this.DB.Where(&Onion{Onion: onion}).First(&addr)
	if addr.Id == 0 {
		addr.Onion = onion
		this.DB.Save(&addr)
	}
	return addr
}

func (this *SSNDB) GetPostCircles(post *Post) []Circle {
	var circles []Circle
	this.Model(post).Related(&circles, "Circles")
	return circles
}

func (this *SSNDB) AddOrUpdatePost(post *Post) {
	var dbPost Post

	post.Originator = this.GetOnion(post.Originator.Onion)
	post.OriginatorId = post.Originator.Id

	post.Author = this.GetOrCreateOnion(post.Author.Onion)
	post.AuthorId = post.Author.Id
	post.CalcHash()

	this.DB.Where(&Post{Hash: post.Hash}).First(&dbPost)
	post.Id = dbPost.Id // wenn post bereits existiert wird upgedated (id != 0) sonst neu angelegt (id = 0)

	if post.ParentId == 0 && post.ParentHash != "" {
		parent, err := this.GetPostByHashUnscoped(post.ParentHash)
		if err != nil {
			logger.Warning(fmt.Sprintf("Can't find parent with hash: %s\n", post.ParentHash))
			return
		}
		post.ParentId = parent.Id
		post.DeletedAt = parent.DeletedAt
	}

	this.Save(post)

	if post.ParentId != 0 {
		this.RedirectComment(post)
	}
}

func (this *SSNDB) RedirectComment(post *Post) {
	self := this.GetSelfOnion()
	var parent Post
	this.Find(&parent, post.ParentId)
	if parent.AuthorId == self.Id {
		logger.Info("Redirecting comment ...")
		circles := this.GetPostCircles(&parent)
		for _, c := range circles {
			logger.Info(fmt.Sprintf("\t-> %s", c.Name))
			this.Model(&c).Association("Posts").Append(post)
		}
	}
}

func (this *SSNDB) AddOrUpdatePosts(posts []Post) {
	length := len(posts)
	for i := 0; i < length; i++ {
		this.AddOrUpdatePost(&posts[i])
	}

	if length > 0 {
		var p Pending
		this.Find(&p, 1)
		p.Posts = true
		this.Save(&p)
	}
}

func (this *SSNDB) AddOrUpdateProfile(prof *Profile) {
	var dbProf Profile

	prof.Onion = this.GetOnion(prof.Onion.Onion)
	prof.OnionId = prof.Onion.Id

	this.DB.Where(&Profile{Key: prof.Key, OnionId: prof.OnionId}).First(&dbProf)
	prof.Id = dbProf.Id // wenn prof bereits existiert wird upgedated (id != 0) sonst neu angelegt (id = 0)
	this.Save(prof)
}

func (this *SSNDB) AddOrUpdateProfiles(profs []Profile) {
	length := len(profs)
	// delete old profile entries => all profiles must have the same originator
	if length > 0 {
		onion := this.GetOnion(profs[0].Onion.Onion)
		this.Unscoped().Where(&Profile{OnionId: onion.Id}).Unscoped().Delete(Profile{})
	}

	for i := 0; i < length; i++ {
		this.AddOrUpdateProfile(&profs[i])
	}
}

func (this *SSNDB) GetContactsLastActivity(contact *Contact) int64 {
	rows, _ := this.getContactsLastActivity.Query(contact.OnionId, contact.OnionId)
	rows.Next()
	var t int64
	rows.Scan(&t)
	rows.Close()
	return t
}

func (this *SSNDB) GetPostByHash(hash string) (*Post, error) {
	post := Post{Hash: hash}
	err := this.Where(&Post{Hash: hash}).First(&post).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (this *SSNDB) GetPostByHashUnscoped(hash string) (*Post, error) {
	post := Post{Hash: hash}
	err := this.Unscoped().Where(&Post{Hash: hash}).First(&post).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (this *SSNDB) GetPostHashById(id int64) (string, error) {
	post := Post{}
	err := this.First(&post, id).Error
	if err != nil {
		return "", err
	}
	return post.Hash, nil
}

func (this *SSNDB) PostsPending() bool {
	var p Pending
	this.Find(&p, 1)
	if p.Posts {
		p.Posts = false
		this.Save(&p)
		return true
	}
	return false
}

func (this *SSNDB) ContactsPending() bool {
	var p Pending
	this.Find(&p, 1)
	if p.Contacts {
		p.Contacts = false
		this.Save(&p)
		return true
	}
	return false
}

func (this *SSNDB) GetProfilePictureId(onionId int64) int64 {
	var profilePicture Profile
	this.Where(&Profile{Key: "picture", OnionId: onionId}).First(&profilePicture)
	return profilePicture.Id
}
