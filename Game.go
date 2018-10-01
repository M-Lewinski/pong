package main

import (
	"log"
	"time"
	"math"
	"bytes"
	"encoding/binary"
	)

const (
	GameWidth               = 640
	GameHeight              = 640
	BallSpawnTimerInSeconds = 1.0
	BallRadius              = 10.0
	SpawnerSize             = 30.0
	SpawnerRotationPerSec   = 123
	BallDefaultSpeed        = 300.0
	DataBajtSize            = 8
	PlatformWidth           = 70.0
	PlatformHeight          = 20.0
	PlatformDefaultSpeed    = 400.0
	DistanceFromPlatform    = 15.0
	DangerAreaSize = 20
)

type Vec2 struct {
	x float64
	y float64
}

type Ball struct {
	Center        Vec2
	Velocity        Vec2
	Radius        float64
	radiusSquared float64
}

type Platform struct {
	Center Vec2
	Velocity Vec2
	Width float64
	Height float64
}

type Spawner struct {
	Center   Vec2
	ArrowPos Vec2
	DefaultPos     Vec2
	Rotation float64
}


type Game struct {
	Balls []*Ball
	Platforms []*Platform
	LastSpawnedBall time.Duration
	MaxSpawnedBalls int
	Spawner *Spawner
	Lifes []byte
}


const (
	Up  = 0
	Right = 1
	Down = 2
	Left = 3

	NoneMove  = 0
	RightMove = 1
	LeftMove  = 2
)


func (game *Game) Start(room *Room)  {
	game.Spawner = &Spawner{Center: Vec2{x: GameWidth/2.0,y:GameHeight/2.0},Rotation: 0.0}
	game.Spawner.DefaultPos = Vec2{x:0,y:-SpawnerSize}
	for{
		room.wg.Wait()
		room.RoomMutex.Lock()
		if room.ReadyCount == room.MaxPlayers{
			room.RoomMutex.Unlock()
			room.Playing = true
			break
		}
		room.RoomMutex.Unlock()
	}
	game.MaxSpawnedBalls = room.MaxPlayers*5
	log.Println("Starting game!")
	prevTime := time.Now()
	timePerUpdate := 1000/60.0
	gameUpdateTime := time.Duration(float64(time.Millisecond)*timePerUpdate)
	playerMoves := make([]byte,room.MaxPlayers)
	game.Platforms = make([]*Platform,room.MaxPlayers)
	for i := range game.Platforms{
		plat := &Platform{}
		plat.Velocity = Vec2{x:0,y:0}
		plat.Width = PlatformWidth
		plat.Height = PlatformHeight
		switch i {
		case 0:
			plat.Center = Vec2{x:GameWidth/2.0,y:GameHeight-DangerAreaSize-(PlatformHeight/2.0)}
		case 1:
			plat.Center = Vec2{x:GameWidth/2.0,y:DangerAreaSize+(PlatformHeight/2.0)}
		case 2:
			plat.Center = Vec2{x:DangerAreaSize+(PlatformHeight/2.0),y:GameHeight/2.0}
			plat.Width = PlatformHeight
			plat.Height = PlatformWidth
		case 3:
			plat.Center = Vec2{x:GameWidth-DangerAreaSize-(PlatformHeight/2.0),y:GameHeight/2.0}
			plat.Width = PlatformHeight
			plat.Height = PlatformWidth
		}
		game.Platforms[i] = plat
	}
	for{
		<-time.After(gameUpdateTime)
		delta := time.Since(prevTime)
		prevTime = time.Now()
		for i, player := range  room.Players{
			player.PlayerMutex.Lock()
			if (player.Connected == true){
				playerMoves[i] = player.LastMove
			} else {
				playerMoves[i] = NoneMove
			}
			player.PlayerMutex.Unlock()
		}
		game.MovePlayers(playerMoves)
		//game.SpawnBall(delta, Vec2{x: GameWidth/2.0,y:GameHeight/2.0,})
		game.SpawnBall(delta, game.Spawner.Center)
		game.Update(delta)
		game.CollisionDetection()
		data := game.CreateData(room)
		for _, player := range room.Players{
			player.PlayerMutex.Lock()
			if player.Connected == true {
				player.GameChannel <- data
			}
			player.PlayerMutex.Unlock()
		}
	}
}

func (game *Game) MovePlayers(playerMove []byte){
	for i, move := range playerMove{
		if move == NoneMove {
			game.Platforms[i].Velocity.x = 0
			game.Platforms[i].Velocity.y = 0
		} else if move == RightMove{
			switch i {
			case 0:
				game.Platforms[i].Velocity.x = PlatformDefaultSpeed
			case 1:
				game.Platforms[i].Velocity.x = -PlatformDefaultSpeed
			case 2:
				game.Platforms[i].Velocity.y = PlatformDefaultSpeed
			case 3:
				game.Platforms[i].Velocity.y = -PlatformDefaultSpeed
			}
		} else if move == LeftMove{
			switch i {
			case 0:
				game.Platforms[i].Velocity.x = -PlatformDefaultSpeed
			case 1:
				game.Platforms[i].Velocity.x = PlatformDefaultSpeed
			case 2:
				game.Platforms[i].Velocity.y = -PlatformDefaultSpeed
			case 3:
				game.Platforms[i].Velocity.y = PlatformDefaultSpeed
			}
		}
	}
}

