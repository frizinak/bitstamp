package api

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/frizinak/bitstamp/generic"
)

type TransactionType byte

func (t *TransactionType) UnmarshalJSON(d []byte) error {
	b := generic.ByteString(*t)
	if err := b.UnmarshalJSON(d); err != nil {
		return err
	}
	*t = TransactionType(b)
	return nil
}

const (
	Deposit            TransactionType = 0
	Withdrawal         TransactionType = 1
	MarketTrade        TransactionType = 2
	SubAccountTransfer TransactionType = 14
	ReferralReward     TransactionType = 32
)

type Transaction struct {
	DateTime generic.UTCDateString `json:"datetime"`
	ID       generic.Uint64String  `json:"id"`
	Type     TransactionType       `json:"type"`
	OrderID  generic.Uint64String  `json:"order_id"`

	Fee generic.Float64String `json:"fee"`

	USD    generic.Float64String `json:"usd"`
	EUR    generic.Float64String `json:"eur"`
	BTC    generic.Float64String `json:"btc"`
	XRP    generic.Float64String `json:"xrp"`
	GBP    generic.Float64String `json:"gbp"`
	LTC    generic.Float64String `json:"ltc"`
	ETH    generic.Float64String `json:"eth"`
	BCH    generic.Float64String `json:"bch"`
	XLM    generic.Float64String `json:"xlm"`
	PAX    generic.Float64String `json:"pax"`
	LINK   generic.Float64String `json:"link"`
	OMG    generic.Float64String `json:"omg"`
	USDC   generic.Float64String `json:"usdc"`
	BTCUSD generic.Float64String `json:"btc_usd"`
}

type Transactions struct {
	api *API

	List []Transaction
}

func (api *API) NewTransactions() *Transactions {
	return &Transactions{api: api, List: make([]Transaction, 0, 100)}
}

func (t *Transactions) Next() (int, error) {
	if len(t.List) == 0 {
		res, err := t.api.Transactions(100, true)
		if err != nil {
			return 0, err
		}
		*t = *res
		return len(t.List), nil
	}

	last := t.List[len(t.List)-1]
	res, err := t.api.TransactionsSinceID(last.ID.Value())
	if err != nil {
		return 0, err
	}
	if len(res.List) == 0 {
		return 0, nil
	}
	if res.List[0].ID == last.ID { // BRUH, why
		res.List = res.List[1:]
	}

	t.List = append(t.List, res.List...)
	return len(res.List), nil
}

func (api *API) transactions(params url.Values) (*Transactions, error) {
	var r io.Reader
	if params != nil {
		r = strings.NewReader(params.Encode())
	}
	res, err := api.Post(api.URL("user_transactions"), r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)
	t := &Transactions{api: api, List: make([]Transaction, 0, 100)}
	return t, dec.Decode(&t.List)
}

func (api *API) Transactions(limit int, asc bool) (*Transactions, error) {
	sort := "desc"
	if asc {
		sort = "asc"
	}
	return api.transactions(
		url.Values{
			"limit": {strconv.Itoa(limit)}, "sort": {sort},
		},
	)
}

func (api *API) TransactionsSinceID(id uint64) (*Transactions, error) {
	return api.transactions(
		url.Values{
			"since_id": {strconv.FormatUint(id, 10)},
		},
	)
}
