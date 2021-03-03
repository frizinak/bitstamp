package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/frizinak/bitstamp/generic"
)

type Order interface {
	URL(*API) string
	Params(url.Values)
}

type LimitOrder struct {
	Action     string
	Pair       generic.CurrencyPair
	Amount     float64
	Price      float64
	LimitPrice float64
	Daily      bool
	IOC        bool
	FOK        bool
}

func NewLimitBuy(pair generic.CurrencyPair, amount, price float64) LimitOrder {
	return LimitOrder{Action: "buy", Pair: pair, Amount: amount, Price: price}
}
func NewLimitSell(pair generic.CurrencyPair, amount, price float64) LimitOrder {
	return LimitOrder{Action: "sell", Pair: pair, Amount: amount, Price: price}
}

func (o LimitOrder) String() string {
	return fmt.Sprintf("%s %.8f@%.2f = %.2f", o.Action, o.Amount, o.Price, o.Amount*o.Price)
}

func (o LimitOrder) URL(api *API) string { return api.URL(o.Action, o.Pair.String()) }
func (o LimitOrder) Params(p url.Values) {
	p.Set("amount", strconv.FormatFloat(o.Amount, 'f', -1, 64))
	p.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
	if o.LimitPrice != 0 {
		p.Set("limit_price", strconv.FormatFloat(o.LimitPrice, 'f', -1, 64))
	}
	if o.Daily {
		p.Set("daily_order", "True")
	}
	if o.Daily {
		p.Set("ioc_order", "True")
	}
	if o.Daily {
		p.Set("fok_order", "True")
	}
}

type SimpleOrder struct {
	Action        string
	Type          string
	Pair          generic.CurrencyPair
	Amount        float64
	AmountCounter bool
}

func (o SimpleOrder) URL(api *API) string { return api.URL(o.Action, o.Type, o.Pair.String()) }
func (o SimpleOrder) Params(p url.Values) {
	p.Set("amount", strconv.FormatFloat(o.Amount, 'f', -1, 64))
	if o.AmountCounter {
		p.Set("amount_in_counter", "True")
	}
}

func NewBuyOrder(pair generic.CurrencyPair, amount float64) SimpleOrder {
	return SimpleOrder{Action: "buy", Type: "market", Pair: pair, Amount: amount}
}

func NewSellOrder(pair generic.CurrencyPair, amount float64) SimpleOrder {
	return SimpleOrder{Action: "sell", Type: "market", Pair: pair, Amount: amount}
}

func NewInstantBuyOrder(pair generic.CurrencyPair, amount float64) SimpleOrder {
	return SimpleOrder{Action: "buy", Type: "instant", Pair: pair, Amount: amount}
}

func NewInstantSellOrder(pair generic.CurrencyPair, amount float64) SimpleOrder {
	return SimpleOrder{Action: "sell", Type: "instant", Pair: pair, Amount: amount}
}

type OrderResponse struct {
	ID generic.Uint64String `json:"id"`
	Status
}

func (api *API) Place(order Order) (OrderResponse, error) {
	var o OrderResponse
	params := url.Values{}
	u := order.URL(api)
	order.Params(params)
	res, err := api.Post(u, strings.NewReader(params.Encode()))
	if err != nil {
		return o, err
	}
	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&o); err != nil {
		return o, err
	}

	return o, o.Error()
}
