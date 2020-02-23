package main

import (
	"log"

	"github.com/sniperem/namebase"
)

func main() {
	nb, err := namebase.NewClient("", "")
	if err != nil {
		log.Fatal(err)
	}

	pair := namebase.NewCurrencyPair("hns", "btc")
	if d, err := nb.GetDepth(pair, 0); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("ask 1: %+v, bid 1: %+v", d.Asks[0], d.Bids[0])
	}

}
