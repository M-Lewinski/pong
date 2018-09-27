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

type GameCreation struct {
	Health int
	PlayerCount int
	Name string
}

type PlayerCreation struct{
	Name string
}

func (gCreation *GameCreation) CreateGameRoom() error {
	if (gCreation.Health > 255  || gCreation.Health < 0){
		return errors.New("health must be a value between 0 and 255")
	}

	if (gCreation.PlayerCount > 4 || gCreation.PlayerCount < 2){
		return errors.New("player count must be between 2 and 4")
	}
	if (len(strings.TrimSpace(gCreation.Name)) == 0){
		return errors.New("Name cannot be empty")
	}

	room := &Room{
		Game: nil,
		Players: make([]*Player,gCreation.PlayerCount),
		Playing: false,
		MaxPlayers: gCreation.PlayerCount,
		NumberOfPlayers: 0,
		RoomMutex: &sync.Mutex{},
	}
	u, _ := uuid.NewV4()
	room.Id = u.String()
	Server.Mutex.Lock()
	Server.Rooms[room.Id] = room
	Server.Mutex.Unlock()
	return nil
}

func (newPlayer *PlayerCreation) CreateNewPlayer() (*Player,error){
	log.Println("Creating new player")
	if (len(strings.TrimSpace(newPlayer.Name)) == 0){
		return nil, errors.New("Player name cannot be empty string")
	}
	createdPlayer := &Player{}
	createdPlayer.Name = newPlayer.Name
	createdPlayer.CurrentRoom = ""
	createdPlayer.Connected = true
	u, _ := uuid.NewV4()
	createdPlayer.Id = u.String()
	Server.Mutex.Lock()
	Server.Players[createdPlayer.Id] = createdPlayer
	Server.Mutex.Unlock()
	return createdPlayer,nil
}