# Signal engine

The signal worker evaluates strategy definitions from `strategies/*.yaml`, the
`strategy_definitions` table, or both. Definitions share one strict schema;
unknown YAML/JSON keys are rejected. In `both` mode a database definition
replaces a file definition with the same ID. Invalid reloads are logged and do
not replace the last valid strategy.

## Strategy conditions

An operand is a finite number, a reference string, or a scaled reference such
as `{ref: ema_20, mul: 1.02, add: 0}`. Scaling is applied as `value * mul +
add`. References use this grammar:

```text
[alias.][timeframe.]field[.output][[-N]]
```

Examples: `close` and `rsi` refer to the current trigger candle;
`ema_20[-1]` refers to the previous trigger candle; `macd.histogram` selects
the MACD histogram; `1h.ema_20` reads the latest closed 1h value; and
`btc.1h.close` reads the configured `btc` reference symbol. Candle fields are
`open`, `high`, `low`, `close`, and `volume`. Multi-output indicator values use
their indicator name and output separated by a dot.

Supported condition operators:

| Operator | Arguments | Meaning |
|---|---|---|
| `gt`, `gte`, `lt`, `lte`, `eq` | `[left, right]` | Numeric comparison |
| `between`, `outside` | `[value, low, high]` | Inclusive range / its inverse |
| `crosses_above`, `crosses_below` | `[value, threshold]` | Previous value at or beyond the threshold, current value strictly across it |
| `rising`, `falling` | `[value, bars]` | Strictly monotonic change over the requested number of bars |
| `within_last` | `{bars, cond}` | Nested condition held at least once in the most recent bars |
| `all`, `any`, `not` | list / condition | Boolean composition |

Missing values and incomplete warm-up are false and mark the evaluation as not
ready. A not-ready evaluation cannot create an entry. Successful leaf checks
record their operands in the decision reason for inspection.

## Signal messages

The indicator worker publishes enriched snapshots on
`NATS_CANDLE_SNAPSHOT_SUBJECT` (default `market.candle.indicators`). The signal
worker consumes them through JetStream with explicit acknowledgments. Signals
are JSON on `signals.<market>.<timeframe>.<symbol>.<type>`; shadow-mode signals
use `signals.shadow.<market>.<timeframe>.<symbol>.<type>`. Subject tokens are
lowercase and punctuation is replaced with `-`. Each message has
`schema_version`, deterministic `signal_id`, strategy identity and mode,
market/symbol/timeframe, candle timestamps, price and risk levels, reason
evidence, and (for exits) the entry ID, exit reason, PnL, and bars held.

Signals are saved before publishing. Failed publishes remain pending and are
retried on startup and periodically. ENTRY rows track OPEN/CLOSED position
state; EXIT rows link to their entry. No order is sent to an exchange or broker.

## Running

```sh
go run ./cmd/worker signal-worker
go run ./cmd/signal-replay --strategy strategies/examples/example_ema_cross.yaml \
  --source file --market crypto --symbols BTCUSDT,ETHUSDT \
  --from 2026-01-01 --to 2026-01-07
```

Replay uses the same router and virtual-position manager as live processing,
does not publish signals or write signal rows, and prints JSON statistics.
