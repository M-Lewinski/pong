package main

import (
	"errors"
	"strings"
	"sync"
	"github.com/nu7hatch/gouuid"
	"log"
)

type CookieSession struct {
	Cookie string
}

func (cookie *CookieSession) CheckCookie() *Player{
	if cookie.Cookie == ""{
		return nil
	}
	return Server.FindPlayer(cookie.Cookie)
}

type RoomCreation struct {
	Life        int
	PlayerCount int
	Name        string
}

type PlayerCreation struct{
	Name string
}

type RoomId struct {
	Id string
}


func (rid *RoomId) JoinRoom(player *Player) (*Room,error) {
	if (player == nil){
		return nil,errors.New("First login with player name")
	}
	room := Server.FindRoom(rid.Id)
	if (room == nil){
		return nil, errors.New("Room with provided id does not exists")
	}
	room.RoomMutex.Lock()
	defer room.RoomMutex.Unlock()
	if room.NumberOfPlayers == room.MaxPlayers{
		return nil,errors.New("Room is already full")
	}
	for i := range room.Players {
		if (room.Players[i] == nil){
			room.Players[i] = player
			room.NumberOfPlayers++
			break
		}
	}
	player.PlayerMutex.Lock()
	player.CurrentRoom = room
	player.PlayerMutex.Unlock()
	return room,nil
}

//Server.MutexRooms.Lock -> Server.MutexRooms.unLock
func (gCreation *RoomCreation) CreateGameRoom() (*Room,error) {
	if (gCreation.Life > 255  || gCreation.Life < 1){
		return nil, errors.New("health must be a value between 1 and 255")
	}

	//if (gCreation.PlayerCount > 4 || gCreation.PlayerCount < 2){
	if (!(gCreation.PlayerCount == 4 || gCreation.PlayerCount == 2)){
		return nil, errors.New("mainPlayer count must be 2 or 4")
	}
	if (len(strings.TrimSpace(gCreation.Name)) == 0){
		return nil, errors.New("Name cannot be empty")
	}
	room := &Room{
		Players: make([]*Player,gCreation.PlayerCount),
		Playing: false,
		MaxPlayers: gCreation.PlayerCount,
		Life: gCreation.Life,
		NumberOfPlayers: 0,
		RoomMutex: &sync.Mutex{},
		Ready: make([]bool,gCreation.PlayerCount),
		ReadyCount: 0,
		wg: &sync.WaitGroup{},
		//Result: make([]int,gCreation.PlayerCount),
		Name: gCreation.Name,
	}
	room.wg.Add(room.MaxPlayers)
	game := &Game{}
	game.Lifes = make([]byte,room.MaxPlayers)
	for i := range game.Lifes{
		game.Lifes[i] = IntToByte(room.Life)
	}
	go game.Start(room)
	u, _ := uuid.NewV4()
	room.Id = u.String()
	Server.MutexRooms.Lock()
	Server.Rooms[room.Id] = room
	Server.MutexRooms.Unlock()
	return room,nil
}

// Server.MutexPlayers.Lock -> Server.MutexPlayers.unLock
func (newPlayer *PlayerCreation) CreateNewPlayer() (*Player,error){
	log.Println("Creating new mainPlayer")
	if (len(strings.TrimSpace(newPlayer.Name)) == 0){
		return nil, errors.New("Player name cannot be empty string")
	}
	createdPlayer := &Player{
		Name: newPlayer.Name,
		CurrentRoom: nil,
		Connected: true,
		PlayerMutex: &sync.Mutex{},
		GameChannel: make(chan []byte),
		LastMove: 0,
	}
	u, _ := uuid.NewV4()
	createdPlayer.Id = u.String()
	Server.MutexPlayers.Lock()
	Server.Players[createdPlayer.Id] = createdPlayer
	Server.MutexPlayers.Unlock()
	return createdPlayer,nil
}