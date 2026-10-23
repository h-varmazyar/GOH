export type MarketMood = 'صعودی' | 'متعادل' | 'نزولی';

export interface IndexSnapshot {
  name: string;
  value: number;
  change: number;
}

export interface MarketAnalysis {
  id: string;
  date: string;
  weekday: string;
  mood: MarketMood;
  totalIndex: number;
  totalIndexChange: number;
  equalIndexChange: number;
  tradeValue: number;
  netMoney: number;
  advancing: number;
  declining: number;
  summary: string;
  indices: IndexSnapshot[];
  buyCandidates: PickCandidate[];
  sellCandidates: PickCandidate[];
}

export interface PickCandidate {
  symbol: string;
  name: string;
  score: number;
  reason: string;
}

export interface Indicator {
  id: string;
  name: string;
  code: string;
  category: 'روند' | 'مومنتوم' | 'حجم' | 'نوسان';
  timeframe: string;
  enabled: boolean;
  parameters: Record<string, number>;
}

export interface MarketItem {
  id: string;
  symbol: string;
  name: string;
  industry: string;
  board: string;
  price: number;
  change: number;
  volume: number;
  active: boolean;
}

const buyCandidates: PickCandidate[] = [
  { symbol: 'فولاد', name: 'فولاد مبارکه اصفهان', score: 89, reason: 'شکست مقاومت با حجم معتبر' },
  { symbol: 'وبملت', name: 'بانک ملت', score: 85, reason: 'ورود پول حقیقی و بهبود روند' },
  { symbol: 'شپنا', name: 'پالایش نفت اصفهان', score: 81, reason: 'واگرایی مثبت و رشد تقاضا' },
  { symbol: 'کگل', name: 'معدنی گل‌گهر', score: 78, reason: 'حمایت حقوقی و نسبت ریسک مناسب' },
  { symbol: 'خودرو', name: 'ایران خودرو', score: 74, reason: 'عبور میانگین کوتاه‌مدت' },
];

const sellCandidates: PickCandidate[] = [
  { symbol: 'فملی', name: 'ملی صنایع مس ایران', score: 83, reason: 'اشباع خرید و کاهش قدرت تقاضا' },
  { symbol: 'شستا', name: 'سرمایه‌گذاری تأمین اجتماعی', score: 79, reason: 'از دست رفتن حمایت کوتاه‌مدت' },
  { symbol: 'وتجارت', name: 'بانک تجارت', score: 76, reason: 'خروج پول و افزایش عرضه' },
  { symbol: 'پارس', name: 'پتروشیمی پارس', score: 72, reason: 'واگرایی منفی در RSI' },
  { symbol: 'ذوب', name: 'ذوب آهن اصفهان', score: 69, reason: 'ضعف حجم در محدوده مقاومت' },
];

const dates = ['۱۴۰۵/۰۵/۰۹', '۱۴۰۵/۰۵/۰۸', '۱۴۰۵/۰۵/۰۷', '۱۴۰۵/۰۵/۰۶', '۱۴۰۵/۰۵/۰۵', '۱۴۰۵/۰۵/۰۴', '۱۴۰۵/۰۵/۰۳', '۱۴۰۵/۰۴/۳۱', '۱۴۰۵/۰۴/۳۰', '۱۴۰۵/۰۴/۲۹', '۱۴۰۵/۰۴/۲۸', '۱۴۰۵/۰۴/۲۷', '۱۴۰۵/۰۴/۲۴', '۱۴۰۵/۰۴/۲۳', '۱۴۰۵/۰۴/۲۲', '۱۴۰۵/۰۴/۲۱', '۱۴۰۵/۰۴/۲۰', '۱۴۰۵/۰۴/۱۷'];
const weekdays = ['پنج‌شنبه', 'چهارشنبه', 'سه‌شنبه', 'دوشنبه', 'یکشنبه', 'شنبه'];
const summaries = [
  'بازار امروز با برتری تقاضا در گروه‌های بانکی، فلزی و پالایشی به پایان رسید. رشد هم‌زمان شاخص کل و هم‌وزن، همراه با افزایش ارزش معاملات خرد، نشانه‌ای از توزیع مناسب تقاضا بود؛ هرچند عرضه در نیمه دوم جلسه افزایش یافت.',
  'معاملات در فضایی متعادل دنبال شد و چرخش نقدینگی میان صنایع بزرگ و کوچک مشاهده شد. حفظ سطوح حمایتی شاخص و ثبات ارزش معاملات، احتمال ادامه نوسان مثبت را تقویت می‌کند.',
  'فشار فروش در نمادهای شاخص‌ساز موجب عقب‌نشینی نماگرهای اصلی شد. کاهش ارزش معاملات و خروج پول حقیقی، رعایت حد ضرر و کاهش اندازه موقعیت‌ها را ضروری می‌کند.',
];

