package main

type Pos struct {
	x float32
	y float32
}

type Ball struct {
	Pos Pos
	VelocityX float32
	VelocityY float32
}

type Platform struct {

}


type Game struct {
	Balls []*Ball
}

