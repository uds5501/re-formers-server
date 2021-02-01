package utils

import (
	"fmt"
	"math/rand"
	"time"
)

var (
	names = []string{"Racoon", "Panda", "Giraffe", "Stridebreaker", "Sherlock", "Echo", "Ponzi", "Shaco", "Blitzcrank", "Mozerella", "Godzilla", "Zombie", "Karen", "Kyle", "Mitro"}
	colours = []string{"Chocolate", "Coral", "Brown", "Black", "Blue", "Cyan", "DarkKhaki", "DeepPink", "ForestGreen", "GreenYellow", "Indigo", "OliveDrab", "Yellow", "Orange", "Red" }
)
type Utils struct {
	ptrName int
	ptrColour int
	NameMapper map[string]bool
}

func (u *Utils) AllowEntry() bool {
	fmt.Println(len(u.NameMapper))
	return false
}
// returns available username + colour combo
func (u *Utils) AssignData() (string, string){
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	a := r1.Intn(25)
	b := r1.Intn(50)
	cnt := 50
	var userName, colour string
	for {
		if cnt == 0 {
			return "-1", "-1"
		}
		userName = names[a%15]
		colour = colours[b%15]
		_, found := u.NameMapper[userName+colour]
		if found == true {
			c := r1.Intn(100)
			d := r1.Intn(100)
			a += c
			b += d
			cnt--
		} else {
			u.NameMapper[userName+colour] = true
			break
		}
	}
	return userName, colour
}

func Init() *Utils {
	return &Utils{ptrName: 0, NameMapper: map[string]bool{}, ptrColour: 0}
}
