package main

import (
	"log"

	"github.com/shopspring/decimal"

	"github.com/sniperem/namebase"
)

func main() {
	nb, err := namebase.NewClient("", "")
	if err != nil {
		log.Fatal(err)
	}

	pair := namebase.NewCurrencyPair("hns", "btc")
	if chDepth, err := nb.SubDepth(pair); err != nil {
		log.Print("failed to subscribe order book")
	} else {
		go func() {
			for d := range chDepth {
				log.Printf("ask 1: %+v, bid 1: %+v", d.Asks[0], d.Bids[0])
			}
		}()
	}

	if d, err := nb.GetDepth(pair, 0); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("ask 1: %+v, bid 1: %+v", d.Asks[0], d.Bids[0])
	}

	if o, err := nb.LimitBuy(decimal.NewFromFloat(100), decimal.NewFromFloat(0.00009), pair); err != nil {
		log.Print("failed to buy: ", err)
	} else {
		log.Printf("%+v", o)
	}

}
