package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/frizinak/bitstamp"
	"github.com/frizinak/bitstamp/api"
	"github.com/frizinak/bitstamp/generic"
	"github.com/google/shlex"
	"github.com/vdobler/chart"
	"github.com/vdobler/chart/txtg"
)

const dateFormat = "2006-01-02 15:04:05"

type action byte

const (
	actionLive action = iota
	actionCurrent
	actionBalance
	actionTransactions
	actionCurrencies
)

type VWAP struct {
	Time   time.Time
	Value  float64
	Volume float64
	Count  float64
}

type VWAPS []VWAP

func (v VWAPS) Add(t time.Time, price, volume float64) VWAPS {
	prev := len(v) - 1
	if prev < 0 || t.Sub(v[prev].Time) > time.Minute*5 {
		vw := VWAP{t, price, volume, 1}
		return append(v, vw)
	}

	l := v[prev]
	l.Count++
	l.Value += price
	l.Volume += volume
	v[prev] = l

	return v
}

func (v VWAPS) Range(from, until time.Time) VWAPS {
	min, max := 0, len(v)
	for i, v := range v {
		if v.Time.Before(from) && i > min {
			min = i
		}

		if v.Time.After(until) && i < max {
			max = i
			break
		}
	}

	return v[min:max]
}

func (v VWAPS) Value() float64 {
	var n, vol float64
	for _, v := range v {
		value, volume := v.Value/v.Count, v.Volume/v.Count
		n += value * volume
		vol += volume
	}
	if vol == 0 {
		return 0
	}

	return n / vol
}

type Alarm struct {
	GT    bool
	Value float64
}

func (a Alarm) Check(prev, current float64) bool {
	return ((a.GT && current >= a.Value) || (!a.GT && current < a.Value)) &&
		((a.GT && prev < a.Value) || (!a.GT && prev >= a.Value))
}

func (a Alarm) String() string {
	sym := "<"
	if a.GT {
		sym = ">"
	}

	return fmt.Sprintf("%s %.8f", sym, a.Value)
}

type Alarms []Alarm

func (a Alarms) Check(prev, current float64) Alarms {
	n := make(Alarms, 0)
	for _, a := range a {
		if a.Check(prev, current) {
			n = append(n, a)
		}
	}
	return n
}

type flagAlarms []string

func (a *flagAlarms) String() string { return "alarms" }

func (a *flagAlarms) Set(value string) error {
	*a = append(*a, value)
	return nil
}

var spaceRE = regexp.MustCompile(`\s+`)

func (a flagAlarms) Parse() (Alarms, error) {
	n := make([]Alarm, 0, len(a))
	for _, a := range a {
		a = spaceRE.ReplaceAllString(a, "")
		if len(a) < 2 {
			return n, errors.New("invalid alarm")
		}
		if a[0] != '<' && a[0] != '>' {
			return n, errors.New("invalid alarm, missing comparison symbol (> or <)") // >_<
		}

		alarm := Alarm{GT: a[0] == '>'}
		var err error
		alarm.Value, err = strconv.ParseFloat(a[1:], 32)
		if err != nil {
			return n, err
		}
		n = append(n, alarm)
	}

	return n, nil
}

func termSize() (int, int) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	res, _ := cmd.Output()
	out := strings.Fields(strings.TrimSpace(string(res)))
	if len(out) != 2 {
		return 0, 0
	}
	x, _ := strconv.Atoi(out[1])
	y, _ := strconv.Atoi(out[0])

	return x, y
}