export const analyses: MarketAnalysis[] = dates.map((date, index) => {
  const cycle = index % 3;
  const mood: MarketMood = cycle === 0 ? 'صعودی' : cycle === 1 ? 'متعادل' : 'نزولی';
  const direction = mood === 'صعودی' ? 1 : mood === 'نزولی' ? -1 : 0.25;
  const totalChange = Number((direction * (0.45 + (index % 5) * 0.2)).toFixed(2));
  const equalChange = Number((direction * (0.32 + (index % 4) * 0.16)).toFixed(2));
  const totalIndex = 2847310 - index * 11740;
  return {
    id: `analysis-${index + 1}`,
    date,
    weekday: weekdays[index % weekdays.length],
    mood,
    totalIndex,
    totalIndexChange: totalChange,
    equalIndexChange: equalChange,
    tradeValue: Number((8.4 - (index % 6) * 0.47).toFixed(1)),
    netMoney: Math.round(direction * (312 - (index % 5) * 42)),
    advancing: mood === 'صعودی' ? 428 - index : mood === 'نزولی' ? 184 + index : 306 - index,
    declining: mood === 'نزولی' ? 392 - index : mood === 'صعودی' ? 171 + index : 284 + index,
    summary: summaries[cycle],
    indices: [
      { name: 'شاخص کل', value: totalIndex, change: totalChange },
      { name: 'شاخص هم‌وزن', value: 896420 - index * 2840, change: equalChange },
      { name: 'شاخص ۳۰ شرکت بزرگ', value: 176230 - index * 640, change: Number((totalChange * 1.08).toFixed(2)) },
      { name: 'شاخص قیمت', value: 462810 - index * 1520, change: Number((totalChange * 0.76).toFixed(2)) },
      { name: 'شاخص فرابورس', value: 27142 - index * 83, change: Number((equalChange * 0.9).toFixed(2)) },
    ],
    buyCandidates,
    sellCandidates,
  };
});

const indicatorSeeds: Omit<Indicator, 'id'>[] = [
  { name: 'شاخص قدرت نسبی', code: 'RSI', category: 'مومنتوم', timeframe: 'روزانه', enabled: true, parameters: { period: 14, overbought: 70, oversold: 30 } },
  { name: 'همگرایی/واگرایی میانگین', code: 'MACD', category: 'روند', timeframe: 'روزانه', enabled: true, parameters: { fast: 12, slow: 26, signal: 9 } },
  { name: 'میانگین متحرک نمایی', code: 'EMA', category: 'روند', timeframe: 'روزانه', enabled: true, parameters: { period: 21 } },
  { name: 'باندهای بولینگر', code: 'BB', category: 'نوسان', timeframe: 'روزانه', enabled: true, parameters: { period: 20, deviation: 2 } },
  { name: 'میانگین دامنه واقعی', code: 'ATR', category: 'نوسان', timeframe: 'روزانه', enabled: false, parameters: { period: 14 } },
  { name: 'حجم متوازن', code: 'OBV', category: 'حجم', timeframe: 'روزانه', enabled: true, parameters: { smoothing: 5 } },
  { name: 'استوکاستیک', code: 'STOCH', category: 'مومنتوم', timeframe: '۶۰ دقیقه', enabled: true, parameters: { k: 14, d: 3, smooth: 3 } },
  { name: 'شاخص جریان نقدینگی', code: 'MFI', category: 'حجم', timeframe: 'روزانه', enabled: true, parameters: { period: 14 } },
  { name: 'ابر ایچیموکو', code: 'ICHIMOKU', category: 'روند', timeframe: 'روزانه', enabled: false, parameters: { conversion: 9, base: 26, span: 52 } },
  { name: 'شاخص قدرت روند', code: 'ADX', category: 'روند', timeframe: 'هفتگی', enabled: true, parameters: { period: 14, threshold: 25 } },
  { name: 'کانال کلتنر', code: 'KC', category: 'نوسان', timeframe: 'روزانه', enabled: false, parameters: { period: 20, multiplier: 2 } },
  { name: 'نرخ تغییر قیمت', code: 'ROC', category: 'مومنتوم', timeframe: 'روزانه', enabled: true, parameters: { period: 12 } },
];
export const initialIndicators: Indicator[] = indicatorSeeds.map((item, index) => ({ ...item, id: `indicator-${index + 1}` }));

