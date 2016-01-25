package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"queue"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const debug = true
const maxSizeHealthStatusQueue = 600 // Magnificent Server Log Store Size, displays the last 600 seconds in the plot

type Msg struct {
	MessageId string
	Content   string
	TimeStamp int64
}

// Client connection consists of the websocket and the client ip
type Client struct {
	websocket *websocket.Conn
	clientIP  string
}

type ServiceMonitor struct {
	errChan           chan error // unbuffered channel
	errChanWebsock    chan error // unbuffered channel
	activeClients     map[string]Client
	healthStatusChan  chan bool
	alertChan         chan string
	newClientChan     chan Client
	alertQueue        *queue.Queue
	healthStatusQueue *queue.Queue
}

func NewServiceMonitor() *ServiceMonitor {
	m := ServiceMonitor{}
	m.activeClients = make(map[string]Client)
	m.errChan = make(chan error)
	m.healthStatusChan = make(chan bool, 10)
	m.alertChan = make(chan string, 10)
	m.newClientChan = make(chan Client, 10)
	m.alertQueue = queue.NewQueue()
	m.healthStatusQueue = queue.NewQueue()
	return &m
}

func IntToString(value int) string {
	return strconv.FormatInt(int64(value), 10)
}

func StringToFloat(value string) float64 {
	result, _ := strconv.ParseFloat(value, 64)
	return result
}

func StringToInt(value string) int {
	result, _ := strconv.ParseInt(value, 10, 64)
	return int(result)
}

func FloatToString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func BoolToString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func (m *ServiceMonitor) sendClientMsg(msg *Msg, ip string) {
	var err error
	var Message = websocket.JSON

	if err = Message.Send(m.activeClients[ip].websocket, msg); err != nil {
		// we could not send the message to a peer
		log.Println("Could not send message to:", ip, err.Error())
		log.Println("Client disconnected:", ip)
		delete(m.activeClients, ip)
	}
}

func (m *ServiceMonitor) sendBroadcastMsg(msg *Msg) {
	var err error
	var Message = websocket.JSON

	for ip, _ := range m.activeClients {
		if err = Message.Send(m.activeClients[ip].websocket, msg); err != nil {
			// we could not send the message to a peer
			log.Println("Could not send message to:", ip, err.Error())
			log.Println("Client disconnected:", ip)
			delete(m.activeClients, ip)
		}
	}
}

func (m *ServiceMonitor) sendQueueData(ip string) {
	for i := 0; i < m.healthStatusQueue.Len(); i++ {
		e, found := m.healthStatusQueue.Get(i)

		if found {
			if msg, ok := e.(*Msg); ok {
				m.sendClientMsg(msg, ip)
			}
		}
	}

	for i := 0; i < m.alertQueue.Len(); i++ {
		e, found := m.alertQueue.Get(i)

		if found {
			if msg, ok := e.(*Msg); ok {
				m.sendClientMsg(msg, ip)
			}
		}
	}
}

// this routine handles all outgoing websocket messages
func (m *ServiceMonitor) pushDataToClients() {
	for {
		select {
		// a new Client is connecting
		case newClient := <-m.newClientChan:
			// send current Queue data to the new connecting client
			m.activeClients[newClient.clientIP] = newClient
			m.sendQueueData(newClient.clientIP)

		// broadcast a new health status message to all clients
		// newHealthStatus == 1 ... deamon failure
		// newHealthStatus == 0 ... deamon ok
		case newHealthStatus := <-m.healthStatusChan:
			msg := Msg{"Plot", BoolToString(newHealthStatus), time.Now().UnixNano() / int64(time.Millisecond)}
			m.sendBroadcastMsg(&msg)
			// add msg to HealthStatusQueue
			if m.healthStatusQueue.Len() < maxSizeHealthStatusQueue {
				m.healthStatusQueue.Push(&msg)
			} else {
				m.healthStatusQueue.Pop()
				m.healthStatusQueue.Push(&msg)
			}

		// broadcast an alert message to all clients
		case newAlert := <-m.alertChan:
			msg := Msg{"Alert", newAlert, time.Now().UnixNano() / int64(time.Millisecond)}
			m.sendBroadcastMsg(&msg)
			// add msg to alertQueue
			m.alertQueue.Push(&msg)
		}
	}
}

// reference: https://github.com/Niessy/websocket-golang-chat
// WebSocket server to handle clients
func (m *ServiceMonitor) WebSocketServer(ws *websocket.Conn) {
	var err error

	// cleanup on server side
	defer func() {
		if err = ws.Close(); err != nil {
			log.Println("Websocket could not be closed", err.Error())
		}
	}()

	client := ws.Request().RemoteAddr
	if debug {
		log.Println("New client connected:", client)
	}

	m.newClientChan <- Client{ws, client}

	// wait for errChan, so the websocket stays open otherwise it'll close
	err = <-m.errChanWebsock
}

// handler for the main page
func HomeHandler(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-type", "text/html")
	webpage, err := ioutil.ReadFile("home.html")

	if err != nil {
		http.Error(response, fmt.Sprintf("home.html file error %v", err), 500)
	}

	fmt.Fprint(response, string(webpage))
}

func (m *ServiceMonitor) parseResponse(resp string) {
	if !strings.Contains(resp, "Magnificent!") {
		m.healthStatusChan <- false
	} else {
		m.healthStatusChan <- true
		m.alertChan <- "Deamon has failed"
	}
}

func (m *ServiceMonitor) requestDeamonStatus(url string) {
	response, err := http.Get(url)
	if err != nil {
		if debug {
			log.Println("requestDeamonStatus failed:", err.Error())
			return
		}
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		if debug {
			log.Println("requestDeamonStatus failed:", err.Error())
			return
		}
	}

	resp := string(contents)
	m.parseResponse(resp)
}

func (m *ServiceMonitor) monitorDeamon(url string, time_interval time.Duration) {
	for {
		go m.requestDeamonStatus(url)
		time.Sleep(time_interval * time.Second)
	}
}

func (m *ServiceMonitor) startHTTPServer() {
	http.Handle("/", http.HandlerFunc(HomeHandler))
	http.Handle("/sock", websocket.Handler(m.WebSocketServer))

	err := http.ListenAndServe(":8080", nil)
	m.errChanWebsock <- err
	m.errChan <- err
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	m := NewServiceMonitor()

	go m.startHTTPServer()
	go m.pushDataToClients()
	go m.monitorDeamon("http://localhost:12345/", 1)

	err := <-m.errChan
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
