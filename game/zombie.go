package game

import "math/rand"

var nameList = []string{
	"Fuser",
	"Leecher",
	"Grunter",
	"Griever",
	"Stout_Zombie",
	"Acher",
	"Snacker",
	"Skipper",
	"Experimental_Zombie",
	"Chewer",
}

type zombie struct {
	name string
	x    int
	y    int
}

func newZombie() zombie {
	return zombie{
		name: nameList[rand.Intn(len(nameList))],
		x:    0,
		y:    0,
	}
}
