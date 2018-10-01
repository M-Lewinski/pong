package main

import (
	"net/http"
	"github.com/gorilla/websocket"
		"log"
	"sync"
)

var	done = make(chan bool)
var Server = ServerState{
	Players: map[string]*Player{},
	Rooms: map[string]*Room{},
	Clients: make([]*ClientSession,0),
	MutexRooms: &sync.Mutex{},
	MutexPlayers: &sync.Mutex{},
	MutexClients: &sync.Mutex{},
}

func main() {
	http.HandleFunc("/",rootHandler)
	http.Handle("/assets/",http.StripPrefix("/assets",http.FileServer(http.Dir("assets"))))
	http.HandleFunc("/ws", upgradeHandler)
	log.Fatal(http.ListenAndServe(":8080",nil))
	<- done
}

func rootHandler(w http.ResponseWriter, request *http.Request){
	http.ServeFile(w,request,"assets/index.html")
}

func upgradeHandler(w http.ResponseWriter, request *http.Request){
	conn, err := websocket.Upgrade(w,request,w.Header(),1024,1024)
	if (err != nil){
		http.Error(w,"Couldn't open websocket connection", http.StatusBadRequest)
		log.Fatalln(err)
		return
	}
	go CreateSession(conn)
}

func CreateSession(conn *websocket.Conn){
	client := &ClientSession{}
	client.Socket = conn
	client.WriteMutex = &sync.Mutex{}
	client.WriteChannel = make(chan Message)
	client.Close = make(chan bool)
	Server.MutexClients.Lock()
	Server.Clients = append(Server.Clients, client)
	Server.MutexClients.Unlock()
	go client.ManageSession()
}
