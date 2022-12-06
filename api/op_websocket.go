package api

import (
	"bytes"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/luoruofeng/dockermanagersingle/container"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type WebSocketHandle struct {
	wg          *sync.WaitGroup
	conn        *websocket.Conn
	receiveChan chan []byte
	sendChan    chan []byte
	oh          OpHandler
}

func NewWebSocketHandle(w http.ResponseWriter, r *http.Request, oh OpHandler) *WebSocketHandle {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return nil
	}

	var wg sync.WaitGroup

	receiveChan := make(chan []byte, 1024)
	sendChan := make(chan []byte, 1024)

	oh.setWebSocketChan(receiveChan, sendChan, &wg)

	return &WebSocketHandle{
		wg:          &wg,
		conn:        conn,
		receiveChan: receiveChan,
		sendChan:    sendChan,
		oh:          oh,
	}
}

func (wsh *WebSocketHandle) Run(cmds []string) {
	wsh.wg.Add(1)
	go wsh.readWs()
	wsh.wg.Add(1)
	go wsh.writeWs()
	wsh.wg.Add(1)
	wsh.oh.run(cmds)
	wsh.wg.Wait()
}

func (wsh *WebSocketHandle) readWs() {
	defer func() {
		log.Println("websocket:exit readWs")
		close(wsh.receiveChan)
		wsh.wg.Done()
	}()

	wsh.conn.SetReadLimit(maxMessageSize)
	wsh.conn.SetReadDeadline(time.Now().Add(pongWait))
	wsh.conn.SetPongHandler(func(string) error { wsh.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := wsh.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			return
		}
		message = append(bytes.TrimSpace(message), newline...)
		wsh.receiveChan <- message
	}
}

func (wsh *WebSocketHandle) writeWs() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		wsh.conn.Close()
		wsh.wg.Done()
		log.Println("websocket:exit writeWs")
	}()
	var mu sync.Mutex
	for {
		select {
		case message, ok := <-wsh.sendChan:
			mu.Lock()
			wsh.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// closed the channel.
				wsh.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := wsh.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
			// wsh.conn.WriteMessage(websocket.BinaryMessage, message)
			mu.Unlock()
		case <-ticker.C:
			wsh.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wsh.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}

}

type OpHandler interface {
	setWebSocketChan(receiveChan, sendChan chan []byte, wg *sync.WaitGroup)
	run(cmds []string)
}

type opContainerHandler struct {
	wg          *sync.WaitGroup
	containerId string
	receiveChan chan []byte
	sendChan    chan []byte
}

func newOpContainerHandler(containerId string) OpHandler {
	return &opContainerHandler{
		containerId: containerId,
	}
}

func (oh *opContainerHandler) setWebSocketChan(receiveChan, sendChan chan []byte, wg *sync.WaitGroup) {
	oh.receiveChan = receiveChan
	oh.sendChan = sendChan
	oh.wg = wg
}

func (oh *opContainerHandler) run(cmds []string) {
	defer func() {
		oh.wg.Done()
		log.Println("websocket:exit run")
	}()

	var waiter *types.HijackedResponse
	var err error

	if cmds == nil {
		waiter, err = container.GetCM().BashContainer(oh.containerId)
	} else {
		waiter, err = container.GetCM().BashContainerWithCmds(oh.containerId, cmds)
	}
	if err != nil {
		log.Println(err)
		return
	}

	defer waiter.Close()

	//output message
	go func() {
		defer func() {
			log.Println("websocket:exit run inner")
		}()

		bs := make([]byte, 4<<10)
		for {
			time.Sleep(time.Second * 1)
			n, err := waiter.Reader.Read(bs)
			if err != nil {
				log.Println(err)
				close(oh.sendChan)
				return
			}
			oh.sendChan <- bs[:n]
		}
	}()

	//input cmd
	for {
		mes, ok := <-oh.receiveChan
		if !ok {
			return
		}
		if strings.ToLower(string(mes)) == "exit" {
			return
		}
		_, err := waiter.Conn.Write(mes)
		if err != nil {
			log.Println(err)
			return
		}
	}
}
