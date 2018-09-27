package main

import "time"

type Player struct {
	Connected          bool
	CurrentRoom        string
	Id                 string
	Name               string
	LastDisconnectDate time.Time
}

func (player *Player) Disconnect(){
	player.Connected = false
	player.LastDisconnectDate = time.Now().UTC()
	if player.CurrentRoom != "" {
		Server.Mutex.Lock()
		room := Server.FindRomm(player.CurrentRoom)
		if room != nil {
			room.RoomMutex.Lock()
			for i := range room.Players {
				if (room.Players[i] == player){
					room.Players[i] = nil
				}
			}
			room.RoomMutex.Unlock()
		}
	}
}
