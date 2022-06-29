package main

import (
	"github.com/adshao/go-binance/v2/futures"
	"github.com/shopspring/decimal"
)

type OrderSlice struct {
	buysell   futures.SideType // buy , sell
	direction futures.PositionSideType
	//openclose	string   	// open , close
	quantity decimal.Decimal
	price    decimal.Decimal
	start    int64
	//order_ret 	interface{}
	symbol       *Symbol
	timeout      int64
	orderRequest *futures.CreateOrderResponse
}

func CreateOrderSlice(target, long_pos, short_pos decimal.Decimal) (slices []*OrderSlice) {
	present := long_pos.Sub(short_pos)
	if target.Equal(present) {
		return
	}
	diff := target.Sub(present)
	var (
		buy_close, buy_open, sell_close, sell_open decimal.Decimal
	)

	buy_close = decimal.NewFromFloat(0)
	buy_open = decimal.NewFromFloat(0)
	sell_close = decimal.NewFromFloat(0)
	sell_open = decimal.NewFromFloat(0)

	if diff.GreaterThan(decimal.NewFromFloat(0)) { // buy
		buy_close = decimal.Min(short_pos, diff)
		buy_open = diff.Sub(buy_close)
	} else if diff.LessThan(decimal.NewFromFloat(0)) { // sell
		sell_close = decimal.Min(long_pos, diff.Abs())
		sell_open = diff.Abs().Sub(sell_close)
	}

	if buy_open.GreaterThan(decimal.NewFromFloat(0)) {
		slice := OrderSlice{direction: "LONG", buysell: "BUY", quantity: buy_open}
		slices = append(slices, &slice)
	}
	if sell_close.GreaterThan(decimal.NewFromFloat(0)) {
		slice := OrderSlice{direction: "LONG", buysell: "SELL", quantity: sell_close}
		slices = append(slices, &slice)
	}
	/////////////
	if sell_open.GreaterThan(decimal.NewFromFloat(0)) {
		slice := OrderSlice{direction: "SHORT", buysell: "SELL", quantity: sell_open}
		slices = append(slices, &slice)
	}

	if buy_close.GreaterThan(decimal.NewFromFloat(0)) {
		slice := OrderSlice{direction: "SHORT", buysell: "BUY", quantity: buy_close}
		slices = append(slices, &slice)
	}

	return
}
