// User & Auth Types
export type UserRole = 'admin' | 'user' | 'viewer';

export interface User {
  id: string;
  username: string;
  firstName: string;
  lastName: string;
  phone: string;
  role: UserRole;
  isVerified: boolean;
  createdAt: string;
}

export interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

// Strategy Types
export type AssetClass = 'crypto' | 'stock';
export type AlertEvent = 'signal' | 'entry' | 'exit' | 'stop_loss' | 'take_profit' | 'error';
export type AlertChannelType = 'telegram' | 'sms' | 'push' | 'email';

export interface AlertChannel {
  enabled: boolean;
  destination: string;
}

export interface StrategyAlerts {
  enabled: boolean;
  events: AlertEvent[];
  channels: Record<AlertChannelType, AlertChannel>;
  minimumConfidence: number;
  quietHours: {
    enabled: boolean;
    start: string;
    end: string;
  };
}

export interface IndicatorConfig {
  [key: string]: string | number | boolean;
}

export interface Indicator {
  id: number | null;
  name: string;
  type: string;
  timeframe: string;
  configs: IndicatorConfig;
}

export interface MarketConfig {
  symbols: string[];
  excluded_symbols: string[];
  timeframes: string[];
  sessions: string[];
}

export interface MarketFilters {
  min_daily_volume_usdt: number;
  max_spread_percent: number;
  min_price: number;
  max_price: number;
  blacklist_low_liquidity: boolean;
}

export interface VolatilityFilter {
  atr: {
    min_percentile: number;
    max_percentile: number;
  };
}

export interface CooldownFilter {
  after_entry_seconds: number;
  after_stoploss_seconds: number;
}

export interface CorrelationFilter {
  enabled: boolean;
  max_same_direction_positions: number;
}

export interface Filters {
  market: MarketFilters;
  volatility: VolatilityFilter;
  cooldown: CooldownFilter;
  correlation: CorrelationFilter;
}

export interface RegimeCondition {
  conditions: {
    all: string[];
  };
}

export interface Regime {
  trending: RegimeCondition;
  ranging: RegimeCondition;
  volatile: RegimeCondition;
}

export interface EntryScore {
  minimum: number;
  weights: {
    trend: number;
    momentum: number;
    volatility: number;
    volume: number;
  };
}

export interface EntryDirection {
  all: string[];
  score: EntryScore;
}

export interface Entry {
  long: EntryDirection;
  short: EntryDirection;
}

export interface ExitDirection {
  any: string[];
}

export interface Exit {
  long: ExitDirection;
  short: ExitDirection;
}

export interface RiskPosition {
  max_open_positions: number;
  max_position_per_symbol: number;
  risk_per_trade_percent: number;
  max_daily_loss_percent: number;
  max_drawdown_percent: number;
}

export interface StopLoss {
  type: string;
  atr: {
    multiplier: number;
    minimum_percent: number;
    maximum_percent: number;
  };
}

export interface TrailingStop {
  enabled: boolean;
  activation_rr: number;
  atr_multiplier: number;
}

export interface TakeProfit {
  type: string;
  risk_reward: {
    minimum: number;
    target: number;
    partial_close_rr: number;
    partial_close_percent: number;
  };
}

export interface Risk {
  position: RiskPosition;
  stoploss: StopLoss;
  trailing_stop: TrailingStop;
  takeprofit: TakeProfit;
}

export interface Execution {
  order_type: string;
  slippage: { max_percent: number };
  retry: { enabled: boolean; attempts: number; delay_ms: number };
  timeout_ms: number;
}

export interface Scoring {
  confidence: { minimum: number };
  penalties: Record<string, number>;
  bonuses: Record<string, number>;
}

export interface MultiTimeframe {
  enabled: boolean;
  confirmation: {
    higher_timeframe: string;
    conditions: { all: string[] };
  };
}

export interface Telemetry {
  enabled: boolean;
  log_signals: boolean;
  log_rejections: boolean;
  log_scores: boolean;
  metrics: string[];
}

export interface Strategy {
  id?: string;
  name: string;
  assetClass: AssetClass;
  provider: string;
  providerName?: string;
  templateId?: string;
  description: string;
  enabled: boolean;
  tags: string[];
  alerts: StrategyAlerts;
  indicators: Record<string, Indicator>;
  market: MarketConfig;
  filters: Filters;
  regime: Regime;
  entry: Entry;
  exit: Exit;
  risk: Risk;
  execution: Execution;
  scoring: Scoring;
  multi_timeframe: MultiTimeframe;
  telemetry: Telemetry;
  createdAt?: string;
  updatedAt?: string;
}

// Dashboard Types
export interface DashboardStats {
  totalTrades: number;
  winRate: number;
  totalPnl: number;
  activeStrategies: number;
  openPositions: number;
  dailyPnl: number;
  maxDrawdown: number;
  sharpeRatio: number;
}

// Server Data Types (fetched from API)
export interface AvailableMarket {
  symbol: string;
  name: string;
  category: string;
  assetClass: AssetClass;
  providerIds: string[];
}

export interface AvailableTimeframe {
  value: string;
  label: string;
}

export interface AvailableIndicator {
  type: string;
  name: string;
  description: string;
  configs: {
    key: string;
    label: string;
    type: 'number' | 'string' | 'select';
    default: string | number;
    options?: string[];
    min?: number;
    max?: number;
  }[];
}

export interface AvailableProvider {
  id: string;
  name: string;
  description: string;
  assetClass: AssetClass;
}

export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends Array<infer U>
    ? Array<U>
    : T[P] extends object
      ? DeepPartial<T[P]>
      : T[P];
};

export interface StrategyTemplate {
  id: string;
  name: string;
  description: string;
  assetClass: AssetClass;
  provider: string;
  providerName: string;
  category: string;
  riskLevel: 'کم' | 'متوسط' | 'زیاد';
  timeframe: string;
  indicators: string[];
  tags: string[];
  preset: DeepPartial<Strategy>;
}

// Telegram Types
export type TelegramAccountType = 'bot' | 'personal';
export type TelegramAccountStatus = 'connected' | 'needs_auth' | 'disconnected';

export interface TelegramAccount {
  id: string;
  username: string;
  type: TelegramAccountType;
  accountId: string;
  status: TelegramAccountStatus;
  phone?: string;
  createdAt: string;
}

export interface TelegramAuthStep {
  step: 'phone' | 'code' | 'password';
  phone?: string;
  code?: string;
  password?: string;
}
