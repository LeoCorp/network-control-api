package websocket

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	hub *Hub
	log *slog.Logger
}

func NewHandler(hub *Hub, log *slog.Logger) *Handler {
	return &Handler{hub: hub, log: log}
}

// Serve upgrades the connection and streams realtime monitoring events.
func (h *Handler) Serve(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Warn("websocket upgrade failed", slog.String("error", err.Error()))
		return
	}

	client := h.hub.Register(conn)
	go client.writePump()
	go client.readPump()
}
