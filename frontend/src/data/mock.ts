import type {
  AvailableTimeframe,
  AvailableIndicator,
  DashboardStats,
  Strategy,
  User,
} from '../types';
import { strategyMarkets, strategyProviders } from './strategyDemo';
import { normalizeStrategy } from '../utils/strategyDefaults';

export const mockUser: User = {
  id: '1',
  username: 'hossein',
  firstName: 'حسین',
  lastName: 'محمدی',
  phone: '09123456789',
  role: 'admin',
  isVerified: true,
  createdAt: '2024-01-15T10:00:00Z',
};

export const mockDashboardStats: DashboardStats = {
  totalTrades: 1247,
  winRate: 62.4,
  totalPnl: 15420.5,
  activeStrategies: 3,
  openPositions: 5,
  dailyPnl: 234.8,
  maxDrawdown: 8.2,
  sharpeRatio: 1.85,
};

export const mockProviders = strategyProviders;

export const mockMarkets = strategyMarkets;

export const mockTimeframes: AvailableTimeframe[] = [
  { value: '1m', label: '1 دقیقه' },
  { value: '3m', label: '3 دقیقه' },
  { value: '5m', label: '5 دقیقه' },
  { value: '15m', label: '15 دقیقه' },
  { value: '30m', label: '30 دقیقه' },
  { value: '1h', label: '1 ساعت' },
  { value: '4h', label: '4 ساعت' },
  { value: '1d', label: '1 روز' },
  { value: '1w', label: '1 هفته' },
];

export const mockIndicators: AvailableIndicator[] = [
  {
    type: 'ema',
    name: 'EMA (Exponential Moving Average)',
    description: 'میانگین متحرک نمایی',
    configs: [
      { key: 'length', label: 'طول', type: 'number', default: 20, min: 1, max: 500 },
      { key: 'source', label: 'منبع', type: 'select', default: 'close', options: ['open', 'high', 'low', 'close', 'hl2', 'hlc3'] },
    ],
  },
  {
    type: 'sma',
    name: 'SMA (Simple Moving Average)',
    description: 'میانگین متحرک ساده',
    configs: [
      { key: 'length', label: 'طول', type: 'number', default: 20, min: 1, max: 500 },
      { key: 'source', label: 'منبع', type: 'select', default: 'close', options: ['open', 'high', 'low', 'close', 'hl2', 'hlc3'] },
    ],
  },
  {
    type: 'rsi',
    name: 'RSI (Relative Strength Index)',
    description: 'شاخص قدرت نسبی',
    configs: [
      { key: 'length', label: 'طول', type: 'number', default: 14, min: 1, max: 100 },
      { key: 'source', label: 'منبع', type: 'select', default: 'close', options: ['open', 'high', 'low', 'close'] },
    ],
  },
  {
    type: 'macd',
    name: 'MACD',
    description: 'واگرایی/همگرایی میانگین متحرک',
    configs: [
      { key: 'fast_length', label: 'طول سریع', type: 'number', default: 12, min: 1, max: 100 },
      { key: 'slow_length', label: 'طول کند', type: 'number', default: 26, min: 1, max: 200 },
      { key: 'signal_length', label: 'طول سیگنال', type: 'number', default: 9, min: 1, max: 50 },
      { key: 'source', label: 'منبع', type: 'select', default: 'close', options: ['open', 'high', 'low', 'close'] },
    ],
  },
  {
    type: 'stochastic',
    name: 'Stochastic',
    description: 'اسیلاتور استوکاستیک',
    configs: [
      { key: 'k', label: 'دوره K', type: 'number', default: 14, min: 1, max: 100 },
      { key: 'd', label: 'دوره D', type: 'number', default: 3, min: 1, max: 50 },
      { key: 'smooth', label: 'هموارسازی', type: 'number', default: 3, min: 1, max: 50 },
    ],
  },
  {
    type: 'adx',
    name: 'ADX (Average Directional Index)',
    description: 'شاخص جهت‌دار میانگین',
    configs: [
      { key: 'length', label: 'طول', type: 'number', default: 14, min: 1, max: 100 },
    ],
  },
  {
    type: 'bollinger',
    name: 'Bollinger Bands',
    description: 'باندهای بولینگر',
    configs: [
      { key: 'length', label: 'طول', type: 'number', default: 20, min: 1, max: 200 },
      { key: 'mult', label: 'ضریب', type: 'number', default: 2, min: 0.5, max: 5 },
      { key: 'source', label: 'منبع', type: 'select', default: 'close', options: ['open', 'high', 'low', 'close'] },
    ],
  },
  {
    type: 'atr',
    name: 'ATR (Average True Range)',
    description: 'میانگین دامنه واقعی',
    configs: [
      { key: 'length', label: 'طول', type: 'number', default: 14, min: 1, max: 100 },
    ],
  },
  {
    type: 'vwap',
    name: 'VWAP',
    description: 'میانگین قیمت وزنی حجم',
    configs: [],
  },
  {
    type: 'volume_sma',
    name: 'Volume SMA',
    description: 'میانگین متحرک حجم',
    configs: [
      { key: 'length', label: 'طول', type: 'number', default: 20, min: 1, max: 200 },
    ],
  },
];

