package ws

import "github.com/frizinak/bitstamp/generic"

type (
	ChannelPrefix string
	Channel       string
	Event         string
)

func (c ChannelPrefix) ForCurrencyPair(pair generic.CurrencyPair) Channel {
	return Channel(string(c) + pair.String())
}

const (
	LiveTrades      ChannelPrefix = "live_trades_"
	LiveOrders      ChannelPrefix = "live_orders_"
	OrderBook       ChannelPrefix = "order_book_"
	DetailOrderBook ChannelPrefix = "detail_order_book_"
	DiffOrderBook   ChannelPrefix = "diff_order_book_"
)

const (
	TradeEvent        Event = "trade"
	OrderCreatedEvent Event = "order_created"
	OrderChangedEvent Event = "order_changed"
	OrderDeletedEvent Event = "order_deleted"
	DataEvent         Event = "data"
)
