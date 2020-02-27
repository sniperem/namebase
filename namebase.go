package namebase

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jflyup/decimal"
	"github.com/jflyup/goup/util"
)

const (
	baseURL   = "https://www.namebase.io"
	baseWsURL = "wss://app.namebase.io:443"
)

// Namebase is an API client of namebase exchange
type Namebase struct {
	apiKey     string
	secretKey  string
	httpClient *http.Client
	symbolInfo map[CurrencyPair]symbolInfo
}

// NewClient creates a API client
func NewClient(key, secret string) (*Namebase, error) {
	client := &Namebase{
		apiKey:    key,
		secretKey: secret,

		httpClient: &http.Client{Timeout: time.Second * 10},
	}

	symbolInfo, err := client.exchInfo()
	if err != nil {
		return nil, err
	}

	client.symbolInfo = symbolInfo

	return client, nil
}

func (nb *Namebase) exchInfo() (map[CurrencyPair]symbolInfo, error) {
	data, err := nb.do(http.MethodGet, "/api/v0/info", nil, false)
	if err != nil {
		return nil, err
	}

	info := exchInfo{}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	m := make(map[CurrencyPair]symbolInfo)
	for _, s := range info.Symbols {
		m[NewCurrencyPair(s.BaseAsset, s.QuoteAsset)] = s
	}

	return m, nil
}

// GetDepth queries the order book of pair
func (nb *Namebase) GetDepth(pair CurrencyPair, size int) (*Depth, error) {
	params := make(map[string]interface{})
	params["symbol"] = pair.String()
	if size != 0 {
		params["limit"] = size
	}

	data, err := nb.do(http.MethodGet, "/api/v0/depth", params, false)
	if err != nil {
		return nil, err
	}

	d := &Depth{}

	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	// note that asks are in ascending order here
	// but decending order is more common, so I reverse it
	for i := len(d.Asks)/2 - 1; i >= 0; i-- {
		opp := len(d.Asks) - 1 - i
		d.Asks[i], d.Asks[opp] = d.Asks[opp], d.Asks[i]
	}

	return d, nil
}

func (nb *Namebase) placeOrder(qty, price decimal.Decimal, pair CurrencyPair,
	orderType string, side OrderSide) (*Order, error) {
	info, ok := nb.symbolInfo[pair]
	if !ok {
		return nil, errors.New("unsupported symbol")
	}

	qty = qty.Truncate(info.BasePrecision)

	if qty.IsZero() {
		return nil, errors.New("qty is zero")
	}

	price = price.Truncate(info.QuotePrecision)

	params := make(map[string]interface{})
	params["symbol"] = pair.String()
	params["side"] = strings.ToUpper(string(side))
	params["type"] = orderType
	params["quantity"] = qty.String()
	if orderType == "LMT" {
		params["price"] = price.String()
	}

	data, err := nb.do(http.MethodPost, "/api/v0/order", params, true)
	if err != nil {
		return nil, err
	}

	o := Order{}

	if err = json.Unmarshal(data, &o); err != nil {
		return nil, err
	}

	return &o, nil
}

// GetAccount query account info
func (nb *Namebase) GetAccount() (*Account, error) {
	params := make(map[string]interface{})
	// coinType optional
	data, err := nb.do(http.MethodGet, "/api/v0/account", params, true)
	if err != nil {
		return nil, err
	}

	var acct Account

	if err := json.Unmarshal(data, &acct); err != nil {
		return nil, err
	}

	return &acct, nil
}

// LimitBuy buy token at limited price
func (nb *Namebase) LimitBuy(amount, price decimal.Decimal, pair CurrencyPair) (*Order, error) {
	return nb.placeOrder(amount, price, pair, "LMT", BuyOrder)
}

// LimitSell sell token at limited price
func (nb *Namebase) LimitSell(amount, price decimal.Decimal, pair CurrencyPair) (*Order, error) {
	return nb.placeOrder(amount, price, pair, "LMT", SellOrder)
}

// MarketBuy buy token at market price
func (nb *Namebase) MarketBuy(amount decimal.Decimal, pair CurrencyPair) (*Order, error) {
	return nb.placeOrder(amount, decimal.Zero, pair, "MKT", BuyOrder)
}

// MarketSell sells token at market price
func (nb *Namebase) MarketSell(amount decimal.Decimal, pair CurrencyPair) (*Order, error) {
	return nb.placeOrder(amount, decimal.Zero, pair, "MKT", SellOrder)
}

