package conn

type IncomingMessage struct {
	Action string `json:"action"`
	Pool   string `json:"pool"`
}

type OutgoingMessage struct {
	Type      string  `json:"type"`
	Pool      string  `json:"pool,omitempty"`
	Price     float64 `json:"price,omitempty"`
	Timestamp int64   `json:"timestamp,omitempty"`

	Kind       string  `json:"kind,omitempty"`
	VolumeUSD  float64 `json:"volumeUsd,omitempty"`
	VolumeBase float64 `json:"volumeBase,omitempty"`
	VolumeSOL  float64 `json:"volumeSol,omitempty"`
	PriceUSD   float64 `json:"priceUsd,omitempty"`
	TxHash     string  `json:"txHash,omitempty"`
	Maker      string  `json:"maker,omitempty"`
}
