# Indicator output reference

The indicator calculator accepts bars in ascending time order. It buffers the
newest update for each market/timeframe and calculates/publishes that bar only
after a later bar arrives. On startup it excludes the newest database candle,
which can still be forming. Direct callers of `pkg/indicators` must pass closed
candles in ascending order.

| Key | Default | Warm-up | Output units and definition |
|---|---:|---:|---|
| `atr_pct` | period 14 | period + 1 bars | `value`: Wilder ATR divided by current close, ×100 (percent). The first ATR is the simple mean of the first `period` true ranges; later values use Wilder smoothing. True range includes gaps from the previous close. A zero close yields no value. |
| `donchian` | period 20 | period bars | `upper`, `lower`, `mid`: max high, min low, and their midpoint over the window including the current closed bar. Price units. |
| `roc` | period 6 and 12 | period + 1 bars | `value`: `(close / close[period] - 1) × 100`, percent. A zero reference close yields no value. |
| `mfi` | period 14 | period + 1 bars | `value`: positive money flow divided by total positive and negative money flow, ×100; range 0–100. If total flow is zero, output is 0. |
| `candle_features` | none | 1 bar | `body_pct`, `upper_wick_pct`, `lower_wick_pct`: each as a percentage of candle range; `range_pct`: range/current close ×100; `close_location`: `(close-low)/(high-low)`, range 0–1. A zero range yields `close_location=0.5`; other ratios are 0. |
| `bollinger` | period 20, deviation 2 | period bars | Existing `upper_band`, `middle_band`, `lower_band` plus `percent_b` in 0–1 under ordinary bands and `bandwidth` as `(upper-lower)/abs(middle) ×100`. Population standard deviation; zero width/mean yields 0 for the corresponding ratio. |

The crypto defaults can be adjusted with `WORKER_INDICATOR_*` values in
`.env`; seeding is controlled by `WORKER_INDICATOR_CRYPTO_P0_ENABLED`. Definitions
are stored per NOBITEX market and 15m/1h timeframe. Values are additive fields
in the existing `market.candle.indicators` snapshot payload.