// CancelOrder implements the API interface
func (nb *Namebase) CancelOrder(orderID int, pair CurrencyPair) (bool, error) {
	params := make(map[string]interface{})
	params["symbol"] = pair.String()
	params["orderId"] = orderID

	_, err := nb.do(http.MethodDelete, "/api/v0/order", params, true)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetOrder queries order detail
func (nb *Namebase) GetOrder(orderID int, pair CurrencyPair) (*Order, error) {
	params := make(map[string]interface{})
	params["symbol"] = pair.String()
	params["orderId"] = orderID

	data, err := nb.do(http.MethodGet, "/api/v0/order", params, true)
	if err != nil {
		return nil, err
	}

	o := &Order{}

	if err := json.Unmarshal(data, o); err != nil {
		return nil, err
	}

	return o, nil
}

// OpenOrders lists all open orders of a trading pair
func (nb *Namebase) OpenOrders(pair CurrencyPair) ([]Order, error) {
	params := make(map[string]interface{})
	params["symbol"] = pair.String()

	data, err := nb.do(http.MethodGet, "/api/v0/order/open", params, true)
	if err != nil {
		return nil, err
	}

	var orders []Order

	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

// GetKlines returns kline for a symbol
func (nb *Namebase) GetKlines(pair CurrencyPair, interval KlineInterval, limit int) ([]Kline, error) {
	params := make(map[string]interface{})
	params["symbol"] = pair.String()
	params["interval"] = interval

	if limit != 0 {
		params["limit"] = limit
	}

	data, err := nb.do(http.MethodGet, "/api/v0/ticker/klines", params, true)
	if err != nil {
		return nil, err
	}

	var klines []Kline

	if err := json.Unmarshal(data, &klines); err != nil {
		return nil, err
	}

	return klines, nil
}

// GetTrades implements the API interface
// func (nb *Namebase) GetTrades(pair CurrencyPair, size int) ([]*Trade, error) {
// 	panic("")
// }

// DepositAddr generates a deposit address
// for now, no memo is needed
func (nb *Namebase) DepositAddr(symbol Currency) (string, error) {
	params := make(map[string]interface{})

	params["asset"] = string(symbol)

	data, err := nb.do(http.MethodPost, "/api/v0/deposit/address", params, true)
	if err != nil {
		return "", err
	}

	result := struct {
		Address string `json:"address"`
		Success bool   `json:"success"`
		Asset   string `json:"asset"`
	}{}

	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	return result.Address, nil
}

// Withdraw withdraw currencies from exchange,
func (nb *Namebase) Withdraw(symbol Currency, amount decimal.Decimal, address, memo string) error {
	params := make(map[string]interface{})

	params["asset"] = string(symbol)

	params["address"] = address
	params["amount"] = amount.String()

	data, err := nb.do(http.MethodPost, "/api/v0/withdraw", params, true)
	if err != nil {
		return err
	}

	log.Print("withdraw raw: ", string(data))

	return nil
}

// OrderHistory implements the API interface
func (nb *Namebase) OrderHistory(pair CurrencyPair, size int) ([]Order, error) {
	panic("")
}

// SubKlines implements the API interface
// func (nb *Namebase) SubKlines(pair CurrencyPair, interval KlineInterval) (chan Kline, error) {
// 	panic("not implemented")
// }

func updateDepth(data DepthRecords, el DepthRecord, ask bool) DepthRecords {
	index := 0
	if ask {
		index = sort.Search(len(data), func(i int) bool {
			return data[i].Price.GreaterThanOrEqual(el.Price)
		})
	} else {
		index = sort.Search(len(data), func(i int) bool {
			return data[i].Price.LessThanOrEqual(el.Price)
		})
	}

	if index < len(data) && data[index].Price.Equal(el.Price) {
		data[index] = el
		// TODO opt
		if el.Amount.IsZero() {
			// indices are in range if 0 <= low <= high <= len(a)
			data = append(data[:index], data[index+1:]...)
		}
	} else {
		data = append(data, DepthRecord{})
		copy(data[index+1:], data[index:])
		data[index] = el
		if el.Amount.IsZero() {
			// indices are in range if 0 <= low <= high <= len(a)
			data = append(data[:index], data[index+1:]...)
		}
	}

	return data
}

// SubDepth subscribes order book updates of a trading pair
func (nb *Namebase) SubDepth(pair CurrencyPair) (chan Depth, error) {
	path := baseWsURL + "/ws/v0/ticker/depth"
	wsConn, _, err := websocket.DefaultDialer.Dial(path, nil)
	if err != nil {
		log.Print("[namebase] failed to establish a websocket connection", err)
		return nil, err
	}

	chDepth := make(chan Depth, 1)

	snapshot, err := nb.GetDepth(pair, 50)
	if err != nil {
		return nil, err
	}

	//log.Print("last event id: ", snapshot.Seq)
	go func() {
		d := struct {
			Depth
			EventType    string
			EventTime    int64
			Symbol       string
			FirstEventID int64
		}{}

		for {
			_, data, err := wsConn.ReadMessage()
			if err != nil {
				log.Printf("[namebase] ERROR\tfailed to read from websocket: %v, local addr: %s",
					err, wsConn.LocalAddr())
				wsConn.Close()
				wsConn, _, err = websocket.DefaultDialer.Dial(path, nil)
				if err != nil {
					log.Print("[namebase] failed to reconnect to websocket", err)
					// TODO notify subscriber about this error
					return
				}

				snapshot, err = nb.GetDepth(pair, 50)
				if err != nil {
					return
				}

				continue
			}

			// reset
			d.Asks = d.Asks[:0]
			d.Bids = d.Bids[:0]
			if err := json.Unmarshal(data, &d); err != nil {
				log.Printf("failed to unmarshal: %s, raw data: %s", err, string(data))
				continue
			}

			if len(d.Asks) == 0 && len(d.Bids) == 0 {
				// FirstEventID = -1
				continue
			}
			//log.Printf("first: %d, last: %d", d.FirstEventID, d.LastEventID)

			// TODO check continuity
			if d.FirstEventID > snapshot.LastEventID {
				for _, ask := range d.Asks {
					snapshot.Asks = updateDepth(snapshot.Asks, ask, true)
				}

				for _, bid := range d.Bids {
					snapshot.Bids = updateDepth(snapshot.Bids, bid, false)
				}
			}

			snapshot.LastEventID = d.LastEventID

			// deep copy
			depth := Depth{
				//Pair:    pair,
				Ts:          d.EventTime,
				LastEventID: snapshot.LastEventID,
				Asks:        make([]DepthRecord, len(snapshot.Asks)),
				Bids:        make([]DepthRecord, len(snapshot.Bids)),
			}

			copy(depth.Asks, snapshot.Asks)
			copy(depth.Bids, snapshot.Bids)

			chDepth <- depth
		}
	}()

	return chDepth, nil
}

// func (c *Namebase) SubTicker(pair CurrencyPair, handler func(*Ticker)) error {
// 	return nil
// }

// SubTrades subscribes trade info of a trading pair
// this interface seems down for now
func (nb *Namebase) SubTrades(pair CurrencyPair) (chan Trade, error) {
	path := baseWsURL + "/ws/v0/stream/trades'"
	wsConn, _, err := websocket.DefaultDialer.Dial(path, nil)
	if err != nil {
		log.Print("[namebase] failed to establish a websocket connection", err)
		return nil, err
	}

	chTrade := make(chan Trade)

	go func() {
		t := struct {
			Trade
			EventType string `json:"eventType"`
			EventTime int64  `json:"eventTime"`
			Symbol    string `json:"symbol"`
		}{}

		for {
			_, data, err := wsConn.ReadMessage()
			if err != nil {
				log.Printf("[namebase] ERROR\tfailed to read from websocket: %v, local addr: %s",
					err, wsConn.LocalAddr())
				wsConn.Close()
				wsConn, _, err = websocket.DefaultDialer.Dial(path, nil)
				if err != nil {
					log.Print("[namebase] failed to reconnect to websocket", err)
					// TODO notify subscriber about this error
					return
				}

				continue
			}

			if err := json.Unmarshal(data, &t); err != nil {
				log.Printf("failed to unmarshal: %s, raw data: %s", err, string(data))
				continue
			}

			chTrade <- t.Trade
		}
	}()

	return chTrade, nil
}

// do invokes the given API command with the given data
// sign indicates whether the api call should be done with signed payload
func (nb *Namebase) do(method, endpoint string, params map[string]interface{}, sign bool) ([]byte, error) {
	var req *http.Request
	var err error
	if sign {
		params["timestamp"] = util.CurrentTimeMillis()
	}
	if method == http.MethodGet {
		var path string
		if params != nil {
			urlParams := url.Values{}
			for k, v := range params {
				urlParams.Set(k, fmt.Sprint(v))
			}

			path = fmt.Sprintf("%s%s?%s", baseURL, endpoint, urlParams.Encode())
		} else {
			path = baseURL + endpoint
		}
		req, err = http.NewRequest(method, path, nil)
	} else {
		payload, _ := json.Marshal(params)
		req, err = http.NewRequest(method, fmt.Sprintf("%s%s", baseURL, endpoint), bytes.NewReader(payload))
		req.Header.Add("Content-Type", "application/json")
	}

	if sign {
		req.SetBasicAuth(nb.apiKey, nb.secretKey)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	// dump, _ := httputil.DumpRequest(req, true)
	// log.Print("raw: ", string(dump))
	resp, err := nb.httpClient.Do(req)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, errors.New("timeout")
		}

		return nil, err
	}

	defer resp.Body.Close()

	// dump, _ = httputil.DumpResponse(resp, true)
	// log.Print("raw resp: ", string(dump))

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := struct {
		Message string
		Code    string
	}{}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("failed to unmarshal json: %v", err)
		return nil, err
	}

	if result.Code != "" {
		return nil, errors.New(result.Message)
	}

	if resp.StatusCode != 200 {
		dump, _ := httputil.DumpResponse(resp, true)
		return nil, fmt.Errorf("http code: %d, body: %s",
			resp.StatusCode, string(dump))
	}

	return body, err
}
