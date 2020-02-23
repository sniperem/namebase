Namebase Exchange Api client in Go
==

Go client for interacting with Namebase Exchange API.

## Usage

Some requests require an API key. You can generate a key from https://www.namebase.io/pro.

See the raw API documentation calls: https://github.com/namebasehq/exchange-api-documentation/

## Installation

### Requirements

- go 1.10 or greater

### Install

> go get github.com/sniperem/namebase

### Usage

REST API for Namebase Exchange
```go
pair := namebase.NewCurrencyPair("hns", "btc")
if d, err := nb.GetDepth(pair, 0); err != nil {
    log.Fatal(err)
} else {
    log.Printf("ask 1: %+v, bid 1: %+v", d.Asks[0], d.Bids[0])
}

if o, err := nb.LimitBuy(decimal.NewFromFloat(100), decimal.NewFromFloat(0.00009),pair); err != nil {
    log.Print("failed to buy: ", err)
}
```

WebSocket API for Namebase
```go
if chDepth, err := nb.SubDepth(pair); err != nil {
    log.Print("failed to subscribe order book")
} else {
    go func() {
        for d := range chDepth {
            log.Printf("ask 1: %+v, bid 1: %+v", d.Asks[0], d.Bids[0])
        }
    }()
}
```
