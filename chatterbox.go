package main

import (
	"code.google.com/p/go.net/websocket"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// TODO: Add sync locking!

type Message struct {
	Type string `json:"type"`
	Src  string `json:"src"`
	Room string `json:"room"`
	Msg  string `json:"msg"`
}

type Client struct {
	id     string
	server *Server
	ws     *websocket.Conn
	logger *log.Logger
}

type Room struct {
	id      string
	clients map[string]*Client
	queue   []string
}

type Server struct {
	rooms  map[string]*Room
	logger *log.Logger
}

func NewRoom(id string) *Room {
	return &Room{id: id, clients: make(map[string]*Client)}
}

func NewServer() *Server {
	return &Server{
		rooms:  make(map[string]*Room),
		logger: log.New(os.Stdout, "Chatterbox: ", log.LstdFlags),
	}
}

var AlreadyInRoomError = errors.New("Already in room.")
var ClientNotInRoomError = errors.New("Client not in room.")
var RoomFullError = errors.New("Room full.")
var RoomNotFoundError = errors.New("Room not found.")
var ExistingRoomError = errors.New("Existing room not found.")

func (r *Room) Add(c *Client) error {
	if _, ok := r.clients[c.Id()]; ok {
		return AlreadyInRoomError
	}

	if len(r.clients) >= 2 {
		return RoomFullError
	} else if len(r.clients) == 1 {
		for _, msg := range r.queue {
			c.Send(msg)
		}
		r.queue = nil
	}

	r.clients[c.Id()] = c
	return nil
}

func (r *Room) Remove(c *Client) error {
	if _, ok := r.clients[c.Id()]; !ok {
		return ClientNotInRoomError
	}
	delete(r.clients, c.Id())
	if len(r.clients) == 0 {
		r.queue = nil
	}
	return nil
}

func (r *Room) Id() string {
	return r.id
}

func (r *Room) Size() int {
	return len(r.clients)
}

func (r *Room) Send(clientId string, msg string) error {
	if _, ok := r.clients[clientId]; !ok {
		return ClientNotInRoomError
	}

	if len(r.clients) < 2 {
		r.queue = append(r.queue, msg)
		return nil
	}

	for cid, c := range r.clients {
		if cid == clientId {
			continue
		}
		c.Send(msg)
	}
	return nil
}

func NewClient(s *Server, ws *websocket.Conn) *Client {
	return &Client{
		server: s,
		ws:     ws,
		logger: log.New(os.Stdout, "Client Unk: ", log.LstdFlags),
	}
}

func (c *Client) setId(id string) {
	c.id = id
	logPfx := fmt.Sprint("Client ", id, ": ")
	c.logger = log.New(os.Stdout, logPfx, log.LstdFlags)
}

func (c *Client) Id() string {
	return c.id
}

func (c *Client) sendErr(msg string) {
	c.logit("Sending error: ", msg)
	websocket.JSON.Send(c.ws, Message{Type: "error", Msg: msg})
}

func (c *Client) Send(msg string) {
	websocket.JSON.Send(c.ws, Message{Type: "msg", Msg: msg})
}

func (c *Client) Listen() {
	c.logit("Entering read loop...")
	for {
		var msg Message
		err := websocket.JSON.Receive(c.ws, &msg)
		if err == io.EOF {
			break
		} else if err != nil {
			c.logit("Error reading from connection: ", err)
			continue
		}
		c.logit("Recv message of length: ", msg)
		if c.id != "" && c.id != msg.Src {
			c.sendErr("Invalid 'src' ID set.")
			continue
		}
		if msg.Src == "" {
			c.sendErr("No 'src' ID set.")
			continue
		}
		if c.id == "" {
			c.setId(msg.Src)
		}

		switch msg.Type {
		case "join":
			c.logit("Join room: ", msg.Room)
			room, err := c.server.Room(msg.Room)
			if err == RoomNotFoundError {
				c.logit("Creating room ", msg.Room)
				room = NewRoom(msg.Room)
				c.server.AddRoom(room)
			}
			err = room.Add(c)
			if room.Size() == 1 {
				c.Send("{\"type\":\"initiator\",\"value\":true}")
			}
			switch err {
			case nil:
				defer func() {
					c.server.Leave(msg.Room, c)
					c.logit("Leaving room ", msg.Room)
				}()
				break
			case RoomFullError:
				c.sendErr(err.Error())
				continue
			}
			break
		case "msg":
			if msg.Room == "" {
				c.sendErr("Room not specified.")
				continue
			}
			if msg.Msg == "" {
				c.sendErr("Empty message.")
				continue
			}
			c.server.Send(msg.Room, msg.Src, msg.Msg)
		}
	}
	c.logit("Closing websocket.")
}

func (c *Client) logit(args ...interface{}) {
	c.logger.Print(args...)
}

func (s *Server) logit(args ...interface{}) {
	s.logger.Print(args...)
}

func (s *Server) Join(roomId string, c *Client) error {
	room, ok := s.rooms[roomId]
	if !ok {
		room = NewRoom(roomId)
		s.rooms[roomId] = room
	}

	err := room.Add(c)

	return err
}

func (s *Server) AddRoom(r *Room) error {
	if _, ok := s.rooms[r.Id()]; ok {
		return ExistingRoomError
	}

	s.rooms[r.Id()] = r
	return nil
}

func (s *Server) Room(id string) (*Room, error) {
	room, ok := s.rooms[id]
	if !ok {
		return nil, RoomNotFoundError
	}

	return room, nil
}

func (s *Server) Leave(roomId string, c *Client) {
	room, ok := s.rooms[roomId]
	if !ok {
		s.logit("Invalid room ", roomId)
	}
	if err := room.Remove(c); err != nil {
		s.logit("Client ", c.Id(), " not in room ", roomId)
	}
}

func (s *Server) Send(roomId string, clientId string, msg string) error {
	room, ok := s.rooms[roomId]
	if !ok {
		return RoomNotFoundError
	}
	return room.Send(clientId, msg)
}

func (s *Server) HandleConnection(ws *websocket.Conn) {
	log.Println("Handling new websocket connection: ", ws)
	c := NewClient(s, ws)
	c.Listen()
	// s.AddClient(c)
	// c.Listen()
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

func main() {
	wd, _ := os.Getwd()
	s := NewServer()
	// go s.Listen()
	fmt.Println("Starting server on port 8080...")
	fmt.Println(wd)
	http.Handle("/socket", websocket.Handler(s.HandleConnection))
	http.Handle("/", http.FileServer(http.Dir(wd)))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

// type Client

// // 0 should represent an unknown client.
// var maxClientId = 0

// type Client struct {
// 	id      int
// 	ws      *websocket.Conn
// 	server  *Server
// 	offer   string
// 	logger  *log.Logger
// 	writeCh chan *Message
// 	doneCh  chan bool
// }

// func NewClient(s *Server, ws *websocket.Conn) *Client {
// 	maxClientId++
// 	id := maxClientId
// 	logPfx := fmt.Sprint("connection ", id, ": ")
// 	connection := &Client{
// 		id,
// 		ws,
// 		s,
// 		"",
// 		log.New(os.Stdout, logPfx, log.LstdFlags),
// 		make(chan *Message),
// 		make(chan bool)}

// 	return connection
// }

// func (c *Client) logit(args ...interface{}) {
// 	c.logger.Print(args...)
// }

// func (c *Client) Write(msg *Message) {
// 	c.writeCh <- msg
// }

// func (c *Client) listenWrite() {
// 	c.logit("entering write loop...")
// 	for {
// 		select {
// 		case msg := <-c.writeCh:
// 			c.logit("writing message: ", msg)
// 			websocket.JSON.Send(c.ws, msg)
// 		case <-c.doneCh:
// 			c.logit("exiting write loop.")
// 			c.server.DelClient(c)
// 			c.doneCh <- true
// 			return
// 		}
// 	}
// }

// func (c *Client) listenRead() {
// 	c.logit("entering read loop...")
// 	for {
// 		select {
// 		case <-c.doneCh:
// 			c.logit("exiting read loop.")
// 			c.server.DelClient(c)
// 			c.doneCh <- true
// 			return
// 		default:
// 			var message string
// 			err := websocket.Message.Receive(c.ws, &message)
// 			if err == io.EOF {
// 				c.logit("closing connection")
// 				c.doneCh <- true
// 			} else if err != nil {
// 				c.logit("error reading from connection: ", err)
// 			} else {
// 				c.logit("recv message of length: ", len(message))
// 			}
// 		}
// 	}
// }

// func (c *Client) Listen() {
// 	go c.listenWrite()
// 	c.listenRead()
// }

// type Server struct {
// 	clients     map[int]*Client
// 	addClientCh chan *Client
// 	delClientCh chan *Client
// }

// func NewServer() *Server {
// 	return &Server{
// 		make(map[int]*Client),
// 		make(chan *Client),
// 		make(chan *Client),
// 	}
// }

// func (s *Server) HandleConnection(ws *websocket.Conn) {
// 	log.Println("Handling new websocket connection: ", ws)
// 	c := NewClient(s, ws)
// 	s.AddClient(c)
// 	c.Listen()
// 	// for cid, c := range s.clients {
// 	// 	if c.id != cid && c.offer != "" {
// 	// 		log.Print("sending offer ", cid, " - > ", c.id)
// 	// 		websocket.Message.Send(c.ws, c.offer)
// 	// 	}
// 	// }
// 	// go c.Listen()
// 	// for {
// 	// 	select {
// 	// 	case offer := <-c.offerCh:
// 	// 		for cid, c := range s.clients {
// 	// 			if cid != c.id {
// 	// 				log.Print("sending offer ", c.id, " -> ", cid)
// 	// 				websocket.Message.Send(c.ws, offer)
// 	// 			}
// 	// 		}
// 	// 	case _ = <-c.doneCh:
// 	// 		delete(s.clients, c.id)
// 	// 	}
// 	// }
// }

// func (s *Server) AddClient(c *Client) {
// 	s.addClientCh <- c
// }

// func (s *Server) DelClient(c *Client) {
// 	s.delClientCh <- c
// }

// // Listens to server channels to take action. This way all actions on
// // the Server object happens in a single thread.
// func (s *Server) Listen() {
// 	for {
// 		select {
// 		case c := <-s.delClientCh:
// 			log.Print("Deleting client: ", c.id)
// 			delete(s.clients, c.id)
// 			for oId, o := range s.clients {
// 				log.Print("checking ", oId)
// 				if c.id != oId {
// 					log.Print("sending to ", oId)
// 					o.Write(&Message{0, oId, "", "", 0, 0, c.id})
// 				}
// 			}
// 		case c := <-s.addClientCh:
// 			log.Print("Adding client: ", c.id)
// 			s.clients[c.id] = c
// 			log.Print("clients: ", len(s.clients))
// 			log.Print("Telling ", c.id, " its true identity.")
// 			c.Write(&Message{0, c.id, "", "", c.id, 0, 0})
// 			for oId, o := range s.clients {
// 				log.Print("checking ", oId)
// 				if c.id != oId {
// 					log.Print("sending to ", oId)
// 					o.Write(&Message{0, oId, "", "", 0, c.id, 0})
// 				}
// 			}

// 			// log.Print("Sending client ", c.id, " list of all other clients")
// 			// clientIds := make([]int, 0, len(s.clients))
// 			// for otherId, _ := range s.clients {
// 			// 	if c.id != otherId {
// 			// 		clientIds = append(clientIds, otherId)
// 			// 	}
// 			// }
// 			// msg := Message{c.id, "", "", clientIds}
// 			// msgJson := json.Marshall(msg)
// 			// c.Write("clientListMessage")
// 		}
// 	}
// }
