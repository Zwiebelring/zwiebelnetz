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

// Todo: Use os/user-type for the names, and check correctness

package main

import (
    "fmt"
    "os"
    "os/user"
	"os/exec"
	"log"
	"net/http"
)

var db_path = "/.ssn/ssn.db"

func dbExist() bool {
    if _, err := os.Stat(userHome()+db_path); os.IsNotExist(err) {
        return true
    }else{
        return false
    }
}

func userHome() string {
    actual_user, _ := user.Current()
    return actual_user.HomeDir
}

func runSyncerd(name string) {
	for {
		syncerd := exec.Command("/home/pi/secsocnet/wizard/start_syncerd.sh", name)
		syncerd.Stdout = os.Stdout
		syncerd.Stderr = os.Stderr
		err := syncerd.Run()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("syncerd closed")
	}
}

func runServer(name string) {
	for {
		server := exec.Command("/home/pi/secsocnet/wizard/start_server.sh", name)
		server.Stdout = os.Stdout
		server.Stderr = os.Stderr
		err := server.Run()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("server closed")
	}
}

func wizardPage() {
	log.Fatal(http.ListenAndServe(":8081", http.FileServer(http.Dir("/home/pi/secsocnet/wizard/web/"))))
}

func main() {
	name := "harry"
	pw := "qwer"	

    if dbExist(){
	    //start secsocnet                                                                                                                                                
        fmt.Println("SecSocNet-Acound found")
    }else{
		//run webserver
		go wizardPage()


	    fmt.Println("run init script")

		init_script := exec.Command("/home/pi/secsocnet/wizard/init_user.sh", name, pw)
		init_script.Stdout = os.Stdout
		init_script.Stderr = os.Stderr
		err := init_script.Run()
		if err != nil {
			log.Fatal(err)
		}	
		fmt.Println("ready")
    }

	//start services in loop
	go runSyncerd(name)
	runServer(name)
}

