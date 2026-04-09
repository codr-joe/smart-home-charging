package server

import (
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/smart-charging/api/internal/energy"
)

type handler struct {
	repo *energy.Repository
	hub  *Hub
}

func newHandler(repo *energy.Repository, hub *Hub) *handler {
	return &handler{repo: repo, hub: hub}
}

// getCurrent returns the most recent energy reading.
func (h *handler) getCurrent(c *fiber.Ctx) error {
	reading, err := h.repo.Latest(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not fetch current reading")
	}
	return c.JSON(reading)
}

// getHistory returns readings within a time window.
// Query params: from (RFC3339), to (RFC3339), limit (int, default 500, max 5000).
func (h *handler) getHistory(c *fiber.Ctx) error {
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now
	limit := 500

	if v := c.Query("from"); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid from timestamp, expected RFC3339")
		}
		from = parsed
	}
	if v := c.Query("to"); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid to timestamp, expected RFC3339")
		}
		to = parsed
	}
	if v := c.QueryInt("limit", 500); v > 0 {
		if v > 5000 {
			v = 5000
		}
		limit = v
	}
	if from.After(to) {
		return fiber.NewError(fiber.StatusBadRequest, "from must be before to")
	}
	readings, err := h.repo.History(c.Context(), from, to, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not fetch history")
	}
	return c.JSON(readings)
}

// stream upgrades the connection to WebSocket and fans out live readings.
func (h *handler) stream(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}
	c.Locals("hub", h.hub)
	return websocket.New(func(conn *websocket.Conn) {
		hub := conn.Locals("hub").(*Hub)
		cl := &client{conn: conn, send: make(chan []byte, 64)}
		hub.register <- cl
		defer func() { hub.unregister <- cl }()

		done := make(chan struct{})
		go func() {
			defer close(done)
			for msg := range cl.send {
				if err := conn.WriteMessage(1, msg); err != nil {
					return
				}
			}
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
		<-done
	})(c)
}
