package main

import (
	"github.com/gorilla/websocket"
		"log"
	"sync"
		"encoding/json"
	"errors"
	"github.com/mitchellh/mapstructure"
)

const (
	MsgSession      = "Session"
	MsgCreatePlayer = "Player"
	MsgCreateRoom   = "CreateRoom"
	MsgError        = "ErrorMsg"
	MsgRoomInfo     = "RoomInfo"
	MsgAllRooms     = "AllRooms"
	MsgJoinRoom     = "JoinRoom"
	MsgLeaveRoom     = "LeaveRoom"
	MsgReadyPlayer  = "ReadyPlayer"
	MsgDeleteRoom = "DeleteRoom"
)

type Room struct {
	Players []*Player
	NumberOfPlayers int
	Life int
	MaxPlayers int
	Playing bool
	Name string
	RoomMutex *sync.Mutex `json:"-"`
	Id string
	Ready []bool
	ReadyCount int
	wg *sync.WaitGroup `json:"-"`
	//Result []int
}

type ClientSession struct {
	Socket *websocket.Conn
	Player *Player
	WriteMutex *sync.Mutex
	WriteChannel chan Message
	Close chan bool
}

type ServerState struct {
	Rooms map[string]*Room
	Players map[string]*Player
	Clients []*ClientSession
	MutexRooms *sync.Mutex
	MutexPlayers *sync.Mutex
	MutexClients *sync.Mutex
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

type PlayerInput struct {
	Player *Player
	Input byte
}


func (client *ClientSession) ManageSession(){
	go client.ReceiveThread()
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
		} else if msgType == websocket.BinaryMessage {
			if len(msg) < 1 {
				client.SendError(errors.New("No data send"))
			}
			err = client.ReceivePlayerInput(msg[0])
			//if err != nil {
			//	client.SendError(err)
			//}
		} else if (msgType == websocket.CloseMessage || msgType == -1){
			log.Println("Closing client session")
			if (client.Player != nil){
				//client.Player.Disconnect()
				client.Player.PlayerMutex.Lock()
				if _, ok := client.Player.ClientGameChannels[client]; ok{
					delete(client.Player.ClientGameChannels,client)
				}
				client.Player.PlayerMutex.Unlock()
			}
			client.LeaveRoom()
			Server.MutexClients.Lock()
			deleted := false
			for i := range Server.Clients {
				if Server.Clients[i] == client{
					deleted = true
					Server.Clients = append(Server.Clients[:i],Server.Clients[i+1:]...)
					break
				}
			}
			if deleted == false {
				log.Fatalln("Couldn't delete client session")
			}
			Server.MutexClients.Unlock()
			break
		} else {
			log.Println(string(msg))
			log.Println(msgType)
			log.Println("Got something else")
		}
	}
	client.Close <- true
}

func (client *ClientSession) ReceiveThread(){
	Loop: for {
		select {
		case msg := <-client.WriteChannel:
				err := client.SendWholeMessage(msg)
				if err != nil {
					break
				}
		case _ = <-client.Close:
			client.Close <- true
			break Loop
		}
	}

}

func (client *ClientSession) HandleMessage(message Message){
	log.Println("Received Message: " + message.Message)
	var err error;
	err = nil
	switch message.Message {
	case MsgSession:
		client.RestoreSession(message)
		client.SendRooms()
	case MsgCreatePlayer:
		err = client.CreatePlayer(message)
	case MsgCreateRoom:
		err = client.CreateRoom(message)
	case MsgJoinRoom:
		err = client.JoinRoom(message)
	case MsgLeaveRoom:
		err = client.LeaveRoom()
	case MsgReadyPlayer:
		err = client.ReadyPlayer()
	}
	if err != nil {
		client.SendError(err)
	}
}

// Server.MutexClients.Lock -> Server.MutexClients.unLock
func InformAll(msg Message){
	Server.MutexClients.Lock()
	defer Server.MutexClients.Unlock()
	for _, client := range Server.Clients{
		client.WriteChannel <- msg
	}
}

