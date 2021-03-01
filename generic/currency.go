package generic

type (
	Currency     string
	CurrencyPair struct{ Base, Counter Currency }
)

func (c Currency) Pair(counter Currency) CurrencyPair { return CurrencyPair{c, counter} }
func (c Currency) String() string                     { return string(c) }

func (c CurrencyPair) String() string { return string(c.Base + c.Counter) }
