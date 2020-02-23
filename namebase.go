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

type Namebase struct {
	apiKey     string
	secretKey  string
	httpClient *http.Client
	// wsConn         *websocket.Conn
	// createWsLock   sync.Mutex
	// pubsub         *util.PubSub
	// topicsRegistry map[string]struct {
	// 	cmd   interface{}
	// 	count int
	// }
	// topicsMu   sync.Mutex
	symbolInfo map[CurrencyPair]symbolInfo
}

func NewClient(key, secret string) (*Namebase, error) {
	client := &Namebase{
		apiKey:    key,
		secretKey: secret,

		httpClient: &http.Client{Timeout: time.Second * 10},
	}

	// symbolInfo, err := client.AllSymbols()
	// if err != nil {
	// 	return nil, err
	// }

	// client.symbolInfo = symbolInfo

	return client, nil
}

// GetTicker implements the API interface
// func (nb *Namebase) GetTicker(pair CurrencyPair) (*Ticker, error) {

// 	return nil, nil
// }

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
		m[NewCurrencyPair("", "")] = s
	}

	return m, nil
}

// GetDepth implements the API interface
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

	// reverse
	// for i := len(d.Asks) - 1; i > 0; i-- {
	// 	depth.AskList = append(depth.AskList, d.Asks[i])
	// }

	return d, nil
}

func (nb *Namebase) placeOrder(amount, price decimal.Decimal, pair CurrencyPair,
	orderType string, side OrderSide) (*Order, error) {
	info, ok := nb.symbolInfo[pair]
	if !ok {
		return nil, errors.New("unsupported symbol")
	}

	amount = amount.Truncate(info.BasePrecision)

	if amount.IsZero() {
		return nil, errors.New("qty is zero")
	}

	price = price.Truncate(info.QuotePrecision)

	params := make(map[string]interface{})
	params["symbol"] = pair.String()
	params["side"] = strings.ToUpper(string(side))
	params["type"] = orderType
	params["quantity"] = amount.String()
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
func (bn *Namebase) GetAccount() (*Account, error) {
	params := make(map[string]interface{})
	// coinType optional
	data, err := bn.do(http.MethodGet, "/api/v0/account", params, true)
	if err != nil {
		return nil, err
	}

	var acct Account

	if err := json.Unmarshal(data, &acct); err != nil {
		return nil, err
	}

	return &acct, nil
}

// LimitBuy implements the API interface
func (bn *Namebase) LimitBuy(amount, price decimal.Decimal, pair CurrencyPair) (*Order, error) {
	return bn.placeOrder(amount, price, pair, "LMT", BuyOrder)
}

// LimitSell implements the API interface
func (bn *Namebase) LimitSell(amount, price decimal.Decimal, pair CurrencyPair) (*Order, error) {
	return bn.placeOrder(amount, price, pair, "LMT", SellOrder)
}

// MarketBuy implements the API interface
func (bn *Namebase) MarketBuy(amount decimal.Decimal, pair CurrencyPair) (*Order, error) {
	return bn.placeOrder(amount, decimal.Zero, pair, "MKT", BuyOrder)
}

// MarketSell implements the API interface
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

// GetOrder implements the API interface
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

// GetKlines implements the API interface
// func (nb *Namebase) GetKlines(pair CurrencyPair, interval KlineInterval, size, start int) ([]*Kline, error) {
// 	panic("")
// }

// GetTrades implements the API interface
// func (nb *Namebase) GetTrades(pair CurrencyPair, size int) ([]*Trade, error) {
// 	panic("")
// }

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

// SubDepth implements the API interface
func (nb *Namebase) SubDepth(pair CurrencyPair) (chan Depth, error) {
	wsConn, _, err := websocket.DefaultDialer.Dial(baseWsURL+"/ws/v0/ticker/depth", nil)
	if err != nil {
		log.Printf("[namebase] failed to establish a websocket connection", err)
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

				// TODO reconnect
				// if err := nb.reconnectWs(); err != nil {
				// 	log.Printf("[namebase] ERROR\twebsocket reconnect error: %s", err)
				// 	return
				// }
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

// SubTicker implements the API interface
// func (c *Namebase) SubTicker(pair CurrencyPair, handler func(*Ticker)) error {
// 	return nil
// }

// SubTrades implements the API interface
// func (nb *Namebase) SubTrades(pair CurrencyPair, handler func([]*Trade)) error {

// 	return nil
// }

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
