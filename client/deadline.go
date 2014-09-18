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
	"crypto/rsa"
	"time"
)

type CallbackFunc func(key *rsa.PrivateKey)

type Deadline struct {
	ticker     *time.Ticker
	key        rsa.PrivateKey
	interval   time.Duration
	callback   CallbackFunc
	executions uint64
}

func NewDeadline(key *rsa.PrivateKey, interval time.Duration, callback CallbackFunc, executions uint64) Deadline {
	callback(key)
	deadline := Deadline{}
	deadline.key = *key
	deadline.interval = interval
	deadline.callback = callback
	deadline.executions = executions
	return deadline
}

func (this *Deadline) Start() {
	go this.exec()
}

func (this *Deadline) Stop() {
	this.ticker.Stop()
}

func (this *Deadline) exec() {
	constantly := 0 == this.executions
	this.ticker = time.NewTicker(this.interval)
	tickerChan := this.ticker.C
	for {
		select {
		case <-tickerChan:
			this.callback(&this.key)
			if !constantly && 0 == this.executions {
				return
			} else {
				this.executions--
			}
		}
	}
}
