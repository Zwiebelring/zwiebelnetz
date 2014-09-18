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

package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

type Logs struct {
	Debug    *log.Logger
	Info     *log.Logger
	Security *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
}

var logger Logs

//init

func Init(debug io.Writer, info io.Writer, sec io.Writer, warn io.Writer, err io.Writer) {

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix("LOG: ")

	logger.Debug = log.New(debug,
		"DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	logger.Info = log.New(info,
		"INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	logger.Security = log.New(sec,
		"SECURITY: ", log.Ldate|log.Ltime|log.Lshortfile)

	logger.Warning = log.New(warn,
		"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)

	logger.Error = log.New(err,
		"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

//calls

func Debug(msg string) {
	logger.Debug.Output(2, msg)
}

func Info(msg string) {
	logger.Info.Output(2, msg)
}

func Security(msg string) {
	logger.Security.Output(2, msg)
}

func Warning(msg string) {
	logger.Warning.Output(2, msg)
}

func ConditionalWarning(err error, msg string) bool {
	if err != nil {
		logger.Warning.Output(2, fmt.Sprintf("%s, Err: %s", msg, err))
		return true
	}
	return false
}

func Error(msg string) {
	logger.Error.Output(2, msg)
	os.Exit(1)
}

func AssertError(condition bool, msg string) {
	if !condition {
		logger.Error.Output(2, fmt.Sprintf("%s", msg))
		os.Exit(1)
	}
}

func ConditionalError(err error, msg string) {
	if err != nil {
		logger.Error.Output(2, fmt.Sprintf("%s, Err: %s", msg, err))
		os.Exit(1)
	}
}

func ConditionalDebug(err error, debug_msg string) {
	if err != nil {
		logger.Error.Output(2, fmt.Sprintf("%s, Debug Message: %s", debug_msg, err))
		os.Exit(1)
	}
}
