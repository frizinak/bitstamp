package api

import (
	"encoding/json"
	"strings"

	"github.com/frizinak/bitstamp/generic"
)

type Balance map[string]generic.Float64String

func (b Balance) ForCurrency(c generic.Currency) Balance {
	n := make(Balance, 3)
	for k, v := range b {
		name := c.String()
		if len(k) > len(name) && strings.HasPrefix(k, name) && k[len(name)] == '_' {
			n[k[len(name)+1:]] = v
		}
	}

	return n
}

func (api *API) BalancePair(pair generic.CurrencyPair) (Balance, error) {
	return api.balance(api.URL("balance", pair.String()))
}

func (api *API) Balance() (Balance, error) {
	return api.balance(api.URL("balance"))
}

func (api *API) balance(url string) (Balance, error) {
	b := make(Balance, 20)
	res, err := api.Post(url, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&b); err != nil {
		return nil, err
	}

	return b, err
}
