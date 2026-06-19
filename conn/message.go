package conn

type IncomingMessage struct {
	Action string `json:"action"`
	Pool   string `json:"pool"`
}

type OutgoingMessage struct {
	Type      string  `json:"type"`
	Pool      string  `json:"pool"`
	Price     float64 `json:"price,omitempty"`
	Timestamp int64   `json:"timestamp,omitempty"`
	Error     string  `json:"error,omitempty"`
}
