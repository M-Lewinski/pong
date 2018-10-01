package main

import (
	"time"
	"sync"
)

type Player struct {
	//Connected          bool
	CurrentRoom        *Room		`json:"-"`
	Id                 string
	Name               string
	LastDisconnectDate time.Time
	PlayerMutex		   *sync.Mutex	`json:"-"`
	GameChannel			map[string]chan []byte	`json:"-"`
	LastMove byte`json:"-"`
}

//PlayerMutex.lock -> Server.MutexRooms.lock -> Server.MutexRooms.unlock -> RoomMutex.lock -> RoomMutex.unlock -> PlayerMutex.unlock
//func (player *Player) Disconnect(client *ClientSession) {
//	player.PlayerMutex.Lock()
//	defer player.PlayerMutex.Unlock()
//	player.LastDisconnectDate = time.Now().UTC()
//
//}
