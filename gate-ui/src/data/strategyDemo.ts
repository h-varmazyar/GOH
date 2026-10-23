import type { AvailableMarket, AvailableProvider, StrategyTemplate } from '../types';

export const strategyProviders: AvailableProvider[] = [
  { id: 'binance', name: 'Binance', description: 'صرافی بین‌المللی (دمو)', assetClass: 'crypto' },
  { id: 'nobitex', name: 'Nobitex', description: 'صرافی ایرانی رمزارز', assetClass: 'crypto' },
  { id: 'tsetmc', name: 'TSETMC', description: 'داده‌های رسمی بورس تهران', assetClass: 'stock' },
  { id: 'rahavard', name: 'ره‌آورد ۳۶۵', description: 'داده و تحلیل بازار سرمایه', assetClass: 'stock' },
  { id: 'demo_broker', name: 'کارگزاری دمو', description: 'اجرای آزمایشی سفارش سهام', assetClass: 'stock' },
];

export const strategyMarkets: AvailableMarket[] = [
  { symbol: 'BTCUSDT', name: 'بیت‌کوین / تتر', category: 'رمزارز', assetClass: 'crypto', providerIds: ['binance', 'nobitex'] },
  { symbol: 'ETHUSDT', name: 'اتریوم / تتر', category: 'رمزارز', assetClass: 'crypto', providerIds: ['binance', 'nobitex'] },
  { symbol: 'SOLUSDT', name: 'سولانا / تتر', category: 'رمزارز', assetClass: 'crypto', providerIds: ['binance', 'nobitex'] },
  { symbol: 'BNBUSDT', name: 'بی‌ان‌بی / تتر', category: 'رمزارز', assetClass: 'crypto', providerIds: ['binance'] },
  { symbol: 'فولاد', name: 'فولاد مبارکه اصفهان', category: 'فلزات اساسی', assetClass: 'stock', providerIds: ['tsetmc', 'rahavard', 'demo_broker'] },
  { symbol: 'فملی', name: 'ملی صنایع مس ایران', category: 'فلزات اساسی', assetClass: 'stock', providerIds: ['tsetmc', 'rahavard', 'demo_broker'] },
  { symbol: 'خودرو', name: 'ایران خودرو', category: 'خودرو', assetClass: 'stock', providerIds: ['tsetmc', 'rahavard', 'demo_broker'] },
  { symbol: 'شپنا', name: 'پالایش نفت اصفهان', category: 'پالایشی', assetClass: 'stock', providerIds: ['tsetmc', 'rahavard', 'demo_broker'] },
  { symbol: 'وبملت', name: 'بانک ملت', category: 'بانک', assetClass: 'stock', providerIds: ['tsetmc', 'rahavard', 'demo_broker'] },
];

