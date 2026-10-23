# Strategy DSL Rules

## Grammar (complete)

- `expression := or_expr`
- `or_expr := and_expr ("||" and_expr)*`
- `and_expr := equality_expr ("&&" equality_expr)*`
- `equality_expr := comparison_expr (("==" | "!=") comparison_expr)*`
- `comparison_expr := term (("<" | "<=" | ">" | ">=") term)*`
- `term := factor (("+" | "-") factor)*`
- `factor := unary (("*" | "/") unary)*`
- `unary := ("!" | "-") unary | primary`
- `primary := NUMBER | BOOL | IDENT_PATH | FUNC_CALL | "(" expression ")"`
- `IDENT_PATH := identifier ("." identifier)*`
- `FUNC_CALL := identifier "(" [expression ("," expression)*] ")"`

## Tokens

- Number: `12`, `12.5`
- Bool: `true`, `false`
- Identifier: `rsi`, `market.close`, `features.momentum`
- Operators: `+ - * / < <= > >= == != && || !`
- Grouping: `(` `)`
- Function args separator: `,`

## Semantic conventions (recommended)

- Paths map to runtime fields, e.g. `market.close`, `indicators.rsi14`.
- Functions are registered in function registry, e.g. `cross_over(a,b)`, `sma(series, period)`.
- Comparison/logical expressions should finally evaluate to boolean for rule usage.
- Arithmetic sub-expressions may be used anywhere in comparison/function arguments.

## Valid examples

- `rsi14 < 30 && close > ema50`
- `cross_over(ema20, ema50) && volume > sma(volume, 20)`
- `(adx >= 25 && trend.score > 0.6) || breakout(high, 20)`
- `!in_cooldown && risk.exposure < 0.6`

## Invalid examples

- `rsi <` (incomplete)
- `&& close > ema50` (missing left side)
- `sma(close 20)` (missing comma)
- `price..close > 1` (invalid path)