func exit(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

func live(pair generic.CurrencyPair, alarmCmd string, alarms Alarms, nograph bool, truncate time.Duration) error {
	notify := func(price float64, alarm Alarm) {}
	if alarmCmd == "" && len(alarms) != 0 {
		return errors.New("no alarm command set")
	}

	if alarmCmd != "" {
		cmd, err := shlex.Split(alarmCmd)
		if err != nil {
			return err
		}
		notify = func(price float64, alarm Alarm) {
			rcmd := make([]string, len(cmd))
			copy(rcmd, cmd)
			for i := range rcmd {
				rcmd[i] = strings.ReplaceAll(rcmd[i], "%p", strconv.FormatFloat(price, 'f', -1, 64))
				rcmd[i] = strings.ReplaceAll(rcmd[i], "%a", alarm.String())
			}
			cmd := exec.Command(rcmd[0], rcmd[1:]...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		}
	}
	type Value struct {
		t time.Time
		v float64
	}
	trades := make(chan bitstamp.Trade, 1)

	client, err := bitstamp.NewDefaults("", "")
	if err != nil {
		return err
	}
	errs := make(chan error, 1)
	go func() {
		errs <- client.TradesLive(
			api.TradesHistoryDay,
			pair,
			trades,
		)
	}()

	tradePoints := make([]chart.EPoint, 0)

	vwap := make(VWAPS, 0)
	vwapPointsFull := make([]chart.EPoint, 0)
	vwapPoints := make([]chart.EPoint, 0)
	var lastVWAP time.Time
	vwapInterval := time.Hour * 6

	var value, lastValue Value

	type notification struct {
		price float64
		alarm Alarm
	}
	notifications := make(chan notification, 100)
	go func() {
		var last time.Time
		for n := range notifications {
			if time.Since(last) > time.Second*5 {
				notify(n.price, n.alarm)
				last = time.Now()
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig
		errs <- nil
	}()

	var pingTime time.Time
	var pingValue float64
	buf := bytes.NewBuffer(make([]byte, 1024*180))

	const clr = "\033[2J"
	const clrLine = "\033[K"
	const cursorHome = "\033[H"
	const cursorBOL = "\033[0G"
	const bg = "\033[40;1m"
	const rst = "\033[0m"
	const clrGreen = "\033[30;41m"
	const clrRed = "\033[30;42m"

	ignoreBefore := time.Now().Add(-truncate)
	lastUpdate := time.Now()
	refreshRate := time.Millisecond * 25
	for {
	sel:
		for {
			select {
			case err := <-errs:
				os.Stdout.WriteString(clr)
				return err
			case trade := <-trades:
				if trade.Live {
					results := alarms.Check(value.v, trade.Price)
					for _, a := range results {
						notifications <- notification{trade.Price, a}
					}
				} else if trade.Date.Before(ignoreBefore) {
					continue
				}

				value = Value{trade.Date, trade.Price}
				vwap = vwap.Add(trade.Date, trade.Price, trade.Amount)
				tradePoints = append(tradePoints, chart.EPoint{X: float64(trade.Date.Unix()), Y: trade.Price})
				rounded := value.t.Truncate(vwapInterval)
				if lastVWAP != rounded {
					lastVWAP = rounded
					v := vwap.Range(rounded.Add(-vwapInterval), rounded).Value()
					if v != 0 {
						vwapPoints = append(vwapPoints, chart.EPoint{X: float64(rounded.Unix()), Y: v})
					}
				}

				now := time.Now()
				vwapPointsFull = append(vwapPoints, chart.EPoint{X: float64(now.Unix()), Y: vwap.Range(rounded, now).Value()})
				since := time.Since(lastUpdate)
				refreshRate = time.Second
				if trade.Live && since > time.Millisecond*25 {
					break sel
				}
			case <-time.After(refreshRate):
				break sel
			}
		}

		now := time.Now()
		lastUpdate = now

		termX, termY := termSize()
		out := cursorBOL
		if termX > 20 && termY > 8 && !nograph {
			tgr := txtg.New(termX, termY-2)
			p := chart.ScatterChart{
				Key:    chart.Key{Hide: true, Cols: 3, Pos: "otc", Border: -1},
				YRange: chart.Range{},
				XRange: chart.Range{
					Time: true,
					MinMode: chart.RangeMode{
						Fixed:  true,
						TValue: now.Add(-truncate),
					},
				},
			}

			const symbol4 = '█'
			const symbol3 = '▓'
			const symbol2 = '▒'
			const symbol1 = '░'
			if len(tradePoints) > 0 {
				p.AddData("VWAP", vwapPointsFull, chart.PlotStyleLines, chart.Style{Symbol: symbol1})
				p.AddData("Trades", tradePoints, chart.PlotStylePoints, chart.Style{Symbol: symbol2})
				p.AddData("Now", tradePoints[len(tradePoints)-1:], chart.PlotStylePoints, chart.Style{Symbol: symbol4})
			}

			p.Plot(tgr)
			out = fmt.Sprintf("%s%s\n", clr, tgr)
		}

		buf.WriteString(out)

		var prefix, suffix string
		if lastValue.v != value.v {
			pingValue = lastValue.v
			pingTime = now
		}

		if pingValue != value.v && now.Sub(pingTime) < time.Second {
			prefix, suffix = clrRed, rst
			if value.v >= pingValue {
				prefix = clrGreen
			}
		}
		str0 := fmt.Sprintf(" %s/%s ", pair.Base, pair.Counter)
		str1 := fmt.Sprintf(" %.2f ", value.v)
		str2 := fmt.Sprintf(
			" %.2f  %.2f ",
			vwap.Range(now.Add(-time.Hour), now).Value(),
			vwap.Range(now.Add(-24*time.Hour), now).Value(),
		)
		pad := make([]byte, (termX-(len(str0)+len(str1)+len(str2)))/2)
		for i := range pad {
			pad[i] = ' '
		}

		fmt.Fprint(buf, clrLine, string(pad), bg, " ", str0, prefix, str1, suffix, bg, str2, " ", rst)
		cursor := cursorHome
		if nograph {
			cursor = cursorBOL
		}
		fmt.Fprint(buf, cursor)

		lastValue = value

		io.Copy(os.Stdout, buf)
	}
}

func main() {
	configDir, _ := os.UserConfigDir()
	if configDir != "" {
		configDir = filepath.Join(configDir, "bitstamp")
	}

	truncate := time.Hour * 24
	alarmsf := make(flagAlarms, 0)
	var alarmCmd string
	var baseCurrency, counterCurrency string
	var nograph bool
	flag.StringVar(&configDir, "c", configDir, "config directory")
	flag.DurationVar(&truncate, "h", truncate, "[live] truncate graph after this duration into the past")
	flag.BoolVar(&nograph, "g", false, "[live] hide graph")
	flag.Var(&alarmsf, "a", "[live] set alarms (e.g. '>10000', '<8000')")
	flag.StringVar(&alarmCmd, "e", "", "[live] command to execute when an alarm is triggered, %p will be replaced with the current market price and %a with the alarm condition")
	flag.StringVar(&baseCurrency, "bc", bitstamp.BTC.String(), "base currency")
	flag.StringVar(&counterCurrency, "cc", bitstamp.EUR.String(), "counter currency")

	flag.Usage = func() {
		out := os.Stdout
		fmt.Fprintf(out, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(out, "Commands:")
		fmt.Fprintln(out, "  live | <empty>:   show market data")
		fmt.Fprintln(out, "  current | c:      show current price")
		fmt.Fprintln(out, "  balance | b:      get account balance")
		fmt.Fprintln(out, "  transactions | t: list account transactions")
		fmt.Fprintln(out, "  list-currencies:  list known currency pairs")
	}
	flag.Parse()

	pair := generic.CurrencyPair{generic.Currency(baseCurrency), generic.Currency(counterCurrency)}

	cmd := flag.Arg(0)
	var a action
	var authed bool
	switch cmd {
	case "", "live":
		a = actionLive
	case "b", "balance":
		a = actionBalance
		authed = true
	case "t", "transactions":
		a = actionTransactions
		authed = true
	case "list-currencies":
		a = actionCurrencies
	case "c", "current":
		a = actionCurrent
	}

	var apiKey, apiSecret string
	if authed {
		if configDir == "" {
			exit(errors.New("please set a config directory"))
		}

		authFile := filepath.Join(configDir, "auth")
		f, err := os.Open(authFile)
		exit(err)
		authBin, err := io.ReadAll(f)
		exit(err)

		lines := strings.Split(strings.TrimSpace(string(authBin)), "\n")
		if len(lines) < 2 {
			exit(errors.New("auth file invalid"))
		}
		apiKey, apiSecret = lines[0], lines[1]
	}

	client, err := bitstamp.NewDefaults(apiKey, apiSecret)
	exit(err)

	switch a {
	case actionBalance:
		r, err := client.API.Balance()
		exit(err)
		for _, v := range bitstamp.AllCurrencies() {
			l := r.ForCurrency(v)
			if len(l) == 0 || l["balance"] == 0 {
				continue
			}
			p := bitstamp.Precision(v)
			f := fmt.Sprintf("%%s: %%10.%df / %%10.%df\n", p, p)
			fmt.Printf(f, v, l["available"], l["balance"])
		}
	case actionTransactions:
		list, err := client.Transactions()
		exit(err)
		type item struct {
			currency generic.Currency
			value    float64
		}

		for _, n := range list {
			items := make([]item, 0, 10)
			for k, v := range n.Values {
				if v == 0 {
					continue
				}
				items = append(items, item{k, v})
			}

			sort.Slice(items, func(i, j int) bool {
				return items[i].currency < items[j].currency
			})

			strs := make([]string, len(items), len(items)+1)
			for i, it := range items {
				p := bitstamp.Precision(it.currency)
				f := fmt.Sprintf("%%s: %%.%df", p)
				strs[i] = fmt.Sprintf(f, it.currency, it.value)
			}
			strs = append(strs, fmt.Sprintf("FEE: %.2f", n.Fee))

			fmt.Printf(
				"%s %10s %s\n",
				n.DateTime.Value().Local().Format(dateFormat),
				n.Type.String(),
				strings.Join(strs, " | "),
			)
		}
	case actionCurrencies:
		for _, p := range bitstamp.AllCurrencies() {
			fmt.Println(p)
		}
	case actionLive:
		alarms, err := alarmsf.Parse()
		exit(err)
		exit(live(pair, alarmCmd, alarms, nograph, truncate))
	case actionCurrent:
		r, err := client.API.Ticker(pair, api.TickerHourly)
		exit(err)
		f := fmt.Sprintf("%%.%df", bitstamp.Precision(pair.Counter))
		fmt.Printf(
			"%s\nLast:\t"+f+"\nLow:\t"+f+"\nHigh:\t"+f+"\nVWAP:\t"+f+"\n",
			r.Time.Value().Local(),
			r.Last,
			r.Low,
			r.High,
			r.VWAP,
		)
	}
}