export const strategyTemplates: StrategyTemplate[] = [

  {
    id: 'trend', name: 'دنبال‌کننده روند', description: 'ترکیب میانگین‌های متحرک و ADX برای همراهی با روندهای قدرتمند بازار.',
    assetClass: 'crypto', provider: 'binance', providerName: 'Binance', category: 'روند', riskLevel: 'متوسط', timeframe: '1h', indicators: ['EMA 50', 'EMA 200', 'ADX'], tags: ['trend-following', 'swing'],
    preset: { name: 'crypto_trend', assetClass: 'crypto', provider: 'binance', providerName: 'Binance', description: 'استراتژی دنبال‌کننده روند رمزارز', market: { symbols: ['BTCUSDT', 'ETHUSDT'], excluded_symbols: [], timeframes: ['1h'], sessions: ['24x7'] }, indicators: { fast_ema: { id: null, name: 'fast_ema', type: 'ema', timeframe: '1h', configs: { length: 50, source: 'close' } }, slow_ema: { id: null, name: 'slow_ema', type: 'ema', timeframe: '1h', configs: { length: 200, source: 'close' } }, trend_adx: { id: null, name: 'trend_adx', type: 'adx', timeframe: '1h', configs: { length: 14 } } }, entry: { long: { all: ['fast_ema.ema > slow_ema.ema', 'trend_adx.adx > 25'] } }, exit: { long: { any: ['fast_ema.ema < slow_ema.ema'] } } },
  },
  {
    id: 'rsi-reversal', name: 'بازگشت RSI', description: 'شناسایی اشباع خرید و فروش برای معاملات بازگشتی کوتاه‌مدت.',
    assetClass: 'crypto', provider: 'nobitex', providerName: 'Nobitex', category: 'بازگشتی', riskLevel: 'زیاد', timeframe: '15m', indicators: ['RSI', 'Bollinger Bands'], tags: ['mean-reversion', 'intraday'],
    preset: { name: 'rsi_reversal', assetClass: 'crypto', provider: 'nobitex', providerName: 'Nobitex', description: 'بازگشت قیمت بر پایه RSI', market: { symbols: ['BTCUSDT'], excluded_symbols: [], timeframes: ['15m'], sessions: ['24x7'] }, indicators: { entry_rsi: { id: null, name: 'entry_rsi', type: 'rsi', timeframe: '15m', configs: { length: 14, source: 'close' } }, bands: { id: null, name: 'bands', type: 'bollinger', timeframe: '15m', configs: { length: 20, mult: 2, source: 'close' } } }, entry: { long: { all: ['entry_rsi.rsi < 30', 'close < bands.lower'] } }, exit: { long: { any: ['entry_rsi.rsi > 55'] } } },
  },
  {
    id: 'breakout', name: 'شکست محدوده', description: 'ورود پس از شکست محدوده قیمت با تأیید حجم و نوسان.',
    assetClass: 'crypto', provider: 'binance', providerName: 'Binance', category: 'شکست', riskLevel: 'متوسط', timeframe: '4h', indicators: ['ATR', 'Volume SMA'], tags: ['breakout', 'swing'],
    preset: { name: 'volume_breakout', assetClass: 'crypto', provider: 'binance', providerName: 'Binance', description: 'شکست محدوده با تأیید حجم', market: { symbols: ['SOLUSDT', 'BNBUSDT'], excluded_symbols: [], timeframes: ['4h'], sessions: ['24x7'] }, indicators: { volatility: { id: null, name: 'volatility', type: 'atr', timeframe: '4h', configs: { length: 14 } }, average_volume: { id: null, name: 'average_volume', type: 'volume_sma', timeframe: '4h', configs: { length: 20 } } }, entry: { long: { all: ['volume > average_volume.sma20', 'volatility.percentile > 60'] } } },
  },
  {
    id: 'tse-momentum', name: 'مومنتوم بورس تهران', description: 'ترکیب قدرت نسبی، حجم و روند برای نمادهای نقدشونده بورس تهران.',
    assetClass: 'stock', provider: 'tsetmc', providerName: 'TSETMC', category: 'مومنتوم', riskLevel: 'کم', timeframe: '1d', indicators: ['RSI', 'EMA 20', 'Volume SMA'], tags: ['momentum', 'swing'],
    preset: { name: 'tse_momentum', assetClass: 'stock', provider: 'tsetmc', providerName: 'TSETMC', description: 'مومنتوم روزانه سهام تهران', market: { symbols: ['فولاد', 'فملی', 'شپنا'], excluded_symbols: [], timeframes: ['1d'], sessions: ['tse_market'] }, indicators: { momentum_rsi: { id: null, name: 'momentum_rsi', type: 'rsi', timeframe: '1d', configs: { length: 14, source: 'close' } }, trend_ema: { id: null, name: 'trend_ema', type: 'ema', timeframe: '1d', configs: { length: 20, source: 'close' } }, average_volume: { id: null, name: 'average_volume', type: 'volume_sma', timeframe: '1d', configs: { length: 20 } } }, entry: { long: { all: ['momentum_rsi.rsi > 55', 'close > trend_ema.ema', 'volume > average_volume.sma20'] } }, risk: { position: { max_open_positions: 5, risk_per_trade_percent: 0.75 } } },
  },
];
