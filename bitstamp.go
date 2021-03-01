package bitstamp

import (
	"sync"
	"time"

	"github.com/frizinak/bitstamp/api"
	"github.com/frizinak/bitstamp/generic"
	"github.com/frizinak/bitstamp/ws"
	"golang.org/x/net/websocket"
)

type Bitstamp struct {
	API *api.API
	WS  *ws.Client

	l       sync.Mutex
	looping bool

	lEvent     sync.RWMutex
	eventChans []chan event
}

func New(api *api.API, ws *ws.Client) *Bitstamp {
	return &Bitstamp{API: api, WS: ws, eventChans: make([]chan event, 0)}
}

func NewDefaults(apiKey, apiSecret string) (*Bitstamp, error) {
	api := api.New(apiKey, apiSecret, "https://www.bitstamp.net/api/v2", nil)
	wsc, err := websocket.NewConfig("wss://ws.bitstamp.net", "https://ws.bitstamp.net")
	if err != nil {
		return nil, err
	}
	ws := ws.New(wsc)
	return New(api, ws), nil
}

type event struct {
	ws.Message
	err error
}

func (b *Bitstamp) eventLoop() {
	b.l.Lock()
	if b.looping {
		b.l.Unlock()
		return
	}
	b.looping = true
	b.l.Unlock()
	go func() {
		for {
			msg, err := b.WS.Read()
			b.lEvent.RLock()
			for _, c := range b.eventChans {
				c <- event{Message: msg, err: err}
			}
			b.lEvent.RUnlock()
		}
	}()
	return
}

func (b *Bitstamp) subscribe() chan event {
	ch := make(chan event, 10)
	b.lEvent.Lock()
	b.eventChans = append(b.eventChans, ch)
	b.lEvent.Unlock()
	return ch
}

func (b *Bitstamp) unsubscribe(ch chan event) {
	b.lEvent.Lock()
	ix := -1
	for i, c := range b.eventChans {
		if c == ch {
			ix = i
			break
		}
	}
	if ix != -1 {
		b.eventChans = append(b.eventChans[:ix], b.eventChans[ix+1:]...)
	}
	b.lEvent.Unlock()
}

type Transaction struct {
	api.Transaction
	Values map[generic.Currency]float64
}

func (b *Bitstamp) Transactions() ([]Transaction, error) {
	t := b.API.NewTransactions()
	for {
		n, err := t.Next()
		if err != nil {
			return nil, err
		}
		if n == 0 {
			break
		}
	}

	list := make([]Transaction, len(t.List))
	for i, t := range t.List {
		list[i] = Transaction{Transaction: t}
		list[i].Values = map[generic.Currency]float64{
			USD:  t.USD.Value(),
			EUR:  t.EUR.Value(),
			BTC:  t.BTC.Value(),
			XRP:  t.XRP.Value(),
			GBP:  t.GBP.Value(),
			LTC:  t.LTC.Value(),
			ETH:  t.ETH.Value(),
			BCH:  t.BCH.Value(),
			XLM:  t.XLM.Value(),
			PAX:  t.PAX.Value(),
			LINK: t.LINK.Value(),
			OMG:  t.OMG.Value(),
			USDC: t.USDC.Value(),
		}
	}

	return list, nil
}

type Trade struct {
	Date   time.Time
	ID     uint64
	Price  float64
	Amount float64
	Type   api.TradeType
}

func (b *Bitstamp) TradesLive(
	history api.TradeHistory,
	pair generic.CurrencyPair,
	trades chan<- Trade,
) error {
	if history != api.TradesHistoryNone {
		r, err := b.API.Trades(history, pair)
		if err != nil {
			return err
		}

		for i := len(r.List) - 1; i >= 0; i-- {
			d := r.List[i]
			trades <- Trade{
				Date:   d.Date.Value(),
				ID:     d.ID.Value(),
				Price:  d.Price.Value(),
				Amount: d.Amount.Value(),
				Type:   d.Type,
			}
		}
	}

	ch := b.subscribe()
	defer b.unsubscribe(ch)
	b.eventLoop()

	channel := ws.LiveTrades.ForCurrencyPair(pair)
	if err := b.WS.Subscribe(channel); err != nil {
		return err
	}

	for e := range ch {
		if e.err != nil {
			return e.err
		}

		if e.Channel == channel && e.Event == ws.TradeEvent {
			d, err := e.DataLiveTrade()
			if err != nil {
				return err
			}

			trades <- Trade{
				Date:   d.MicroTimestamp.Value(),
				ID:     d.ID.Value(),
				Price:  d.Price.Value(),
				Amount: d.Amount.Value(),
				Type:   api.TradeType(d.Type.Value()),
			}
		}
	}

	return nil
}
