package utils

import (
	b64 "encoding/base64"
	"fmt"
	"github.com/uds5501/re-formers-server/config"
	"math/rand"
	"strconv"
	"sync"
	"time"
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

	// 1 client can edit only 1 element
	clientFormMap map[string]int
	// 1 element can be edited by many clients
	formClientMap map[int][]string
	utilityMutex sync.Mutex
}

func (u *Utils) AllowEntry() bool {
	if len(u.NameMapper) < 30 {
		return true
	} else {
		return false
	}
}

// returns available username + colour combo
func (u *Utils) AssignData() (string, string) {
	u.utilityMutex.Lock()
	defer u.utilityMutex.Unlock()
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

func (u *Utils) AssignLock(clientToken string, formId int, ele config.FormElement) bool {
	if ele.IsDeleted {
		return false
	}
	cfId, found := u.clientFormMap[clientToken]
	if found {
		if cfId == formId {
			return true
		}
		return false
	} else {
		u.clientFormMap[clientToken] = formId
		u.formClientMap[formId] = append(u.formClientMap[formId], clientToken)
		return true
	}

}

func (u *Utils) UnlockForm(clientToken string) {
	u.utilityMutex.Lock()
	defer u.utilityMutex.Unlock()
	formId := u.clientFormMap[clientToken]
	delete(u.clientFormMap, clientToken)
	requiredSlice := u.formClientMap[formId]
	for i, p := range requiredSlice {
		if p == clientToken {
			requiredSlice = append(requiredSlice[:i], requiredSlice[i+1:]...)
		}
	}
	u.formClientMap[formId] = requiredSlice
}
func (u *Utils) IsLocked(id int) bool {
	if len(u.formClientMap[id]) > 0 {
		return true
	} else {
		return false
	}
}
func (u *Utils) Reset()  {
	u.ptrName = 0
	u.ptrColour = 0
	u.NameMapper = map[string]bool{}
	u.clientFormMap = map[string]int{}
	u.formClientMap = map[int][]string{}
}
func Init() *Utils {
	rand.Seed(time.Now().UnixNano())
	return &Utils{
		ptrName: 0,
		NameMapper: map[string]bool{},
		ptrColour: 0,
		clientFormMap: map[string]int{},
		formClientMap: map[int][]string{},
	}
}
