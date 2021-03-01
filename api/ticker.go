package api

import (
	"encoding/json"
	"errors"

	"github.com/frizinak/bitstamp/generic"
)

type TickerResult struct {
	CurrencyPair generic.CurrencyPair

	Last generic.Float64String `json:"last"`
	High generic.Float64String `json:"high"`
	Low  generic.Float64String `json:"low"`

	VWAP   generic.Float64String `json:"vwap"`
	Volume generic.Float64String `json:"volume"`

	Open generic.Float64String `json:"open"`

	Bid generic.Float64String `json:"bid"`
	Ask generic.Float64String `json:"ask"`

	Time generic.UnixString `json:"timestamp"`
}

type TickerInterval byte

const (
	TickerDaily TickerInterval = iota
	TickerHourly
)

func (api *API) Ticker(pair generic.CurrencyPair, interval TickerInterval) (TickerResult, error) {
	var t TickerResult
	t.CurrencyPair = pair

	var e string
	switch interval {
	case TickerDaily:
		e = "ticker"
	case TickerHourly:
		e = "ticker_hour"
	default:
		return t, errors.New("invalid ticker interval")
	}

	res, err := api.Get(api.URL(e, pair.String()), nil)
	if err != nil {
		return t, err
	}

	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&t); err != nil {
		return t, err
	}

	return t, err
}

func (api *API) TickerHourly(pair generic.CurrencyPair) (TickerResult, error) {
	return api.Ticker(pair, TickerHourly)
}

func (api *API) TickerDaily(pair generic.CurrencyPair) (TickerResult, error) {
	return api.Ticker(pair, TickerDaily)
}
