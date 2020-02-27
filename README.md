Namebase Exchange API client in Go
==

Go client for interacting with Namebase Exchange API.

## Usage

Some requests require an API key. You can generate a key from https://www.namebase.io/pro.

See the original API documentation: https://github.com/namebasehq/exchange-api-documentation/

## Installation

### Requirements

- go 1.10 or greater

### Install

> go get github.com/sniperem/namebase

### Usage

query order book:
```go
pair := namebase.NewCurrencyPair("hns", "btc")
if d, err := nb.GetDepth(pair, 0); err != nil {
    log.Fatal(err)
} else {
    log.Printf("ask 1: %+v, bid 1: %+v", d.Asks[0], d.Bids[0])
}
```

place order
```go
if o, err := nb.LimitBuy(decimal.NewFromFloat(100), decimal.NewFromFloat(0.00009),pair); err != nil {
    log.Print("failed to buy: ", err)
}
```

query account info
```go
if acct, err := nb.GetAccount(); err != nil {
    log.Print("failed to get account info: ", err)
} else {
    log.Printf("%+v", acct)
}
```

withdraw assets (**change deposit address before testing, or it would deposit to my wallet**)
```go
tokenAmount := decimal.NewFromFloat(2000)
if err := nb.Withdraw("HNS", tokenAmount,
    "hs1qc7kmegpjkn4qrhuactul9feu69nvsqnjpkk6sy", ""); err != nil {
    t.Error(err)
	}
```

Subscribe order book updates:
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

Subscribe trade info of a pair:
```go
if chTrade, err := nb.SubPair(pair); err != nil {
    log.Print("failed to subscribe trade info")
} else {
    go func() {
        for t := range chTrade {
            log.Printf("latest trade: %+v",t)
        }
    }()
}
```
