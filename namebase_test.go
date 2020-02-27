package namebase

import (
	"log"
	"testing"

	"github.com/jflyup/decimal"
)

var nb, _ = NewClient("", "")

func TestExchInfo(t *testing.T) {
	if symbols, err := nb.exchInfo(); err != nil {
		t.Errorf("exch info error: %v", err)
	} else {
		log.Print(symbols)
	}
}

func TestGetAccount(t *testing.T) {
	if acct, err := nb.GetAccount(); err != nil {
		t.Error(err)
	} else {
		log.Printf("%+v", acct)
	}
}

func TestWithdraw(t *testing.T) {
	tokenAmount := decimal.NewFromFloat(2000)
	if err := nb.Withdraw("HNS", tokenAmount,
		"hs1qc7kmegpjkn4qrhuactul9feu69nvsqnjpkk6sy", ""); err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	if _, err := nb.CancelOrder(1733667,
		NewCurrencyPair("hns", "btc")); err != nil {
		t.Errorf("error: %v", err)
	}
}

func TestGetOrder(t *testing.T) {
	if order, err := nb.GetOrder(1733236,
		NewCurrencyPair("hns", "btc")); err != nil {
		t.Error(err)
	} else {
		log.Printf("%+v", order)
	}

}

func TestMarketBuy(t *testing.T) {
	if o, err := nb.MarketBuy(decimal.NewFromFloat(16.0408695),
		NewCurrencyPair("HNS", "btc")); err != nil {
		t.Error(err)
	} else {
		log.Print(o)
	}
}

func TestLimitBuy(t *testing.T) {
	if o, err := nb.LimitBuy(decimal.NewFromFloat(100), decimal.NewFromFloat(0.21),
		NewCurrencyPair("hns", "btc")); err != nil {
		t.Error(err)
	} else {
		log.Print(o)
	}
}

func TestLimitSell(t *testing.T) {
	//	acct, _ := nb.GetAccount()
	// amount := acct.SubAccounts[goup.NewCurrency("hns")].Amount
	amount := decimal.NewFromFloat(200)
	if o, err := nb.LimitSell(amount,
		decimal.NewFromFloat(0.0000104),
		NewCurrencyPair("hns", "btc")); err != nil {
		t.Error(err)
	} else {
		log.Print(o)
	}
}

func TestGetDepth(t *testing.T) {
	if d, err := nb.GetDepth(NewCurrencyPair("hns", "btc"), 0); err != nil {
		t.Error(err)
	} else {
		log.Printf("ask 1: %+v, bid 1: %+v", d.Asks[0], d.Bids[0])
	}
}

func TestSubDepth(t *testing.T) {
	pair := NewCurrencyPair("hns", "btc")

	if ch, err := nb.SubDepth(pair); err != nil {
		t.Error(err)
	} else {
		for d := range ch {
			log.Printf("ask 1: %s", d.Asks[0].Price)
			log.Printf("bid 1: %s\n", d.Bids[0].Price)
		}
	}
}

func TestSubTrade(t *testing.T) {
	pair := NewCurrencyPair("hns", "btc")

	if ch, err := nb.SubTrades(pair); err != nil {
		t.Error(err)
	} else {
		for t := range ch {
			log.Printf("latest trade: %+v", t)
		}
	}
}
