package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Hub manages websocket clients and distributes realtime events.
type Hub struct {
	log *slog.Logger

	events    chan Event
	register  chan *Client
	unregister chan *Client

	clients map[*Client]bool
	mu      sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	running atomic.Bool
}

func NewHub(log *slog.Logger, buffer int) *Hub {
	if buffer <= 0 {
		buffer = 256
	}

	return &Hub{
		log:        log,
		events:     make(chan Event, buffer),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) EventsSink() chan<- Event {
	return h.events
}

func (h *Hub) Start(parent context.Context) error {
	if !h.running.CompareAndSwap(false, true) {
		return errors.New("websocket hub is already running")
	}

	h.ctx, h.cancel = context.WithCancel(parent)
	h.wg.Add(1)
	go h.run()

	h.log.Info("websocket hub started")
	return nil
}

func (h *Hub) Stop() {
	if !h.running.CompareAndSwap(true, false) {
		return
	}

	h.cancel()
	h.wg.Wait()
	h.log.Info("websocket hub stopped")
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) Register(conn *websocket.Conn) *Client {
	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
		id:   uuid.NewString(),
	}

	h.register <- client
	return client
}

func (h *Hub) run() {
	defer h.wg.Done()

	for {
		select {
		case <-h.ctx.Done():
			h.closeAllClients()
			return

		case client := <-h.register:
			h.addClient(client)
			h.log.Info("websocket client connected",
				slog.String("client_id", client.id),
				slog.Int("clients", h.ClientCount()),
			)

		case client := <-h.unregister:
			h.removeClient(client)
			h.log.Info("websocket client disconnected",
				slog.String("client_id", client.id),
				slog.Int("clients", h.ClientCount()),
			)

		case event := <-h.events:
			h.broadcast(event)
		}
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)
	close(client.send)
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.send)
	}
	clear(h.clients)
}

func (h *Hub) broadcast(event Event) {
	message, err := json.Marshal(event)
	if err != nil {
		h.log.Error("failed to marshal websocket event",
			slog.String("type", event.Type),
			slog.String("error", err.Error()),
		)
		return
	}

	slowClients := make([]*Client, 0)

	h.mu.RLock()
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			slowClients = append(slowClients, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range slowClients {
		h.log.Warn("websocket client too slow, disconnecting",
			slog.String("client_id", client.id),
		)
		h.unregister <- client
	}
}
