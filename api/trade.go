package api

import (
	"encoding/json"
	"net/url"

	"github.com/frizinak/bitstamp/generic"
)

type TradeHistory string

const (
	TradesHistoryNone   TradeHistory = ""
	TradesHistoryMinute TradeHistory = "minute"
	TradesHistoryHour   TradeHistory = "hour"
	TradesHistoryDay    TradeHistory = "day"
)

type TradeType byte

func (t *TradeType) UnmarshalJSON(d []byte) error {
	b := generic.ByteString(*t)
	if err := b.UnmarshalJSON(d); err != nil {
		return err
	}
	*t = TradeType(b)
	return nil
}

const (
	Buy  TradeType = 0
	Sell TradeType = 1
)

type Trade struct {
	Date   generic.UnixString    `json:"date"`
	ID     generic.Uint64String  `json:"tid"`
	Price  generic.Float64String `json:"price"`
	Amount generic.Float64String `json:"amount"`
	Type   TradeType             `json:"type"`
}

type Trades struct {
	List []Trade
}

func (api *API) Trades(history TradeHistory, pair generic.CurrencyPair) (Trades, error) {
	var t Trades
	u, err := url.Parse(api.URL("transactions", pair.String()))
	if err != nil {
		return t, err
	}
	q := u.Query()
	q.Set("time", string(history))
	u.RawQuery = q.Encode()

	res, err := api.Get(u.String(), nil)
	if err != nil {
		return t, err
	}
	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)
	t.List = make([]Trade, 0, 100)
	return t, dec.Decode(&t.List)
}
