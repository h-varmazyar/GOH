import type { AlertChannelType, AssetClass, DeepPartial, Strategy, StrategyAlerts } from '../types';

export const providerNames: Record<string, string> = {
  binance: 'Binance',
  nobitex: 'Nobitex',
  tsetmc: 'TSETMC',
  rahavard: 'ره‌آورد ۳۶۵',
  demo_broker: 'کارگزاری دمو',
};

export const defaultAlerts: StrategyAlerts = {
  enabled: true,
  events: ['signal', 'entry', 'exit', 'stop_loss', 'take_profit', 'error'],
  channels: {
    telegram: { enabled: true, destination: '@gate_signals' },
    sms: { enabled: false, destination: '' },
    push: { enabled: true, destination: 'مرورگر فعلی' },
    email: { enabled: false, destination: '' },
  },
  minimumConfidence: 60,
  quietHours: { enabled: false, start: '23:00', end: '07:00' },
};

export function createDefaultStrategy(): Strategy {
  return {
    name: '', assetClass: 'crypto', provider: 'binance', providerName: 'Binance', description: '', enabled: true, tags: [],
    alerts: clone(defaultAlerts), indicators: {},
    market: { symbols: [], excluded_symbols: [], timeframes: [], sessions: ['24x7'] },
    filters: {
      market: { min_daily_volume_usdt: 1000000, max_spread_percent: 0.2, min_price: 0.0001, max_price: 100000, blacklist_low_liquidity: true },
      volatility: { atr: { min_percentile: 20, max_percentile: 95 } },
      cooldown: { after_entry_seconds: 300, after_stoploss_seconds: 600 },
      correlation: { enabled: true, max_same_direction_positions: 3 },
    },
    regime: { trending: { conditions: { all: [] } }, ranging: { conditions: { all: [] } }, volatile: { conditions: { all: [] } } },
    entry: {
      long: { all: [], score: { minimum: 0.65, weights: { trend: 0.4, momentum: 0.3, volatility: 0.1, volume: 0.2 } } },
      short: { all: [], score: { minimum: 0.65, weights: { trend: 0.4, momentum: 0.3, volatility: 0.1, volume: 0.2 } } },
    },
    exit: { long: { any: [] }, short: { any: [] } },
    risk: {
      position: { max_open_positions: 10, max_position_per_symbol: 1, risk_per_trade_percent: 1, max_daily_loss_percent: 5, max_drawdown_percent: 15 },
      stoploss: { type: 'atr', atr: { multiplier: 1.5, minimum_percent: 0.4, maximum_percent: 3 } },
      trailing_stop: { enabled: true, activation_rr: 1, atr_multiplier: 1 },
      takeprofit: { type: 'risk_reward', risk_reward: { minimum: 1.5, target: 2, partial_close_rr: 1, partial_close_percent: 50 } },
    },

    execution: { order_type: 'market', slippage: { max_percent: 0.15 }, retry: { enabled: true, attempts: 3, delay_ms: 200 }, timeout_ms: 3000 },
    scoring: { confidence: { minimum: 0.6 }, penalties: {}, bonuses: {} },
    multi_timeframe: { enabled: false, confirmation: { higher_timeframe: '1h', conditions: { all: [] } } },
    telemetry: { enabled: true, log_signals: true, log_rejections: true, log_scores: true, metrics: ['winrate', 'sharpe', 'pnl', 'drawdown'] },
  };
}

export function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

export function deepMerge<T>(base: T, patch?: DeepPartial<T>): T {
  if (!patch) return clone(base);
  const output = clone(base) as Record<string, unknown>;
  Object.entries(patch as Record<string, unknown>).forEach(([key, value]) => {
    if (value === undefined) return;
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      output[key] = deepMerge((output[key] ?? {}) as object, value as never);
    } else {
      output[key] = clone(value);
    }
  });
  return output as T;
}

export function normalizeStrategy(input: DeepPartial<Strategy>): Strategy {
  const assetClass: AssetClass = input.assetClass === 'stock' ? 'stock' : 'crypto';
  const provider = input.provider || (assetClass === 'stock' ? 'tsetmc' : 'binance');
  const merged = deepMerge(createDefaultStrategy(), input);
  const channelTypes: AlertChannelType[] = ['telegram', 'sms', 'push', 'email'];
  channelTypes.forEach((channel) => {
    merged.alerts.channels[channel] = {
      ...defaultAlerts.channels[channel],
      ...(input.alerts?.channels?.[channel] || {}),
    };
  });
  return { ...merged, assetClass, provider, providerName: input.providerName || providerNames[provider] || provider };
}
