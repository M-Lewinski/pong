package main

import (
	"github.com/gorilla/websocket"
		"log"
	"sync"
	"github.com/mitchellh/mapstructure"
	"encoding/json"
	"errors"
)

const (
	MsgSession = "Session"
	MsgCreatePlayer = "Player"
	MsgJoinRoom = "JoinRoom"
	MsgError = "ErrorMsg"
)

type Room struct {
	Players []*Player
	NumberOfPlayers int
	MaxPlayers int
	Game *Game
	Playing bool
	RoomMutex *sync.Mutex
	Id string
}

type ClientSession struct {
	Socket *websocket.Conn
	Player *Player
	WriteMutex *sync.Mutex
}

type ServerState struct {
	Rooms map[string]*Room
	Players map[string]*Player
	Mutex *sync.Mutex
}

type CommandParams map[string]interface{}

//type Message map[string]*json.RawMessage

type Message struct {
	Message string
	Data CommandParams
}

type ErrorMessage struct {
	ErrorMsg string
}


func (client *ClientSession) ManageSession(){
	for{
		msgType, msg, err := client.Socket.ReadMessage()
		if err != nil {
			log.Println(err)
		}
		if (msgType == websocket.TextMessage){
			log.Println(string(msg))
			msgObject := Message{}
			err = json.Unmarshal(msg,&msgObject)
			if err != nil {
				client.SendError(err)
				continue
			}
			if msgObject.Message == "" {
				client.SendError(errors.New("Message field is missing"))
				continue
			}
			client.HandleMessage(msgObject)
		} else if (msgType == websocket.CloseMessage || msgType == -1){
			log.Println("Closing client session")
			if (client.Player != nil){
				client.Player.Disconnect()
			}
			return
		} else {
			log.Println(string(msg))
			log.Println(msgType)
			log.Println("Got something else")
		}
	}
}

func (client *ClientSession) HandleMessage(message Message){
	log.Println("Received Message: " + message.Message)
	var err error;
	err = nil
	switch message.Message {
	case MsgSession:
		err = client.RestoreSession(message)
	case MsgCreatePlayer:
		err = client.CreatePlayer(message)
	}
	if err != nil {
		client.SendError(err)
	}
}

func (client *ClientSession) RestoreSession(message Message) error{
	cookie := CookieSession{}
	err := mapstructure.Decode(message.Data,&cookie)
	if err != nil{
		return err
	}
	player := cookie.CheckCookie()
	client.Player = player
	if player != nil {
		client.SendMessage(MsgCreatePlayer,player)
	}
	return nil
}

func (client *ClientSession) CreatePlayer(message Message) error {
	if client.Player != nil {
		err := errors.New("Player already exists for this session")
		return err
	}
	msgPlayer := PlayerCreation{}
	err := mapstructure.Decode(message.Data,&msgPlayer)
	if err != nil{
		return err
	}
	createdPlayer,err := msgPlayer.CreateNewPlayer()
	if err != nil {
		return err
	}
	client.SendMessage(MsgCreatePlayer,createdPlayer)
	client.Player = createdPlayer
	return nil
}

func (serv *ServerState) FindRomm(roomId string) *Room  {
	serv.Mutex.Lock()
	defer serv.Mutex.Unlock()
	if room , ok := serv.Rooms[roomId]; ok {
		return room
	}
	return nil
}

func (serv *ServerState) FindPlayer(playerId string) *Player {
	serv.Mutex.Lock()
	defer serv.Mutex.Unlock()
	if play , ok := serv.Players[playerId]; ok {
		return play
	}
	return nil
}


func (client *ClientSession) SendRoom(message Message){

}

func (client *ClientSession) CreateRoom(message Message){

}

func (client *ClientSession) SendMessage(msgText string ,object interface{}) error {
	client.WriteMutex.Lock()
	defer client.WriteMutex.Unlock()
	var data = map[string]interface{}{}
	mar, _ := json.Marshal(object)
	json.Unmarshal(mar, &data)
	message := Message{}
	message.Message = msgText
	message.Data = data
	err := client.Socket.WriteJSON(message)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (client *ClientSession) SendError(msg error){
	log.Println(msg)
	err := ErrorMessage{}
	err.ErrorMsg = msg.Error()
	client.SendMessage(MsgError, err)
}
