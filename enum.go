package bitstamp

import "github.com/frizinak/bitstamp/generic"

const (
	BTC  generic.Currency = "btc"
	USD                   = "usd"
	GBP                   = "gbp"
	EUR                   = "eur"
	PAX                   = "pac"
	USDC                  = "usdc"
	XRP                   = "xrp"
	LTC                   = "ltc"
	ETH                   = "eth"
	BCH                   = "bch"
	XLM                   = "xlm"
	LINK                  = "link"
	OMG                   = "omg"
)

func Precision(c generic.Currency) int {
	switch c {
	case USD, GBP, EUR:
		return 2
	}
	return 8
}

func AllCurrencies() []generic.Currency {
	return []generic.Currency{
		BTC,
		USD,
		GBP,
		EUR,
		PAX,
		USDC,
		XRP,
		LTC,
		ETH,
		BCH,
		XLM,
		LINK,
		OMG,
	}
}

func BTCUSD() generic.CurrencyPair { return generic.CurrencyPair{BTC, USD} }
func BTCEUR() generic.CurrencyPair { return generic.CurrencyPair{BTC, EUR} }
func BTCGBP() generic.CurrencyPair { return generic.CurrencyPair{BTC, GBP} }
func BTCPAX() generic.CurrencyPair { return generic.CurrencyPair{BTC, PAX} }
func GBPUSD() generic.CurrencyPair { return generic.CurrencyPair{GBP, USD} }
func GBPEUR() generic.CurrencyPair { return generic.CurrencyPair{GBP, EUR} }
func EURUSD() generic.CurrencyPair { return generic.CurrencyPair{EUR, USD} }
func XRPUSD() generic.CurrencyPair { return generic.CurrencyPair{XRP, USD} }
func XRPEUR() generic.CurrencyPair { return generic.CurrencyPair{XRP, EUR} }
func XRPBTC() generic.CurrencyPair { return generic.CurrencyPair{XRP, BTC} }
func XRPGBP() generic.CurrencyPair { return generic.CurrencyPair{XRP, GBP} }
func XRPPAX() generic.CurrencyPair { return generic.CurrencyPair{XRP, PAX} }
func LTCUSD() generic.CurrencyPair { return generic.CurrencyPair{LTC, USD} }
func LTCEUR() generic.CurrencyPair { return generic.CurrencyPair{LTC, EUR} }
func LTCBTC() generic.CurrencyPair { return generic.CurrencyPair{LTC, BTC} }
func LTCGBP() generic.CurrencyPair { return generic.CurrencyPair{LTC, GBP} }
func ETHUSD() generic.CurrencyPair { return generic.CurrencyPair{ETH, USD} }
func ETHEUR() generic.CurrencyPair { return generic.CurrencyPair{ETH, EUR} }
func ETHBTC() generic.CurrencyPair { return generic.CurrencyPair{ETH, BTC} }
func ETHGBP() generic.CurrencyPair { return generic.CurrencyPair{ETH, GBP} }
func ETHPAX() generic.CurrencyPair { return generic.CurrencyPair{ETH, PAX} }
func BCHUSD() generic.CurrencyPair { return generic.CurrencyPair{BCH, USD} }
func BCHEUR() generic.CurrencyPair { return generic.CurrencyPair{BCH, EUR} }
func BCHBTC() generic.CurrencyPair { return generic.CurrencyPair{BCH, BTC} }
func BCHGBP() generic.CurrencyPair { return generic.CurrencyPair{BCH, GBP} }
func PAXUSD() generic.CurrencyPair { return generic.CurrencyPair{PAX, USD} }
func PAXEUR() generic.CurrencyPair { return generic.CurrencyPair{PAX, EUR} }
func PAXGBP() generic.CurrencyPair { return generic.CurrencyPair{PAX, GBP} }
func XLMBTC() generic.CurrencyPair { return generic.CurrencyPair{XLM, BTC} }
func XLMUSD() generic.CurrencyPair { return generic.CurrencyPair{XLM, USD} }
func XLMEUR() generic.CurrencyPair { return generic.CurrencyPair{XLM, EUR} }
func XLMGBP() generic.CurrencyPair { return generic.CurrencyPair{XLM, GBP} }
func OMGUSD() generic.CurrencyPair { return generic.CurrencyPair{OMG, USD} }
func OMGEUR() generic.CurrencyPair { return generic.CurrencyPair{OMG, EUR} }
func OMGGBP() generic.CurrencyPair { return generic.CurrencyPair{OMG, GBP} }
func OMGBTC() generic.CurrencyPair { return generic.CurrencyPair{OMG, BTC} }

func LINKUSD() generic.CurrencyPair { return generic.CurrencyPair{LINK, USD} }
func LINKEUR() generic.CurrencyPair { return generic.CurrencyPair{LINK, EUR} }
func LINKGBP() generic.CurrencyPair { return generic.CurrencyPair{LINK, GBP} }
func LINKBTC() generic.CurrencyPair { return generic.CurrencyPair{LINK, BTC} }
func LINKETH() generic.CurrencyPair { return generic.CurrencyPair{LINK, ETH} }
func USDCUSD() generic.CurrencyPair { return generic.CurrencyPair{USDC, USD} }
func USDCEUR() generic.CurrencyPair { return generic.CurrencyPair{USDC, EUR} }
func ETHUSDC() generic.CurrencyPair { return generic.CurrencyPair{ETH, USDC} }
func BTCUSDC() generic.CurrencyPair { return generic.CurrencyPair{BTC, USDC} }