func (game *Game) CreateData(room *Room) []byte {
	// dataSize + playerCount + life*NumberOfPlayers +BallRadius + PlatformWidth + PlatformHeight + DangerZoneSize + spawnerRotation*2*dataSize  + dataSize*2pos*playerCount + dataSize*2pos*Balls
	size := 1 + 1 +  room.MaxPlayers + DataBajtSize + (2*DataBajtSize) + DataBajtSize + (2*DataBajtSize) + (2*DataBajtSize*len(game.Platforms)) + (DataBajtSize*2*len(game.Balls))
	data := make([]byte,size)
	index := 0
	data[index] = DataBajtSize
	index++
	data[index] = IntToByte(room.MaxPlayers)
	index++
	index = PutBytesIntoData(index, data,game.Lifes)
	index = PutBytesIntoData(index, data,FloatToBytes(BallRadius))
	index = PutBytesIntoData(index, data, FloatToBytes(PlatformWidth))
	index = PutBytesIntoData(index, data, FloatToBytes(PlatformHeight))
	index = PutBytesIntoData(index, data, FloatToBytes(DangerAreaSize))
	index = PutBytesIntoData(index,data,FloatToBytes(game.Spawner.ArrowPos.x))
	index = PutBytesIntoData(index,data,FloatToBytes(game.Spawner.ArrowPos.y))
	for _, plat := range game.Platforms{
		index = PutBytesIntoData(index,data,FloatToBytes(plat.Center.x))
		index = PutBytesIntoData(index,data,FloatToBytes(plat.Center.y))
	}
	for  _, ball := range game.Balls{
		index = PutBytesIntoData(index,data,FloatToBytes(ball.Center.x))
		index = PutBytesIntoData(index,data,FloatToBytes(ball.Center.y))
	}
	return data
}

func IntToByte(data int) byte{
	byteArray := make([]byte,4)
	binary.LittleEndian.PutUint32(byteArray, uint32(data))
	return byteArray[0]
}

func PutBytesIntoData(i int,data []byte,byteInput []byte) int{
	var j int
	for j = 0; j < len(byteInput) ; j++  {
		data[i] = byteInput[j]
		i++
	}
	return i
}

func FloatToBytes(data float64) []byte {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, data)
	if err != nil {
		log.Println("Float to binary convertion error")
		return nil
	}
	return buf.Bytes()
}

func (game *Game) Update(delta time.Duration)  {
	for _, ball := range game.Balls {
		ball.Center.x += ball.Velocity.x*delta.Seconds()
		ball.Center.y += ball.Velocity.y*delta.Seconds()
	}
	for _, plat := range game.Platforms{
		plat.Center.x += plat.Velocity.x*delta.Seconds()
		plat.Center.y += plat.Velocity.y*delta.Seconds()
		if plat.Center.x - (plat.Width/2.0) < 0 {
			plat.Center.x = plat.Width/2.0
		} else if plat.Center.x + plat.Width/2.0 > GameWidth{
			plat.Center.x = GameWidth - plat.Width/2.0
		}
		if plat.Center.y - plat.Height/2.0 < 0{
			plat.Center.y = plat.Height/2.0
		} else if plat.Center.y + plat.Height/2.0 > GameHeight{
			plat.Center.y = GameHeight - plat.Height/2.0
		}
	}

	game.Spawner.Rotation += SpawnerRotationPerSec*delta.Seconds()
	if (game.Spawner.Rotation > 360.0){
		game.Spawner.Rotation -= 360.0
	}
	game.Spawner.ArrowPos = game.Spawner.DefaultPos.Rotate(game.Spawner.Rotation)
	game.Spawner.ArrowPos.x += game.Spawner.Center.x
	game.Spawner.ArrowPos.y += game.Spawner.Center.y
}

func (vec Vec2) Rotate(deg float64) Vec2{
	newVec := Vec2{}
	rad := DegToRad(deg)
	newVec.x = math.Cos(rad)*vec.x - math.Sin(rad)*vec.y
	newVec.y = math.Sin(rad)*vec.x + math.Cos(rad)*vec.y
	return newVec
}

func (game *Game) SpawnBall(deltaSeconds time.Duration, spawnPlace Vec2){
	if len(game.Balls) >= game.MaxSpawnedBalls {
		return
	}
	game.LastSpawnedBall += deltaSeconds
	if game.LastSpawnedBall.Seconds() <= (float64)(BallSpawnTimerInSeconds*len(game.Balls)) {
		return
	}
	game.LastSpawnedBall = time.Duration(0)
	rad := DegToRad(game.Spawner.Rotation)
	//velX := cos(deg)*x - sin(deg)*y
	velX := -math.Sin(rad)*BallDefaultSpeed
	//velY := sin(deg)*x + cos(deg)*y
	velY := math.Cos(rad)*BallDefaultSpeed
	newBall := &Ball{Center: spawnPlace,Radius: BallRadius,Velocity:Vec2{x:velX,y:velY},radiusSquared: math.Pow(BallRadius,2)}
	game.Balls = append(game.Balls,newBall)
}

