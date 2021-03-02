package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/frizinak/bitstamp"
	"github.com/frizinak/bitstamp/api"
	"github.com/frizinak/bitstamp/generic"
	"github.com/google/shlex"
	"github.com/vdobler/chart"
	"github.com/vdobler/chart/txtg"
)

var dateFormat = "2006-01-02 15:04:05"

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

func main() {
	configDir, _ := os.UserConfigDir()
	if configDir != "" {
		configDir = filepath.Join(configDir, "bitstamp")
	}

	var hours uint = 24
	alarmsf := make(flagAlarms, 0)
	var alarmCmd string
	var baseCurrency, counterCurrency string
	flag.StringVar(&configDir, "c", configDir, "config directory")
	flag.UintVar(&hours, "h", hours, "[live] truncate graph after this amount of hours into the past")
	flag.Var(&alarmsf, "a", "[live] set alarms (e.g. '>10000', '<8000')")
	flag.StringVar(&alarmCmd, "e", "", "[live] command to execute when an alarm is triggered, %p will be replaced with the current market price and %a with the alarm condition")
	flag.StringVar(&baseCurrency, "bc", bitstamp.BTC.String(), "base currency")
	flag.StringVar(&counterCurrency, "cc", bitstamp.EUR.String(), "counter currency")

	flag.Usage = func() {
		out := os.Stdout
		fmt.Fprintf(out, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(out, "Commands:")
		fmt.Fprintln(out, "  balance | b:      get account balance")
		fmt.Fprintln(out, "  transactions | t: list account transactions")
		fmt.Fprintln(out, "  live | <empty>:   show market data")
		fmt.Fprintln(out, "  list-currencies:  list known currency pairs")
	}
	flag.Parse()

	notify := func(price float64, alarm Alarm) {}
	if alarmCmd == "" && len(alarmsf) != 0 {
		exit(errors.New("no alarm command set"))
	}

	alarms, err := alarmsf.Parse()
	exit(err)

	if alarmCmd != "" {
		cmd, err := shlex.Split(alarmCmd)
		exit(err)
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
	apiKey, apiSecret := lines[0], lines[1]
	client, err := bitstamp.NewDefaults(apiKey, apiSecret)
	exit(err)

	cmd := flag.Arg(0)
	if cmd != "" && cmd != "live" {
		switch cmd {
		case "b", "balance":
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
		case "t", "transactions":
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
		case "list-currencies":
			for _, p := range bitstamp.AllCurrencies() {
				fmt.Println(p)
			}
		}

		return
	}

	type Value struct {
		t time.Time
		v float64
	}
	trades := make(chan bitstamp.Trade, 1)

	go func() {
		err := client.TradesLive(
			api.TradesHistoryDay,
			generic.CurrencyPair{generic.Currency(baseCurrency), generic.Currency(counterCurrency)},
			trades,
		)
		exit(err)
	}()

	var lastUpdate time.Time

	tradePoints := make([]chart.EPoint, 0)

	vwap := make(VWAPS, 0)
	vwapPointsFull := make([]chart.EPoint, 0)
	vwapPoints := make([]chart.EPoint, 0)
	var lastVWAP time.Time
	vwapInterval := time.Hour * 6

	var value, lastValue Value
	start := time.Now()

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

	var pingTime time.Time
	var pingValue float64

	for {
	sel:
		for {
			select {
			case trade := <-trades:
				if trade.Date.After(start) {
					results := alarms.Check(value.v, trade.Price)
					for _, a := range results {
						notifications <- notification{trade.Price, a}
					}
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

			case <-time.After(time.Millisecond * 200):
				break sel
			}
		}

		now := time.Now()
		if now.Sub(lastUpdate) < time.Millisecond*25 {
			continue
		}

		lastUpdate = time.Now()

		const clr = "\033[2J"
		const bg = "\033[40;1m"
		const rst = "\033[0m"

		termX, termY := termSize()
		if termX > 20 && termY > 8 && len(tradePoints) > 0 {
			tgr := txtg.New(termX, termY-2)
			p := chart.ScatterChart{
				Key:    chart.Key{Hide: true, Cols: 3, Pos: "otc", Border: -1},
				YRange: chart.Range{},
				XRange: chart.Range{
					Time: true,
					MinMode: chart.RangeMode{
						Fixed:  true,
						TValue: time.Now().Add(-time.Duration(hours) * time.Hour),
					},
				},
			}

			const symbol4 = '█'
			const symbol3 = '▓'
			const symbol2 = '▒'
			const symbol1 = '░'
			p.AddData("VWAP", vwapPointsFull, chart.PlotStyleLines, chart.Style{Symbol: symbol1})
			p.AddData("Trades", tradePoints, chart.PlotStylePoints, chart.Style{Symbol: symbol2})
			p.AddData("Now", tradePoints[len(tradePoints)-1:], chart.PlotStylePoints, chart.Style{Symbol: symbol4})

			p.Plot(tgr)
			fmt.Print(clr, tgr, "\n")
		}

		var prefix, suffix string
		if lastValue.v != value.v {
			pingValue = lastValue.v
			pingTime = now
		}

		if pingValue != value.v && now.Sub(pingTime) < time.Second {
			prefix, suffix = "\033[30;41m", rst
			if value.v >= pingValue {
				prefix = "\033[30;42m"
			}
		}
		str0 := fmt.Sprintf(" %s/%s ", baseCurrency, counterCurrency)
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

		fmt.Print("\033[K", string(pad), bg, " ", str0, prefix, str1, suffix, bg, str2, " ", rst, "\033[H")

		lastValue = value
	}
}
