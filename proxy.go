package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

const TARGET_PORT = ":8000"
const PROXY_PORT = ":3344"
const CHANNEL_SIZE = 0x1000
const BUFFER_SIZE = 0x1000

var upgrader = websocket.Upgrader{
	ReadBufferSize:  0x1000,
	WriteBufferSize: 0x1000,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

func ygoEndpoint(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup

	upgrader.CheckOrigin = wsChecker

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	log.Println("Websocket connected")

	tcp, err := net.Dial("tcp", "127.0.0.1"+PROXY_PORT)
	if err != nil {
		log.Fatal(err)
	}
	defer tcp.Close()

	wg.Add(2)
	go wsProxy(ws, &tcp, &wg)
	go tcpProxy(&tcp, ws, &wg)
	wg.Wait()
}

func wsProxy(ws *websocket.Conn, tcp *net.Conn, wg *sync.WaitGroup) {
	for {
		messageType, buf, err := ws.ReadMessage()
		if err != nil {
			log.Fatal(err)
			break
		}

		if messageType == websocket.CloseMessage {
			log.Println("Websocket closed")
			break
		}

		log.Println("websocket to tcp: " + string(buf))

		writer := bufio.NewWriter(*tcp)
		buffer := make([]byte, BUFFER_SIZE)

		_, err = writer.Write(buffer)
		if err != nil {
			log.Fatal(err)
			break
		}
	}

	wg.Done()
}

func tcpProxy(tcp *net.Conn, ws *websocket.Conn, wg *sync.WaitGroup) {
	for {
		reader := bufio.NewReader(*tcp)
		buffer := make([]byte, BUFFER_SIZE)

		_, err := reader.Read(buffer)
		if err != nil {
			log.Fatal(err)
			break
		}

		log.Println("tcp to websocket: " + string(buffer))

		err = ws.WriteMessage(websocket.TextMessage, buffer) // temporary TextMessage, should be BinaryMessage in ygopro
		if err != nil {
			log.Fatal(err)
			break
		}
	}

	wg.Done()
}

func wsChecker(r *http.Request) bool { return true }

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ygo", ygoEndpoint)
}

func main() {
	setupRoutes()

	log.Fatal(http.ListenAndServe(TARGET_PORT, nil))
}
