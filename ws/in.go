package ws

import (
	"encoding/json"

	"github.com/frizinak/bitstamp/generic"
)

type Message struct {
	Channel Channel              `json:"channel"`
	Event   Event                `json:"event"`
	Data    generic.EmbeddedJSON `json:"data"`
}

func (m Message) DataLiveTrade() (LiveTrade, error) {
	l := &LiveTrade{}
	return *l, json.Unmarshal(m.Data, l)
}

type LiveTrade struct {
	ID          generic.Uint64String `json:"id"`
	SellOrderID generic.Uint64String `json:"sell_order_id"`
	BuyOrderID  generic.Uint64String `json:"buy_order_id"`

	Type generic.ByteString `json:"type"`

	Amount       generic.Float64String `json:"amount"`
	AmountString string                `json:"amount_str"` // bruh

	Price       generic.Float64String `json:"price"`
	PriceString string                `json:"price_str"`

	Timestamp      generic.UnixString      `json:"timestamp"`
	MicroTimestamp generic.UnixMicroString `json:"microtimestamp"`
}
