#
# Copyright (c) 2014
#   Dario Brandes
#   Thies Johannsen
#   Paul Kröger
#   Sergej Mann
#   Roman Naumann
#   Sebastian Thobe
# All rights reserved.
#
# Redistribution and use in source and binary forms, with or without modification,
# are permitted provided that the following conditions are met:
#
# 1. Redistributions of source code must retain the above copyright notice, this
#    list of conditions and the following disclaimer.
#
# 2. Redistributions in binary form must reproduce the above copyright notice,
#    this list of conditions and the following disclaimer in the documentation
#    and/or other materials provided with the distribution.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
# ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
# WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
# DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
# ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
# (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
# LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
# ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
# (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
# SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
#

all:	install

build.syncerd:	
	go build -o ../sync/syncerd ../sync/syncerd.go
build.server:	
	go build -o ../web/server ../web/server.go
build.testclient:
	go build -o ../test/client_main ../test/client_main.go
build.wizard:	
	go build wizard.go

update.repo:
	git stash
	git checkout master
	git pull

update:	update.repo build.syncerd build.server build.testclient build.wizard

install:	update
	sudo install -m755 zwiebelnetz /etc/init.d/zwiebelnetz
	cd /etc/init.d
	sudo update-rc.d zwiebelnetz defaults