//Server.MutexClients.Lock -> Server.MutexClients.unlock
func InformOther(msg Message,player []*Player){
	Server.MutexClients.Lock()
	defer Server.MutexClients.Unlock()
	for _, play := range player{
		for _, client := range Server.Clients{
			if client.Player == play {
				client.WriteChannel <- msg
			}
		}
	}
}

//serv.MutexRooms.Lock -> serv.MutexRooms.unLock
func (serv *ServerState) FindRoom(roomId string) *Room  {
	serv.MutexRooms.Lock()
	defer serv.MutexRooms.Unlock()
	if room , ok := serv.Rooms[roomId]; ok {
		return room
	}
	return nil
}

//serv.MutexPlayers.Lock -> serv.MutexPlayers.unLock
func (serv *ServerState) FindPlayer(playerId string) *Player {
	serv.MutexPlayers.Lock()
	defer serv.MutexPlayers.Unlock()
	if play , ok := serv.Players[playerId]; ok {
		return play
	}
	return nil
}


//client.WriteMutex.Lock -> client.WriteMutex.unLock
func (client *ClientSession) SendWholeMessage(message Message) error {
	client.WriteMutex.Lock()
	defer client.WriteMutex.Unlock()
	err := client.Socket.WriteJSON(message)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (client *ClientSession) SendMessage(msgText string ,object interface{}) error {
	message := createMessage(msgText, object)
	return client.SendWholeMessage(message)
}

func createMessage(msgText string,object interface{}) Message {
	var data = map[string]interface{}{}
	mar, _ := json.Marshal(object)
	json.Unmarshal(mar, &data)
	message := Message{}
	message.Message = msgText
	message.Data = data
	return message
}

func (client *ClientSession) SendError(msg error){
	log.Println(msg)
	err := ErrorMessage{}
	err.ErrorMsg = msg.Error()
	client.SendMessage(MsgError, err)
}


func (client *ClientSession) JoinRoom(msg Message) error {
	if (client.Player == nil){
		return errors.New("First login with player name")
	}
	roomId := RoomId{}
	err := mapstructure.Decode(msg.Data,&roomId)
	if err != nil{
		return err
	}
	client.Player.PlayerMutex.Lock()
	prevRoom := client.Player.CurrentRoom
	client.Player.PlayerMutex.Unlock()
	if prevRoom != nil && prevRoom.Id == roomId.Id {
		return nil
	}
	client.LeaveRoom()
	client.Player.PlayerMutex.Lock()
	prevRoom = client.Player.CurrentRoom
	client.Player.PlayerMutex.Unlock()
	if prevRoom != nil{
		return errors.New("Cannot join new room because current room is still playing game")
	}
	room, err := roomId.JoinRoom(client.Player)
	if err != nil {
		return err
	}
	room.RoomMutex.Lock()
	roomMsg := createMessage(MsgRoomInfo,room)
	room.RoomMutex.Unlock()
	InformAll(roomMsg)
	return nil
}


func (client *ClientSession) RestoreSession(message Message) error{
	if client.Player != nil {
		return errors.New("Cannot restore session because client already has assigned player")
	}
	cookie := CookieSession{}
	err := mapstructure.Decode(message.Data,&cookie)
	if err != nil{
		return err
	}
	player := cookie.CheckCookie()
	if player == nil {
		return errors.New("Couldn't find player session")
	}
	player.PlayerMutex.Lock()
	//player.Connected = true
	player.ClientGameChannels[client] = make (chan []byte)
	msg := createMessage(MsgCreatePlayer,player)
	player.PlayerMutex.Unlock()
	client.Player = player
	client.SendWholeMessage(msg)
	go client.SendGameInfo()
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
	createdPlayer.PlayerMutex.Lock()
	createdPlayer.ClientGameChannels[client] = make (chan []byte)
	msg := createMessage(MsgCreatePlayer,createdPlayer)
	createdPlayer.PlayerMutex.Unlock()
	client.Player = createdPlayer
	client.SendWholeMessage(msg)
	go client.SendGameInfo()
	return nil
}

func (client *ClientSession) SendRooms() error{
	Server.MutexRooms.Lock()
	if (len(Server.Rooms) == 0){
		Server.MutexRooms.Unlock()
		return nil
	}
	for _, room := range Server.Rooms{
		room.RoomMutex.Lock()
	}
	msg := createMessage(MsgAllRooms,Server.Rooms)
	for _, room := range Server.Rooms{
		room.RoomMutex.Unlock()
	}
	Server.MutexRooms.Unlock()
	err :=  client.SendWholeMessage(msg)

	return err
}

//Server.MutexRooms.Lock -> Server.MutexRooms.unLock -> RoomMutex.lock -> RoomMutex.unlock
func (client *ClientSession) CreateRoom(message Message) error{
	roomCreation := RoomCreation{}
	err := mapstructure.Decode(message.Data, &roomCreation)
	if err != nil {
		return err
	}
	createdRoom, err := roomCreation.CreateGameRoom()
	if err != nil {
		return err
	}

	createdRoom.RoomMutex.Lock()
	msg := createMessage(MsgRoomInfo,createdRoom)
	createdRoom.RoomMutex.Unlock()
	InformAll(msg)
	return nil
}


func (client *ClientSession) LeaveRoom() error {
	if (client.Player == nil){
		return errors.New("First login with player name")
	}
	room := client.Player.CurrentRoom
	if (room == nil){
		return errors.New("Player didn't join any room");
	}
	room.RoomMutex.Lock()
	defer room.RoomMutex.Unlock()
	if room.Playing == true {
		return errors.New("Cannot leave room if game is playing")
	}
	for i := range room.Players{
		if room.Players[i] == client.Player {
			room.Players[i] = nil
			room.NumberOfPlayers--
			if room.Ready[i] == true{
				room.Ready[i] = false
				room.ReadyCount--
				room.wg.Add(1)
			}
			break
		}
	}
	client.Player.PlayerMutex.Lock()
	client.Player.CurrentRoom = nil
	client.Player.PlayerMutex.Unlock()
	msg := createMessage(MsgRoomInfo,room)
	InformAll(msg)
	return nil
}

func (client *ClientSession) ReadyPlayer() error {
	if client.Player == nil {
		return errors.New("First login with player name")
	}
	client.Player.PlayerMutex.Lock()
	room := client.Player.CurrentRoom
	client.Player.PlayerMutex.Unlock()
	if room == nil {
		return errors.New("Cannot ready because player didn't join any room")
	}
	room.RoomMutex.Lock()
	defer room.RoomMutex.Unlock()
	for i := range room.Players{
		if room.Players[i] == client.Player && room.Ready[i] == false{
			room.Ready[i] = true
			room.ReadyCount++
			room.wg.Done(	)
		}
	}
	msg := createMessage(MsgRoomInfo,room)
	InformAll(msg)
	return nil
}

func (client *ClientSession) SendGameInfo(){
	if client.Player == nil{
		log.Fatalln("Something went wrong")
		return
	}
	Loop: for {
		select {
		case data := <-client.Player.ClientGameChannels[client]:
			client.WriteMutex.Lock()
			client.Socket.WriteMessage(websocket.BinaryMessage,data)
			client.WriteMutex.Unlock()
		case <- client.Close:
			client.Close <- true
			break Loop
		}
	}
}

func (client *ClientSession) ReceivePlayerInput(move byte) error {
	if client.Player == nil {
		return errors.New("Cannot process input if not logged")
	}
	if move > 2 {
		return errors.New("Not valid input")
	}
	client.Player.PlayerMutex.Lock()
	defer client.Player.PlayerMutex.Unlock()
	if client.Player.CurrentRoom == nil{
		return errors.New("Player didn't join any room")
	}
	if (client.Player.CurrentRoom.Playing == false){
		return errors.New("Cannot process input with no game playing")
	}
	client.Player.LastMove = move
	return nil
}