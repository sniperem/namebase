package namebase

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// KlineInterval is the interval of k-Line
type KlineInterval string

const (
	// KlineInterval1Min
	KlineInterval1Min  KlineInterval = "1m"
	KlineInterval5Min  KlineInterval = "5m"
	KlineInterval15Min KlineInterval = "15m"
	KlineInterval1H    KlineInterval = "1h"
	KlineInterval12H   KlineInterval = "12h"
	KlineInterval1Day  KlineInterval = "1d"
	KlineInterval1Week               = "1w"
)

// OrderSide is either BUY or SELL
type OrderSide string

const (
	BuyOrder  OrderSide = "BUY"
	SellOrder OrderSide = "SELL"
)

// Currency is the symbol of crypto currency, such as BTC, ETH, etc.
type Currency string

// NewCurrency creates a Currency from string
func NewCurrency(s string) Currency {
	return Currency(strings.ToUpper(s))
}

// CurrencyPair is a trading pair
type CurrencyPair struct {
	Base  Currency
	Quote Currency
}

// String implements the Stringer interface
func (pair CurrencyPair) String() string {
	// make sure it's upper case
	return strings.Join([]string{string(pair.Base), string(pair.Quote)}, "")
}

// NewCurrencyPair creates a trading pair from string
func NewCurrencyPair(base, quote string) CurrencyPair {
	return CurrencyPair{
		NewCurrency(base),
		NewCurrency(quote),
	}
}

// DepthRecord represents an item in the order book
type DepthRecord struct {
	Price,
	Amount decimal.Decimal
}

// UnmarshalJSON unmarshal the given depth raw data like ["0.06844", "10760"],
// [price, amount], and converts to DepthRecord
func (b *DepthRecord) UnmarshalJSON(data []byte) error {
	if b == nil {
		return fmt.Errorf("UnmarshalJSON on nil pointer")
	}

	if len(data) == 0 {
		return nil
	}
	// TODO
	s := strings.ReplaceAll(string(data), `"`, "")
	s = strings.Trim(s, "[]")
	tokens := strings.Split(s, ",")
	if len(tokens) < 2 {
		return fmt.Errorf("at least two fields are expected but got: %v", tokens)
	}
	var err error
	b.Price, err = decimal.NewFromString(tokens[0])
	b.Amount, err = decimal.NewFromString(tokens[1])

	return err
}

// DepthRecords is multiple DepthRecord
type DepthRecords []DepthRecord

func (dr DepthRecords) Len() int {
	return len(dr)
}

func (dr DepthRecords) Swap(i, j int) {
	dr[i], dr[j] = dr[j], dr[i]
}

func (dr DepthRecords) Less(i, j int) bool {
	return dr[i].Price.LessThan(dr[j].Price)
}

// Depth represents order book
type Depth struct {
	Bids        []DepthRecord
	Asks        []DepthRecord
	Ts          int64
	LastEventID int64
}

// Kline is candlestick
type Kline struct {
	OpenTime       int64  `json:"openTime"`
	CloseTime      int64  `json:"closeTime"`
	OpenPrice      string `json:"openPrice"`
	HighPrice      string `json:"highPrice"`
	LowPrice       string `json:"lowPrice"`
	ClosePrice     string `json:"closePrice"`
	Volume         string `json:"volume"`
	QuoteVolume    string `json:"quoteVolume"`
	NumberOfTrades int    `json:"numberOfTrades"`
}

type Trade struct {
	TradeID       int    `json:"tradeId"`
	Price         string `json:"price"`
	Quantity      string `json:"quantity"`
	QuoteQuantity string `json:"quoteQuantity"`
	CreatedAt     int64  `json:"createdAt"`
	IsBuyerMaker  bool   `json:"isBuyerMaker"`
}

// Account represents account info
type Account struct {
	MakerFee int  `json:"makerFee"`
	TakerFee int  `json:"takerFee"`
	CanTrade bool `json:"canTrade"`
	Balances []struct {
		Asset          string `json:"asset"`
		Unlocked       string `json:"unlocked"`
		LockedInOrders string `json:"lockedInOrders"`
		CanDeposit     bool   `json:"canDeposit"`
		CanWithdraw    bool   `json:"canWithdraw"`
	} `json:"balances"`
}

// Order is order
type Order struct {
	OrderID int `json:"orderId"`
	Price,
	OriginalQuantity,
	ExecutedQuantity decimal.Decimal
	Status    string `json:"status"`
	Type      string `json:"type"`
	Side      string `json:"side"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type symbolInfo struct {
	Symbol         string   `json:"symbol"`
	Status         string   `json:"status"`
	BaseAsset      string   `json:"baseAsset"`
	BasePrecision  int32    `json:"basePrecision"`
	QuoteAsset     string   `json:"quoteAsset"`
	QuotePrecision int32    `json:"quotePrecision"`
	OrderTypes     []string `json:"orderTypes"`
}

type exchInfo struct {
	Timezone   string
	ServerTime int64
	Symbols    []symbolInfo
}
