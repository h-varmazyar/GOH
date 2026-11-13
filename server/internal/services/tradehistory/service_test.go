package tradehistory

import (
	"testing"

	"github.com/h-varmazyar/gatec/internal/adapters/stock/tse"
)

func TestDayResultTotalsLeaveCanceledTradesOut(t *testing.T) {
	var result DayResult
	result.addTotals([]tse.TradeHistory{
		{Price: 1000, Quantity: 10},
		{Price: 1010, Quantity: 5, Canceled: 1},
		{Price: 990, Quantity: 20},
	})
	want := DayResult{TradeCount: 3, CanceledCount: 1, Volume: 35, NetVolume: 30, NetValue: 1000*10 + 990*20}
	if result != want {
		t.Fatalf("totals = %+v, want %+v", result, want)
	}
}