const marketSeeds: Omit<MarketItem, 'id'>[] = [
  { symbol: 'فولاد', name: 'فولاد مبارکه اصفهان', industry: 'فلزات اساسی', board: 'بورس - بازار اول', price: 4830, change: 2.4, volume: 328, active: true },
  { symbol: 'فملی', name: 'ملی صنایع مس ایران', industry: 'فلزات اساسی', board: 'بورس - بازار اول', price: 7460, change: -1.1, volume: 241, active: true },
  { symbol: 'وبملت', name: 'بانک ملت', industry: 'بانک‌ها', board: 'بورس - بازار اول', price: 3294, change: 3.2, volume: 812, active: true },
  { symbol: 'وتجارت', name: 'بانک تجارت', industry: 'بانک‌ها', board: 'بورس - بازار دوم', price: 1784, change: -0.8, volume: 945, active: true },
  { symbol: 'شپنا', name: 'پالایش نفت اصفهان', industry: 'فرآورده‌های نفتی', board: 'بورس - بازار اول', price: 6230, change: 1.7, volume: 187, active: true },
  { symbol: 'شبندر', name: 'پالایش نفت بندرعباس', industry: 'فرآورده‌های نفتی', board: 'بورس - بازار اول', price: 8140, change: 0.6, volume: 146, active: true },
  { symbol: 'خودرو', name: 'ایران خودرو', industry: 'خودرو', board: 'بورس - بازار دوم', price: 2497, change: 4.1, volume: 1104, active: true },
  { symbol: 'خساپا', name: 'سایپا', industry: 'خودرو', board: 'بورس - بازار دوم', price: 1942, change: -2.2, volume: 804, active: false },
  { symbol: 'کگل', name: 'معدنی و صنعتی گل‌گهر', industry: 'استخراج کانه فلزی', board: 'بورس - بازار اول', price: 13820, change: 1.3, volume: 78, active: true },
  { symbol: 'کچاد', name: 'معدنی و صنعتی چادرملو', industry: 'استخراج کانه فلزی', board: 'بورس - بازار اول', price: 9120, change: 0.4, volume: 96, active: true },
  { symbol: 'شستا', name: 'سرمایه‌گذاری تأمین اجتماعی', industry: 'سرمایه‌گذاری', board: 'بورس - بازار دوم', price: 1268, change: -1.5, volume: 522, active: true },
  { symbol: 'وغدیر', name: 'سرمایه‌گذاری غدیر', industry: 'سرمایه‌گذاری', board: 'بورس - بازار اول', price: 15740, change: 0.9, volume: 55, active: true },
  { symbol: 'ذوب', name: 'ذوب آهن اصفهان', industry: 'فلزات اساسی', board: 'فرابورس - بازار دوم', price: 3521, change: -3.1, volume: 301, active: false },
  { symbol: 'آریا', name: 'پلیمر آریاساسول', industry: 'محصولات شیمیایی', board: 'فرابورس - بازار اول', price: 22140, change: 2.8, volume: 42, active: true },
  { symbol: 'نوری', name: 'پتروشیمی نوری', industry: 'محصولات شیمیایی', board: 'بورس - بازار اول', price: 78420, change: 1.1, volume: 21, active: true },
  { symbol: 'پارس', name: 'پتروشیمی پارس', industry: 'محصولات شیمیایی', board: 'بورس - بازار اول', price: 35620, change: -0.7, volume: 33, active: false },
  { symbol: 'دعبید', name: 'داروسازی دکتر عبیدی', industry: 'دارویی', board: 'بورس - بازار اول', price: 29110, change: 3.6, volume: 18, active: true },
  { symbol: 'برکت', name: 'گروه دارویی برکت', industry: 'دارویی', board: 'بورس - بازار دوم', price: 6840, change: -0.3, volume: 64, active: true },
];
export const initialMarkets: MarketItem[] = marketSeeds.map((item, index) => ({ ...item, id: `market-${index + 1}` }));

export const watchlist = initialMarkets.slice(0, 14).map((item, index) => ({
  symbol: item.symbol,
  name: item.name,
  price: item.price,
  change: item.change,
  signal: (index % 3 === 0 ? 'خرید' : index % 3 === 1 ? 'نگهداری' : 'فروش') as 'خرید' | 'نگهداری' | 'فروش',
  note: index % 2 === 0 ? 'نزدیک محدوده حمایتی؛ بررسی واکنش حجم' : 'انتظار برای تثبیت قیمت بالای مقاومت روزانه',
  alert: index % 4 === 0,
}));

const eventTypes = ['مجمع', 'عرضه اولیه', 'گزارش', 'سود نقدی'] as const;
const eventSymbols = ['فولاد', 'آریا', 'وبملت', 'شپنا', 'نوری', 'کگل', 'دعبید', 'شبندر', 'خودرو', 'فملی', 'شستا', 'وغدیر', 'برکت'];
export const marketCalendar = eventSymbols.map((symbol, index) => {
  const type = eventTypes[index % eventTypes.length];
  const day = (12 + index).toLocaleString('fa-IR', { minimumIntegerDigits: 2, useGrouping: false });
  return {
    id: `event-${index + 1}`,
    date: `۱۴۰۵/۰۵/${day}`,
    symbol,
    type,
    title: type === 'مجمع' ? 'مجمع عمومی عادی سالیانه' : type === 'عرضه اولیه' ? 'عرضه اولیه سهام شرکت' : type === 'گزارش' ? 'انتشار گزارش عملکرد ماهانه' : 'آخرین مهلت دریافت سود نقدی',
    description: type === 'مجمع' ? 'توقف نماد دو روز کاری پیش از برگزاری مجمع' : 'جزئیات و زمان‌بندی رویداد در اطلاعیه کدال منتشر شده است.',
  };
});
