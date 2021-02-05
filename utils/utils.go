package utils

import (
	"fmt"
	"github.com/uds5501/re-formers-server/config"
	"math/rand"
	"strconv"
	"time"
	b64 "encoding/base64"
)

var (
	names = []string{"Racoon", "Panda", "Giraffe", "Stridebreaker", "Sherlock", "Echo", "Ponzi", "Shaco", "Blitzcrank", "Mozerella", "Godzilla", "Zombie", "Karen", "Kyle", "Mitro"}
	colours = []string{"Chocolate", "Coral", "Brown", "Black", "Blue", "Cyan", "DarkKhaki", "DeepPink", "ForestGreen", "GreenYellow", "Indigo", "OliveDrab", "Yellow", "Orange", "Red" }
	letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)
type Utils struct {
	ptrName int
	ptrColour int
	NameMapper map[string]bool
}

func (u *Utils) AllowEntry() bool {
	if len(u.NameMapper) < 15 {
		return true
	} else {
		return false
	}
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

func (u *Utils) CreateMessage(message string, clientData *config.ClientObject) config.ServerClientCommunication {
	return config.ServerClientCommunication{
		MessageType: message,
		ClientObject: clientData,
	}
}

func (u *Utils) GetEntryToken(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(100) % 32]
	}
	strb := fmt.Sprintf("%s%s", string(b), strconv.Itoa(int(time.Now().UnixNano())))

	//log.Println(int(time.Now().UnixNano()))
	return b64.URLEncoding.EncodeToString([]byte(strb))
}

func Init() *Utils {
	rand.Seed(time.Now().UnixNano())
	return &Utils{ptrName: 0, NameMapper: map[string]bool{}, ptrColour: 0}
}
