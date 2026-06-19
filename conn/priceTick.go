package conn

import "time"

type PriceTick struct {
	Pool      string
	Price     float64
	Timestamp time.Time
}
