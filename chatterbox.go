package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Message struct {
	Src        int
	Dst        int
	Offer      string
	Answer     string
	Id         int
	ClientAdd  int
	ClientDrop int
}

type Client

// 0 should represent an unknown client.
var maxClientId = 0

type Client struct {
	id      int
	ws      *websocket.Conn
	server  *Server
	offer   string
	logger  *log.Logger
	writeCh chan *Message
	doneCh  chan bool
}

func NewClient(s *Server, ws *websocket.Conn) *Client {
	maxClientId++
	id := maxClientId
	logPfx := fmt.Sprint("connection ", id, ": ")
	connection := &Client{
		id,
		ws,
		s,
		"",
		log.New(os.Stdout, logPfx, log.LstdFlags),
		make(chan *Message),
		make(chan bool)}

	return connection
}

func (c *Client) logit(args ...interface{}) {
	c.logger.Print(args...)
}

func (c *Client) Write(msg *Message) {
	c.writeCh <- msg
}

func (c *Client) listenWrite() {
	c.logit("entering write loop...")
	for {
		select {
		case msg := <-c.writeCh:
			c.logit("writing message: ", msg)
			websocket.JSON.Send(c.ws, msg)
		case <-c.doneCh:
			c.logit("exiting write loop.")
			c.server.DelClient(c)
			c.doneCh <- true
			return
		}
	}
}

func (c *Client) listenRead() {
	c.logit("entering read loop...")
	for {
		select {
		case <-c.doneCh:
			c.logit("exiting read loop.")
			c.server.DelClient(c)
			c.doneCh <- true
			return
		default:
			var message string
			err := websocket.Message.Receive(c.ws, &message)
			if err == io.EOF {
				c.logit("closing connection")
				c.doneCh <- true
			} else if err != nil {
				c.logit("error reading from connection: ", err)
			} else {
				c.logit("recv message of length: ", len(message))
			}
		}
	}
}

func (c *Client) Listen() {
	go c.listenWrite()
	c.listenRead()
}

type Server struct {
	clients     map[int]*Client
	addClientCh chan *Client
	delClientCh chan *Client
}

func NewServer() *Server {
	return &Server{
		make(map[int]*Client),
		make(chan *Client),
		make(chan *Client),
	}
}

func (s *Server) HandleConnection(ws *websocket.Conn) {
	log.Println("Handling new websocket connection: ", ws)
	c := NewClient(s, ws)
	s.AddClient(c)
	c.Listen()
	// for cid, c := range s.clients {
	// 	if c.id != cid && c.offer != "" {
	// 		log.Print("sending offer ", cid, " - > ", c.id)
	// 		websocket.Message.Send(c.ws, c.offer)
	// 	}
	// }
	// go c.Listen()
	// for {
	// 	select {
	// 	case offer := <-c.offerCh:
	// 		for cid, c := range s.clients {
	// 			if cid != c.id {
	// 				log.Print("sending offer ", c.id, " -> ", cid)
	// 				websocket.Message.Send(c.ws, offer)
	// 			}
	// 		}
	// 	case _ = <-c.doneCh:
	// 		delete(s.clients, c.id)
	// 	}
	// }
}

func (s *Server) AddClient(c *Client) {
	s.addClientCh <- c
}

func (s *Server) DelClient(c *Client) {
	s.delClientCh <- c
}

// Listens to server channels to take action. This way all actions on
// the Server object happens in a single thread.
func (s *Server) Listen() {
	for {
		select {
		case c := <-s.delClientCh:
			log.Print("Deleting client: ", c.id)
			delete(s.clients, c.id)
			for oId, o := range s.clients {
				log.Print("checking ", oId)
				if c.id != oId {
					log.Print("sending to ", oId)
					o.Write(&Message{0, oId, "", "", 0, 0, c.id})
				}
			}
		case c := <-s.addClientCh:
			log.Print("Adding client: ", c.id)
			s.clients[c.id] = c
			log.Print("clients: ", len(s.clients))
			log.Print("Telling ", c.id, " its true identity.")
			c.Write(&Message{0, c.id, "", "", c.id, 0, 0})
			for oId, o := range s.clients {
				log.Print("checking ", oId)
				if c.id != oId {
					log.Print("sending to ", oId)
					o.Write(&Message{0, oId, "", "", 0, c.id, 0})
				}
			}

			// log.Print("Sending client ", c.id, " list of all other clients")
			// clientIds := make([]int, 0, len(s.clients))
			// for otherId, _ := range s.clients {
			// 	if c.id != otherId {
			// 		clientIds = append(clientIds, otherId)
			// 	}
			// }
			// msg := Message{c.id, "", "", clientIds}
			// msgJson := json.Marshall(msg)
			// c.Write("clientListMessage")
		}
	}
}

func main() {
	wd, _ := os.Getwd()
	s := NewServer()
	go s.Listen()
	fmt.Println("Starting server on port 8080...")
	fmt.Println(wd)
	http.Handle("/socket", websocket.Handler(s.HandleConnection))
	http.Handle("/", http.FileServer(http.Dir(wd)))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