func (game *Game) CollisionDetection(){
	game.CollisionBalls()
	game.CollisionPlatform()
}

func (game *Game) CollisionBalls() {
	for _, ball := range game.Balls {
		if ball.Center.x-BallRadius < 0 {
			ball.Center.x = 0.0 + BallRadius
			ball.Velocity.x *= -1
		} else if ball.Center.x+BallRadius > GameWidth {
			ball.Center.x = GameWidth - BallRadius
			ball.Velocity.x *= -1
		}
		if ball.Center.y-BallRadius < 0 {
			ball.Center.y = 0 + BallRadius
			ball.Velocity.y *= -1
		} else if ball.Center.y+BallRadius > GameHeight {
			ball.Center.y = GameHeight - BallRadius
			ball.Velocity.y *= -1
		}
	}
	for i, ball := range game.Balls {
		//for j := i + 1; j < len(game.Balls); j++ {
		for j := 0; j < len(game.Balls); j++ {
			if (j == i) {
				continue
			}
			distVec, dist := CalcDistance(ball.Center, game.Balls[j].Center)
			if dist < ball.Radius {
				normDistVec := distVec.Normalize2(dist)
				normDistVec.x *= BallDefaultSpeed
				normDistVec.y *= BallDefaultSpeed
				ball.Velocity = normDistVec
				ball.Center.x += distVec.x / 2.0
				ball.Center.y += distVec.y / 2.0
				normDistVec2 := Vec2{x: normDistVec.x * -1, y: normDistVec.y * -1}
				game.Balls[j].Velocity = normDistVec2
				game.Balls[j].Center.x -= distVec.x / 2.0
				game.Balls[j].Center.y -= distVec.y / 2.0
			}
		}
	}
}


func (game *Game) CollisionPlatform(){
	for _ , plat := range game.Platforms {
		for _, ball := range game.Balls {
			nearestPoint := NearestPointRectBall(ball,plat.Center,plat.Width,plat.Height)
			distVec, dist  := CalcDistance(nearestPoint,ball.Center)
			if dist < ball.Radius{
				direction := CalculateDirection(distVec)
				switch direction {
				case Up:
					ball.Center.y += distVec.y + (-ball.Radius)
					ball.Velocity.y *= -1
				case Right:
					//ball.Center.x += distVec.x + ball.Radius
					ball.Center.x = nearestPoint.x + ball.Radius
					ball.Velocity.x *= -1
				case Down:
					ball.Center.y += distVec.y + ball.Radius
					ball.Velocity.y *= -1
				case Left:
					//ball.Center.x += distVec.x + (-ball.Radius)
					ball.Center.x = nearestPoint.x + (-ball.Radius)
					ball.Velocity.x *= -1
				}
			}
		}
	}
}

func CalculateDirection(dist Vec2) int{
	directVectors := []Vec2{
		{x: 0, y: 1.0},
		{x: 1.0, y: 0.0},
		{x: 0.0, y: -1.0},
		{x: -1.0, y: 0.0},
	}
	normDist := dist.Normalize()
	max := 0.0
	directContact := -1
	for i := 0; i < len(directVectors) ; i++ {
		angle :=  dotProduct(normDist,directVectors[i])
		if (angle > max){
			max = angle
			directContact = i
		}
	}
	return directContact
}

func dotProduct(vec1 Vec2, vec2 Vec2) float64{
	return (vec1.x * vec2.x) + (vec1.y*vec2.y)
}

func NearestPointRectBall(ball *Ball, rectCenter Vec2, rectWidth float64, rectHeight float64) Vec2{
	LeftTopPoint :=  Vec2{x: rectCenter.x - (rectWidth/2.0),y: rectCenter.y - (rectHeight/2.0)}
	RightBottomPoint :=  Vec2{x: rectCenter.x + (rectWidth/2.0),y: rectCenter.y + (rectHeight/2.0)}
	NearestPointX := math.Max(LeftTopPoint.x, math.Min(ball.Center.x,RightBottomPoint.x))
	NearestPointY := math.Max(LeftTopPoint.y, math.Min(ball.Center.y,RightBottomPoint.y))
	return Vec2{x:NearestPointX,y:NearestPointY}
}

func CalcDistance(pos1 Vec2, pos2 Vec2) (Vec2, float64){
	dist := Vec2{x:pos1.x-pos2.x, y:pos1.y-pos2.y}
	return dist, dist.CalculateMagnitude()
}

func (vec Vec2) CalculateMagnitude() float64{
	return math.Sqrt(math.Pow(vec.x,2)+math.Pow(vec.y,2))
}


func (vec Vec2) Normalize2(magnitude float64) Vec2 {
	return Vec2{x:vec.x/magnitude,y:vec.y/magnitude}
}

func (vec Vec2) Normalize() Vec2 {
	magnitude := vec.CalculateMagnitude()
	return vec.Normalize2(magnitude)
}


func DegToRad(degrees float64) float64 {
	return float64(degrees * (math.Pi / 180.0))
}
