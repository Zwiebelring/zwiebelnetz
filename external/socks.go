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

package socks

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	_ "os"
)

type socket struct {
	conn *net.TCPConn
}

// Connect provides functionality to connect to a socks server
// socks response object. Byte order for resp is: 0x00(discard) 0xXX(status) 0xXX 0xXX(2 bytes to ignore) 0xXX 0xXX 0xXX 0xXX (4 bytes to ignore)
// socks status codes: 0x5a(90) == granted ; 0x5b(91) == rejected/failed ; 0x5c(92) == failed because missing identd ; 0x5d(93) == identd couldn't confirm identity from user ID
func Connect(conn *net.TCPConn, domain string) error {
	sock := new(socket)
	sock.conn = conn
	version := []byte{0x04} // socks version 4
	cmd := []byte{0x01}     // socks stream mode
	port := 3141            // destination http port
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, version)
	binary.Write(buffer, binary.BigEndian, cmd)
	binary.Write(buffer, binary.BigEndian, uint16(port))                   // pad port with 0x00
	binary.Write(buffer, binary.BigEndian, []byte{0x00, 0x00, 0x00, 0x01}) // fake ip address forces socks4a to resolve the domain below using the socks protocol
	binary.Write(buffer, binary.BigEndian, []byte{0x00})
	binary.Write(buffer, binary.BigEndian, []byte(domain))
	binary.Write(buffer, binary.BigEndian, []byte{0x00})
	binary.Write(sock.conn, binary.BigEndian, buffer.Bytes())
	return sock.read()
}

func (this *socket) read() error {
	data := make([]byte, 8) // socks responses are 8 bytes
	count, err := this.conn.Read(data)
	if err != nil {
		return errors.New("unable to read bytes from data stream.\n")
	} else if count == 0 {
		return errors.New("socks host closed connection.\n")
	} else if data[1] == 0x5a { // success
		return nil
	} else if data[1] == 0x5b { // request failed
		return errors.New("socks host could not connect to given onion")
	}
	return errors.New("socks host reports unknown error")
}
