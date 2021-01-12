// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/websocket"

	// "github.com/smartcontractkit/libocr/offchainreporting/serialization"
	"github.com/smartcontractkit/libocr/offchainreporting/internal/serialization/protobuf"
	// "github.com/smartcontractkit/libocr/offchainreporting/types"
	"google.golang.org/protobuf/proto"
)

var accessKey = "szmVSrHTZ0QjleU4"
var secret = "HKoVa3iYEMLG15SHe6jyUCmycRSdaXF5t9RLDLt0EjOT8iRP3unyJc5mHZrFSA0v"
var addr = flag.String("addr", "localhost:8081", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	authHeader := http.Header{}
	authHeader.Add("X-Explore-Chainlink-Accesskey", accessKey)
	authHeader.Add("X-Explore-Chainlink-Secret", secret)

	// c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	c, _, err := websocket.DefaultDialer.Dial(u.String(), authHeader)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			pbm := &protobuf.TelemetryWrapper{
				Wrapped: &protobuf.TelemetryWrapper_RoundStarted{&protobuf.TelemetryRoundStarted{
					ConfigDigest: hexutil.MustDecode("0x11223344556677889900aabbccddeeff"),
					Epoch:        3,
					Round:        17,
					Leader:       7,
					Time:         uint64(time.Date(2018, 07, 03, 12, 11, 9, 4, time.UTC).UnixNano()),
				}},
			}
			b, err := proto.Marshal(pbm)
			// 42,34,10,16,17,34,51,68,85,102,119,239,191,189,239,191,189,0,239,191,189,239,191,189,239,191,189,239,191,189,239,191,189,239,191,189,16,3,24,17,32,7,40,239,191,189,196,153,207,148,239,191,189,239,191,189,239,191,189,21
			if err != nil {
				panic(fmt.Errorf("could not marshal protobuf message: %v", err))
			}

			// err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			fmt.Printf("writing round started to web socket - bytes: %v\n", b)
			c.EnableWriteCompression(false)
			// fmt.Printf("writing round started to web socket - string: %v\n", string(b))
			// err = c.WriteMessage(websocket.TextMessage, b)
			err = c.WriteMessage(websocket.BinaryMessage, b)
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
