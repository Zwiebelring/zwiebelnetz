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
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

type userdata struct {
	name, pw string
}

var data = make(chan userdata, 1)

var db_path = "/.ssn/ssn.db"
var ssnUser_path = "/.ssn/ssn_user.txt"

func ssnUserAddAndCreate(name string) {
	if _, err := os.Stat(userHome() + "/.ssn"); os.IsNotExist(err) != false {
		os.Mkdir(userHome()+"/.ssn", 0700)
	}
	f, err := os.Create(userHome() + ssnUser_path)
	if err != nil {
		log.Fatal("error while creating ssn_user.txt: ", err)
	}
	defer f.Close()
	_, err = f.WriteString(name + "\n")
	if err != nil {
		log.Fatal("error while write on ssn_user.txt: ", err)
	}
}

func ssnUserExist() bool {
	if _, err := os.Stat(userHome() + ssnUser_path); os.IsNotExist(err) == false {
		log.Println("SSN users found")
		return true
	} else {
		log.Println("No SSN user on system")
		return false
	}
}

func ssnUserRead() string {
	f, err := os.Open(userHome() + ssnUser_path)
	if err != nil {
		log.Fatal("error opening ssn_user.txt: ", err)
	}
	r := bufio.NewReader(f)
	line, _, err := r.ReadLine()
	if err != nil {
		log.Fatal("error reading ssn_user.txt: ", err)
	}
	name := string(line)
	return name
}

func userHome() string {
	actual_user, _ := user.Current()
	return actual_user.HomeDir
}

func runSyncerd(name string) {
	syncerd := exec.Command("/home/pi/zwiebelnetz/wizard/start_syncerd.sh", name)
	syncerd.Stdout = os.Stdout
	syncerd.Stderr = os.Stderr
	err := syncerd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func runServer(name string) {
	server := exec.Command("/home/pi/zwiebelnetz/wizard/start_server.sh", name)
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr
	err := server.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func initUser(data userdata) {

	init_script := exec.Command("/home/pi/zwiebelnetz/wizard/init_user.sh", data.name, data.pw)
	init_script.Stdout = os.Stdout
	init_script.Stderr = os.Stderr
	err := init_script.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func expandFilesystem() {
	script := exec.Command("/home/pi/zwiebelnetz/wizard/expand_fs.sh")
	script.Stdout = os.Stdout
	script.Stderr = os.Stderr
	err := script.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func changeHostname(name string) {
	script := exec.Command("/home/pi/zwiebelnetz/wizard/change_hostname.sh", "zwiebel-"+name)
	script.Stdout = os.Stdout
	script.Stderr = os.Stderr
	err := script.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func getIpAddress() string {
	out, err := exec.Command("ifconfig", "eth0").Output()
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.SplitAfter(string(out), "\n")
	addresses := strings.SplitAfter(lines[1], ":")
	address := strings.SplitAfter(addresses[1], " ")
	return strings.TrimSpace(address[0])
}

func handler(w http.ResponseWriter, req *http.Request) {
	name := req.FormValue("name")
	pw := req.FormValue("pass")

	ipAddress := getIpAddress()
	waitPage, _ := ioutil.ReadFile("websrc/wait.html")
	waitPageString := string(waitPage)
	waitPageString = strings.Replace(waitPageString, "++username++", name, -1)
	waitPageString = strings.Replace(waitPageString, "++ipAddress++", ipAddress, -1)
	fmt.Fprintln(w, string(waitPageString))

	data <- userdata{name, pw}
}

func webserver(data chan userdata, done chan int) {

	http.Handle("/", http.FileServer(http.Dir("/home/pi/zwiebelnetz/wizard/websrc/")))
	http.HandleFunc("/script/", handler)

	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("Error listening: ", err)
	}
	<-done
}

func main() {
	// initialized syslog logging
	syslogger, err := syslog.New(syslog.LOG_INFO, "ssn-wizard")
	if err != nil {
		log.Fatalln("could not get connection to syslog")
	}
	log.SetOutput(syslogger)

	name := ""

	if ssnUserExist() == false {
		//run webserver
		done := make(chan int, 1)

		log.Println("Start Configuration Site")
		go webserver(data, done)

		log.Println("Waiting for Serverinput")

		//Verarbeite userdata, lege profil an

		user := <-data
		fmt.Println(user.name)
		
		initUser(user)
		ssnUserAddAndCreate(user.name)
		log.Println("ready creating profile")

		// change hostname
		log.Println("change hostname")
		changeHostname(user.name)

		// expand filesystem
		log.Println("expanding filesystem and reboot")
		expandFilesystem()

		//name = user.name

		//Beende Webserver
		//done <- 1

	} else {
		name = ssnUserRead()
		//start services in loop
		log.Println("Time for starting ssn services")
		log.Println(name)
		runSyncerd(name)
		runServer(name)
	}
}