export const mockStrategies: Strategy[] = [
  {
    id: '1',
    name: 'stochastic_rsi_trend',
    assetClass: 'crypto',
    provider: 'nobitex',
    providerName: 'Nobitex',
    alerts: {
      enabled: true,
      events: ['signal', 'entry', 'exit', 'stop_loss', 'take_profit', 'error'],
      channels: {
        telegram: { enabled: true, destination: '@gate_signals' },
        sms: { enabled: false, destination: '' },
        push: { enabled: true, destination: 'مرورگر فعلی' },
        email: { enabled: false, destination: '' },
      },
      minimumConfidence: 65,
      quietHours: { enabled: false, start: '23:00', end: '07:00' },
    },
    description: 'Trend-following stochastic strategy with RSI, EMA, ADX and volume confirmation.',
    enabled: true,
    tags: ['scalping', 'intraday', 'trend-following'],
    indicators: {
      fast_ema: { id: 15, name: 'fast_ema', type: 'ema', timeframe: '5m', configs: { length: 50, source: 'close' } },
      slow_ema: { id: 16, name: 'slow_ema', type: 'ema', timeframe: '5m', configs: { length: 200, source: 'close' } },
      entry_stoch: { id: null, name: 'entry_stoch', type: 'stochastic', timeframe: '5m', configs: { k: 14, d: 3, smooth: 3 } },
      entry_rsi: { id: 44, name: 'entry_rsi', type: 'rsi', timeframe: '5m', configs: { length: 14, source: 'close' } },
      trend_adx: { id: 88, name: 'trend_adx', type: 'adx', timeframe: '5m', configs: { length: 14 } },
    },
    market: {
      symbols: ['*'],
      excluded_symbols: ['BTCDOMUSDT'],
      timeframes: ['5m'],
      sessions: ['24x7'],
    },
    filters: {
      market: { min_daily_volume_usdt: 1000000, max_spread_percent: 0.2, min_price: 0.0001, max_price: 100000, blacklist_low_liquidity: true },
      volatility: { atr: { min_percentile: 20, max_percentile: 95 } },
      cooldown: { after_entry_seconds: 300, after_stoploss_seconds: 600 },
      correlation: { enabled: true, max_same_direction_positions: 3 },
    },
    regime: {
      trending: { conditions: { all: ['adx.adx > 25', 'ema.ema50 > ema.ema200'] } },
      ranging: { conditions: { all: ['adx.adx < 18'] } },
      volatile: { conditions: { all: ['atr.percentile > 85'] } },
    },
    entry: {
      long: {
        all: ['ema.ema50 > ema.ema200', 'adx.adx > 25', 'stochastic.k < 20', 'crosses_over(stochastic.k, stochastic.d)', 'rsi.rsi < 35'],
        score: { minimum: 0.65, weights: { trend: 0.4, momentum: 0.3, volatility: 0.1, volume: 0.2 } },
      },
      short: {
        all: ['ema.ema50 < ema.ema200', 'adx.adx > 25', 'stochastic.k > 80', 'crosses_under(stochastic.k, stochastic.d)', 'rsi.rsi > 65'],
        score: { minimum: 0.65, weights: { trend: 0.4, momentum: 0.3, volatility: 0.1, volume: 0.2 } },
      },
    },
    exit: {
      long: { any: ['crosses_under(stochastic.k, stochastic.d)', 'rsi.rsi > 75', 'macd.histogram < 0'] },
      short: { any: ['crosses_over(stochastic.k, stochastic.d)', 'rsi.rsi < 25', 'macd.histogram > 0'] },
    },
    risk: {
      position: { max_open_positions: 10, max_position_per_symbol: 1, risk_per_trade_percent: 1, max_daily_loss_percent: 5, max_drawdown_percent: 15 },
      stoploss: { type: 'atr', atr: { multiplier: 1.5, minimum_percent: 0.4, maximum_percent: 3 } },
      trailing_stop: { enabled: true, activation_rr: 1, atr_multiplier: 1 },
      takeprofit: { type: 'risk_reward', risk_reward: { minimum: 1.5, target: 2, partial_close_rr: 1, partial_close_percent: 50 } },
    },
    execution: {
      order_type: 'market',
      slippage: { max_percent: 0.15 },
      retry: { enabled: true, attempts: 3, delay_ms: 200 },
      timeout_ms: 3000,
    },
    scoring: {
      confidence: { minimum: 0.6 },
      penalties: { low_volume: -0.2, high_spread: -0.3, weak_adx: -0.25, conflicting_signal: -0.4 },
      bonuses: { strong_trend: 0.2, strong_volume: 0.15, multi_timeframe_alignment: 0.3 },
    },
    multi_timeframe: {
      enabled: true,
      confirmation: { higher_timeframe: '1h', conditions: { all: ['ema.ema50 > ema.ema200', 'adx.adx > 20'] } },
    },
    telemetry: {
      enabled: true,
      log_signals: true,
      log_rejections: true,
      log_scores: true,
      metrics: ['winrate', 'sharpe', 'pnl', 'drawdown', 'exposure'],
    },
    createdAt: '2024-06-01T10:00:00Z',
    updatedAt: '2024-06-15T14:30:00Z',
  },
  normalizeStrategy({
    id: '2', name: 'مومنتوم سهام بزرگ', assetClass: 'stock', provider: 'tsetmc', providerName: 'TSETMC',
    description: 'پایش مومنتوم روزانه نمادهای بزرگ و نقدشونده بورس تهران.', enabled: true, tags: ['momentum', 'swing'],
    market: { symbols: ['فولاد', 'فملی', 'شپنا'], excluded_symbols: [], timeframes: ['1d'], sessions: ['tse_market'] },
    indicators: {
      trend_ema: { id: null, name: 'trend_ema', type: 'ema', timeframe: '1d', configs: { length: 20, source: 'close' } },
      momentum_rsi: { id: null, name: 'momentum_rsi', type: 'rsi', timeframe: '1d', configs: { length: 14, source: 'close' } },
    },
    alerts: {
      enabled: true, events: ['signal', 'entry', 'stop_loss'], minimumConfidence: 70,
      channels: { telegram: { enabled: true, destination: '@tse_signals' }, email: { enabled: true, destination: 'demo@gate.local' } },
      quietHours: { enabled: true, start: '18:00', end: '08:30' },
    },
    createdAt: '2025-01-10T08:00:00Z',
  }),
];

export const mockSessions = ['24x7', 'tse_market', 'us_market', 'eu_market', 'asia_market'];

export const mockTags = ['scalping', 'intraday', 'swing', 'trend-following', 'mean-reversion', 'breakout', 'momentum'];

export const mockOrderTypes = ['market', 'limit', 'stop_market', 'stop_limit'];

export const mockStoplossTypes = ['atr', 'percent', 'fixed', 'swing'];

export const mockTakeprofitTypes = ['risk_reward', 'percent', 'fixed', 'trailing'];

export const mockMetrics = ['winrate', 'sharpe', 'pnl', 'drawdown', 'exposure', 'profit_factor', 'avg_trade_duration'];
