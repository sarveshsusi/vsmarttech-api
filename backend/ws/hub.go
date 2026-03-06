package ws

type Hub struct {
	clients map[string][]chan string
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string][]chan string)}
}

func (h *Hub) Notify(userID string, msg string) {
	for _, ch := range h.clients[userID] {
		ch <- msg
	}
}
