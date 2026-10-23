package replay

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/h-varmazyar/gatec/internal/services/signal/domain"
)

type Trade struct {
	Entry         domain.Signal `json:"entry"`
	Exit          domain.Signal `json:"exit"`
	NetPnLPercent float64       `json:"net_pnl_pct"`
}

type Statistics struct {
	Signals              int                   `json:"signals"`
	Trades               int                   `json:"trades"`
	WinRate              float64               `json:"win_rate_pct"`
	AverageProfit        float64               `json:"average_profit_pct"`
	AverageLoss          float64               `json:"average_loss_pct"`
	ProfitFactor         *float64              `json:"profit_factor,omitempty"`
	Expectancy           float64               `json:"expectancy_pct"`
	MaxConsecutiveLosses int                   `json:"max_consecutive_losses"`
	MaxDrawdown          float64               `json:"max_drawdown_pct"`
	AverageBarsHeld      float64               `json:"average_bars_held"`
	ExitsByReason        map[string]int        `json:"exits_by_reason"`
	BySymbol             map[string]Statistics `json:"by_symbol,omitempty"`
	ByMonth              map[string]Statistics `json:"by_month,omitempty"`
}

type Report struct {
	From        time.Time  `json:"from"`
	To          time.Time  `json:"to"`
	FeePercent  float64    `json:"fee_percent_per_side"`
	Split       float64    `json:"split"`
	Trades      []Trade    `json:"trades"`
	All         Statistics `json:"all"`
	InSample    Statistics `json:"in_sample"`
	OutOfSample Statistics `json:"out_of_sample"`
}

func BuildReport(signals []domain.Signal, from, to time.Time, feePercent, split float64) (Report, error) {
	if feePercent < 0 || math.IsNaN(feePercent) || math.IsInf(feePercent, 0) {
		return Report{}, fmt.Errorf("fee percent must be finite and non-negative")
	}
	if split <= 0 || split >= 1 {
		return Report{}, fmt.Errorf("split must be between 0 and 1")
	}
	entries := make(map[string]domain.Signal)
	trades := make([]Trade, 0)
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].CandleCloseTime.Equal(signals[j].CandleCloseTime) {
			return signals[i].SignalID < signals[j].SignalID
		}
		return signals[i].CandleCloseTime.Before(signals[j].CandleCloseTime)
	})
	for _, signal := range signals {
		switch signal.Type {
		case domain.SignalEntry:
			entries[signal.SignalID] = signal
		case domain.SignalExit:
			if signal.Exit == nil {
				continue
			}
			entry, ok := entries[signal.Exit.EntrySignalID]
			if !ok {
				continue
			}
			net := signal.Exit.PnLPercent.InexactFloat64() - feePercent*2
			trades = append(trades, Trade{Entry: entry, Exit: signal, NetPnLPercent: net})
			delete(entries, signal.Exit.EntrySignalID)
		}
	}
	sort.Slice(trades, func(i, j int) bool {
		if trades[i].Entry.CandleCloseTime.Equal(trades[j].Entry.CandleCloseTime) {
			return trades[i].Entry.SignalID < trades[j].Entry.SignalID
		}
		return trades[i].Entry.CandleCloseTime.Before(trades[j].Entry.CandleCloseTime)
	})
	boundary := from.Add(time.Duration(float64(to.Sub(from)) * split))
	report := Report{From: from, To: to, FeePercent: feePercent, Split: split, Trades: trades}
	entriesCount := 0
	for _, signal := range signals {
		if signal.Type == domain.SignalEntry {
			entriesCount++
		}
	}
	report.All = statistics(trades, entriesCount, true)
	inTrades := make([]Trade, 0)
	outTrades := make([]Trade, 0)
	for _, trade := range trades {
		if trade.Entry.CandleCloseTime.Before(boundary) {
			inTrades = append(inTrades, trade)
		} else {
			outTrades = append(outTrades, trade)
		}
	}
	report.InSample = statistics(inTrades, len(inTrades), true)
	report.OutOfSample = statistics(outTrades, len(outTrades), true)
	return report, nil
}

func statistics(trades []Trade, signals int, breakdowns bool) Statistics {
	stats := Statistics{Signals: signals, Trades: len(trades), ExitsByReason: make(map[string]int)}
	if breakdowns {
		stats.BySymbol = make(map[string]Statistics)
		stats.ByMonth = make(map[string]Statistics)
	}
	var wins, losses, grossWins, grossLoss, total, peak, equity, maxDD, bars float64
	currentLosses := 0
	symbolTrades := make(map[string][]Trade)
	monthTrades := make(map[string][]Trade)
	for _, trade := range trades {
		pnl := trade.NetPnLPercent
		total += pnl
		equity += pnl
		if equity > peak {
			peak = equity
		}
		drawdown := peak - equity
		if drawdown > maxDD {
			maxDD = drawdown
		}
		if trade.Exit.Exit != nil {
			stats.ExitsByReason[trade.Exit.Exit.Reason]++
			bars += float64(trade.Exit.Exit.BarsHeld)
		}
		if pnl > 0 {
			wins++
			grossWins += pnl
			currentLosses = 0
		} else if pnl < 0 {
			losses++
			grossLoss += -pnl
			currentLosses++
			if currentLosses > stats.MaxConsecutiveLosses {
				stats.MaxConsecutiveLosses = currentLosses
			}
		}
		if breakdowns {
			symbol := trade.Entry.Symbol
			month := trade.Entry.CandleCloseTime.UTC().Format("2006-01")
			symbolTrades[symbol] = append(symbolTrades[symbol], trade)
			monthTrades[month] = append(monthTrades[month], trade)
		}
	}
	if len(trades) > 0 {
		stats.WinRate = wins / float64(len(trades)) * 100
		stats.Expectancy = total / float64(len(trades))
		stats.AverageBarsHeld = bars / float64(len(trades))
	}
	if wins > 0 {
		stats.AverageProfit = grossWins / wins
	}
	if losses > 0 {
		stats.AverageLoss = -grossLoss / losses
		factor := grossWins / grossLoss
		stats.ProfitFactor = &factor
	}
	stats.MaxDrawdown = maxDD
	if breakdowns {
		for key, group := range symbolTrades {
			stats.BySymbol[key] = statistics(group, len(group), false)
		}
		for key, group := range monthTrades {
			stats.ByMonth[key] = statistics(group, len(group), false)
		}
	} else {
		stats.BySymbol = nil
		stats.ByMonth = nil
	}
	return stats
}
