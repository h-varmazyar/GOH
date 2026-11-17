package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/clickhouse"
)

// MarketCandleRepository stores the candles built from trades in ClickHouse,
// next to the trades they are built from. Candles are aggregated by ClickHouse
// with INSERT ... SELECT, so the trades never leave the database.
//
// The table is a ReplacingMergeTree keyed by market, timeframe and time: a
// candle built again, such as the weekly candle of a week that was still open,
// replaces the stored one, so reads use FINAL.
type MarketCandleRepository struct {
	client *clickhouse.Client
}

func NewMarketCandleRepository(client *clickhouse.Client) (*MarketCandleRepository, error) {
	if client == nil {
		return nil, fmt.Errorf("ClickHouse client is required")
	}
	return &MarketCandleRepository{client: client}, nil
}

const marketCandleTable = "market_candles"

// TehranTimezone is the timezone of TSE trade times and candle boundaries.
const TehranTimezone = "Asia/Tehran"

func (r *MarketCandleRepository) EnsureSchema(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS ` + marketCandleTable + ` (
		provider_id UInt64,
		market_id UInt64,
		timeframe_id UInt64,
		time DateTime('UTC'),
		open Float64,
		high Float64,
		low Float64,
		close Float64,
		volume Float64,
		value Float64,
		trade_count UInt64,
		closed Bool,
		version UInt64
	) ENGINE = ReplacingMergeTree(version)
	PARTITION BY toYear(time)
	ORDER BY (market_id, timeframe_id, time)`
	if err := r.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("create ClickHouse market candle table: %w", err)
	}
	return nil
}

// CandleTimeframe is a timeframe candles are built in.
type CandleTimeframe struct {
	ID       uint
	Duration time.Duration
}

// MonthlyDuration is the nominal duration of monthly timeframes; their
// candles are Jalali months.
const MonthlyDuration = 30 * 24 * time.Hour

// Weekly reports whether the candles of the timeframe are TSE weeks, which
// start on Saturday in Tehran. Monthly candles are Jalali months starting at
// midnight in Tehran. Other timeframes divide a day and start at multiples of
// their duration from midnight in Tehran.
func (t CandleTimeframe) Weekly() bool { return t.Duration == 7*24*time.Hour }

// Monthly reports whether the candles of the timeframe are Jalali months.
func (t CandleTimeframe) Monthly() bool { return t.Duration == MonthlyDuration }

// TSETradeRange returns the first and last trading days with trades of the
// insCodes in [from, to]; nil bounds are open. found is false without trades.
func (r *MarketCandleRepository) TSETradeRange(ctx context.Context, insCodes []string, from, to *time.Time) (first, last time.Time, found bool, err error) {
	if len(insCodes) == 0 {
		return time.Time{}, time.Time{}, false, nil
	}
	query := `SELECT toString(min(trade_date)) AS first, toString(max(trade_date)) AS last, count() > 0 AS found
		FROM ` + models.TSETradeHistory{}.TableName() + ` WHERE ` + tradeFilter(insCodes, from, to) + ` FORMAT JSONEachRow`
	var row struct {
		First string `json:"first"`
		Last  string `json:"last"`
		Found uint8  `json:"found"`
	}
	err = r.client.Query(ctx, query, func(data []byte) error { return json.Unmarshal(data, &row) })
	if err != nil || row.Found == 0 {
		return time.Time{}, time.Time{}, false, err
	}
	if first, err = time.Parse(time.DateOnly, row.First); err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("parse first trade date %q: %w", row.First, err)
	}
	if last, err = time.Parse(time.DateOnly, row.Last); err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("parse last trade date %q: %w", row.Last, err)
	}
	return first, last, true, nil
}

// TSECandleBuild builds the candles of one market from the TSE trades of its
// insCodes on the days [From, To].
type TSECandleBuild struct {
	ProviderID uint
	MarketID   uint
	InsCodes   []string
	// From and To must cover whole candles of every timeframe: a candle built
	// from part of its trades replaces the full one. Days starting on Saturday
	// and ending on Friday cover every timeframe.
	From       time.Time
	To         time.Time
	Timeframes []CandleTimeframe
	// MonthStarts are the first days of the Jalali months from the month of
	// From to the month after To; required with a monthly timeframe, whose
	// builds must cover whole months.
	MonthStarts []time.Time
	// Version orders builds; the candle of the latest build is kept.
	Version uint64
}

// BuildTSECandles aggregates the trades into candles of every timeframe and
// returns the number of candles written. Canceled trades are left out, and
// only the latest fetch of each day is used. The trades of a day are ordered
// by time; TSETMC numbers them newest first, so the higher index of two trades
// in the same second is the earlier one.
func (r *MarketCandleRepository) BuildTSECandles(ctx context.Context, build TSECandleBuild) (int64, error) {
	if len(build.InsCodes) == 0 || len(build.Timeframes) == 0 {
		return 0, nil
	}
	// Each timeframe is (id, seconds, kind): kind 0 divides a day, 1 is weekly
	// and 2 is monthly.
	timeframes := make([]string, len(build.Timeframes))
	monthly := false
	for i, timeframe := range build.Timeframes {
		seconds := int64(timeframe.Duration / time.Second)
		kind := 0
		switch {
		case timeframe.Monthly():
			kind, monthly = 2, true
		case timeframe.Weekly():
			kind = 1
		case seconds <= 0 || 86400%seconds != 0:
			return 0, fmt.Errorf("timeframe %d of %s does not divide a day", timeframe.ID, timeframe.Duration)
		}
		timeframes[i] = fmt.Sprintf("(%d, %d, %d)", timeframe.ID, seconds, kind)
	}
	monthStarts, err := monthStartsLiteral(build, monthly)
	if err != nil {
		return 0, err
	}
	from, to := build.From, build.To
	filter := tradeFilter(build.InsCodes, &from, &to)
	tz := clickhouse.Quote(TehranTimezone)
	table := models.TSETradeHistory{}.TableName()
	query := fmt.Sprintf(`INSERT INTO %[1]s (provider_id, market_id, timeframe_id, time, open, high, low, close,
		volume, value, trade_count, closed, version)
	SELECT %[2]d, %[3]d, tf.1 AS timeframe, bucket,
		argMin(price, sort_key), max(price), min(price), argMax(price, sort_key),
		sum(quantity), sum(price * quantity), count(),
		if(tf.3 = 2, makeDateTime(toYear(any(month_end)), toMonth(any(month_end)), toDayOfMonth(any(month_end)), 0, 0, 0, %[8]s),
			bucket + tf.2) <= now(), %[4]d
	FROM (
		SELECT toFloat64(price) AS price, toFloat64(quantity) AS quantity, trade_date,
			intDiv(trade_time, 10000) * 3600 + intDiv(trade_time, 100) %% 100 * 60 + trade_time %% 100 AS seconds,
			(trade_date, trade_time, -toInt64(trade_index)) AS sort_key,
			subtractDays(trade_date, (toDayOfWeek(trade_date) + 1) %% 7) AS week,
			arrayFirst(x -> x <= trade_date, arrayReverse(%[9]s)) AS month,
			arrayFirst(x -> x > trade_date, %[9]s) AS month_end
		FROM %[5]s
		WHERE %[6]s AND NOT canceled
			AND (ins_code, trade_date, version) IN (
				SELECT ins_code, trade_date, max(version) FROM %[5]s WHERE %[6]s GROUP BY ins_code, trade_date)
	)
	ARRAY JOIN [%[7]s] AS tf
	GROUP BY tf, multiIf(tf.3 = 2,
			makeDateTime(toYear(month), toMonth(month), toDayOfMonth(month), 0, 0, 0, %[8]s),
			tf.3 = 1,
			makeDateTime(toYear(week), toMonth(week), toDayOfMonth(week), 0, 0, 0, %[8]s),
			makeDateTime(toYear(trade_date), toMonth(trade_date), toDayOfMonth(trade_date),
				intDiv(intDiv(seconds, tf.2) * tf.2, 3600), intDiv(intDiv(seconds, tf.2) * tf.2, 60) %% 60,
				intDiv(seconds, tf.2) * tf.2 %% 60, %[8]s)) AS bucket`,
		marketCandleTable, build.ProviderID, build.MarketID, build.Version, table, filter,
		strings.Join(timeframes, ", "), tz, monthStarts)
	summary, err := r.client.ExecSummary(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("build candles of market %d: %w", build.MarketID, err)
	}
	return summary.WrittenRows, nil
}

// monthStartsLiteral is the ClickHouse array of the month starts of a build.
// Without a monthly timeframe it holds dates around the build, which keep
// the month columns of the query defined.
func monthStartsLiteral(build TSECandleBuild, monthly bool) (string, error) {
	starts := build.MonthStarts
	if !monthly {
		starts = []time.Time{time.Unix(0, 0).UTC(), build.To.AddDate(0, 0, 1)}
	}
	if len(starts) < 2 || starts[0].After(build.From) || !starts[len(starts)-1].After(build.To) {
		return "", fmt.Errorf("month starts must cover the build from %s to %s", build.From.Format(time.DateOnly), build.To.Format(time.DateOnly))
	}
	dates := make([]string, len(starts))
	for i, start := range starts {
		if i > 0 && !start.After(starts[i-1]) {
			return "", fmt.Errorf("month starts must be ascending")
		}
		dates[i] = "toDate(" + clickhouse.Quote(start.Format(time.DateOnly)) + ")"
	}
	return "[" + strings.Join(dates, ", ") + "]", nil
}

// Range returns the candles of a market in a timeframe opening in [from, to],
// oldest first. With limit > 0 only the newest limit candles are returned.
func (r *MarketCandleRepository) Range(ctx context.Context, marketID, timeframeID uint, from, to *time.Time, limit int) ([]models.MarketCandle, error) {
	conditions := []string{fmt.Sprintf("market_id = %d AND timeframe_id = %d", marketID, timeframeID)}
	if from != nil {
		conditions = append(conditions, fmt.Sprintf("time >= toDateTime(%d)", from.Unix()))
	}
	if to != nil {
		conditions = append(conditions, fmt.Sprintf("time <= toDateTime(%d)", to.Unix()))
	}
	query := `SELECT provider_id, market_id, timeframe_id, toInt64(toUnixTimestamp(time)) AS time, open, high, low, close,
		volume, value, trade_count, closed FROM ` + marketCandleTable + ` FINAL WHERE ` + strings.Join(conditions, " AND ")
	if limit > 0 {
		query = fmt.Sprintf("SELECT * FROM (%s ORDER BY time DESC LIMIT %d) ORDER BY time ASC", query, limit)
	} else {
		query += " ORDER BY time ASC"
	}
	// 64-bit integers are written as JSON numbers.
	query += " FORMAT JSONEachRow SETTINGS output_format_json_quote_64bit_integers = 0"
	candles := make([]models.MarketCandle, 0)
	err := r.client.Query(ctx, query, func(data []byte) error {
		var candle models.MarketCandle
		if err := json.Unmarshal(data, &candle); err != nil {
			return fmt.Errorf("decode candle: %w", err)
		}
		candles = append(candles, candle)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read candles of market %d: %w", marketID, err)
	}
	return candles, nil
}

// tradeFilter selects the trades of the insCodes on the days [from, to].
func tradeFilter(insCodes []string, from, to *time.Time) string {
	quoted := make([]string, len(insCodes))
	for i, code := range insCodes {
		quoted[i] = clickhouse.Quote(code)
	}
	conditions := []string{"ins_code IN (" + strings.Join(quoted, ", ") + ")"}
	if from != nil {
		conditions = append(conditions, "trade_date >= "+clickhouse.Quote(from.Format(time.DateOnly)))
	}
	if to != nil {
		conditions = append(conditions, "trade_date <= "+clickhouse.Quote(to.Format(time.DateOnly)))
	}
	return strings.Join(conditions, " AND ")
}
