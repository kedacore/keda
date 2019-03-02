/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/knative/serving/test"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{
	// Allow any origin, since we are spoofing requests anyway.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading websocket:", err)
		return
	}
	defer conn.Close()
	log.Println("Connection upgraded to WebSocket. Entering receive loop.")
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			// We close abnormally, because we're just closing the connection in the client,
			// which is okay. There's no value delaying closure of the connection unnecessarily.
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				log.Println("Client disconnected.")
			} else {
				log.Println("Handler exiting on error:", err)
			}
			return
		}
		log.Printf("Successfully received: %q", message)
		if err = conn.WriteMessage(messageType, message); err != nil {
			log.Println("Failed to write message:", err)
			return
		}
		log.Printf("Successfully wrote: %q", message)
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	test.ListenAndServeGracefully(*addr, handler)
}
