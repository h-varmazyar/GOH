# Feature: ورکر سیگنال (Strategy Engine) + تحلیل روزانه‌ی بورس

> وضعیت کلی: پیاده‌سازی تا گام ۱.۱۰ پیش رفته؛ Gate فاز ۱ به‌دلیل نبود بازار/timeframe کریپتو در دیتابیس این محیط تایید نشده است. گام ۲.۱ تکمیل شده و پیاده‌سازی گام ۲.۲ انجام شده، اما DoD آن به‌دلیل نبود کاتالوگ کریپتو و تایید نشدن اجرای زنده‌ی worker باز است. به‌درخواست کاربر، فاز ۳ به‌دلیل نبود داده‌ی کریپتو رد شده و کار فاز ۴ پیش از گام‌های باز فاز ۱/۲ ادامه می‌یابد. گام ۴.۱ تکمیل شد؛ گام بعدی ۴.۲ است.
> این فایل راهنمای گام‌به‌گام پیاده‌سازی برای ایجنت کدنویسی است. پیشرفت با تیک‌زدن چک‌باکس‌ها ثبت می‌شود.


---

## 1. قوانین کار (برای ایجنت)

1. این فایل تنها منبع حقیقت برای پیشرفت است. در شروع هر جلسه کل فایل را بخوان و از اولین گامِ تیک‌نخورده ادامه بده.
2. هر بار فقط یک گام. پس از پایان گام: همه‌ی تسک‌هایش را مرور کن ← «معیار پذیرش (DoD)» را واقعاً اجرا و تایید کن ← چک‌باکس‌های تسک‌ها و چک‌باکس همان گام در «فهرست پیشرفت» را `[x]` کن ← در «یادداشت اجرا»ی گام، خلاصه‌ی کار را بنویس (فایل‌های اصلی، تصمیم‌ها، انحراف از طرح) ← سپس متوقف شو.
3. فقط چیزی را `[x]` کن که واقعاً اجرا و تایید شده است. کار ناقص یا تاییدنشده `[ ]` می‌ماند و دلیلش در یادداشت اجرا نوشته می‌شود. تسک‌های «اختیاری» برای تیک‌خوردن گام لازم نیستند.
4. کدبیس موجود مرجع است. از convention‌های پروژه (ساختار پوشه‌ها، نام‌گذاری، error handling، logging، config، migration دیتابیس، الگوی ورکرها و NATS) پیروی کن. کدهای این سند pseudo-code و مستقل از زبان‌اند؛ آن‌ها را به زبان و سبک پروژه ترجمه کن. اگر طرح این سند با کدبیس نمی‌خواند، منطقی‌ترین راه را برگزین و انحراف را ثبت کن. برای تصمیم بزرگ معماری که اینجا پوشش داده نشده، پیش از ادامه از کاربر بپرس.
5. نوشتن test file لازم نیست و فایل تست جدید commit نکن؛ ولی صحت اجرا باید ثابت شود: build/compile بدون خطا، lint/static analysis پروژه بدون خطای جدید، و اجرای واقعی با مشاهده‌ی خروجی (`nats pub/sub`، لاگ، کوئری DB یا حالت replay). اسکریپت‌های موقتِ بررسی commit نمی‌شوند.
6. اصول طراحی (غیرقابل مذاکره):
   - Interface-first: هسته‌ی موتور هیچ چیز درباره‌ی محتوای یک استراتژی یا بازار خاص نمی‌داند (بدون `if strategy_id == ...`).
   - بدون look-ahead bias: فقط داده‌ی کندل‌های بسته‌شده تا همان لحظه.
   - خطای یک استراتژی، نماد یا پیام نباید ورکر را متوقف کند (ایزوله + لاگ).
   - هیچ آستانه، وزن یا پارامتر تجاری در کد hardcode نشود؛ همه از config.
   - همه‌ی I/O با context/timeout و graceful shutdown.
   - هیچ secret یا کلید API در کد یا کانفیگ commit نشود.
7. کتابخانه‌ی جدید فقط وقتی لازم است که پروژه معادلی ندارد؛ دلیلش را در یادداشت اجرا بنویس.

ابزارهای تایید (در صورت وجود):

```text
nats sub "signals.>"                          # دیدن سیگنال‌ها
nats pub <candle-subject> '<json نمونه>'      # تزریق «کندل + اندیکاتور» نمونه
```

## 2. وضعیت فعلی (ورودی‌های موجود)

- ورکرهای جمع‌آوری (realtime و historical): داده را از API دریافت، با data modelهای برنامه normalize، بر اساس timestamp بازار در دیتابیس ذخیره و یک رکورد را روی NATS broadcast می‌کنند.
- ورکر اندیکاتور: کندل را از NATS می‌گیرد، اندیکاتورهای تعریف‌شده برای آن بازار را حساب می‌کند و «کندل + مقادیر اندیکاتور» را دوباره روی NATS می‌فرستد.
- اندیکاتورهای موجود: `adx, atr, average volume, bollinger bands, ema distance, ema slope, ema, macd, rsi, rvi, sma, stochastics, volume spike, vwap`.
- داده‌ی ارز دیجیتال: OHLCV.
- داده‌ی بورس: OHLCV + به‌صورت روزانه و برای هر نماد، تعداد، حجم و ارزش خرید و فروش حقیقی و حقوقی.
- ورکر سیگنال هنوز وجود ندارد؛ موضوع این سند است.

## 3. اهداف و خارج از محدوده

اهداف:

1. ورکر سیگنال (نام پیشنهادی `signal-worker`) با استراتژی‌های پویا از فایل (YAML) یا دیتابیس؛ تطبیق با بازار/نماد/timeframe، تحلیل کندل‌های دریافتی با اندیکاتورهای همان استراتژی، و ارسال سیگنال ورود/خروج روی NATS. پیاده‌سازی interface-based و مستقل از محتوای استراتژی.
2. استراتژی تستی ارز دیجیتال: سیگنال روی ارزهایی که احتمال رسیدن به سود ۱ تا ۲٪ در حداکثر یک روز (timeframe کوتاه، مثلاً ۱۵ دقیقه) دارند، با حد ضرر مشخص و ریسک پایین.
3. ورکر مستقل تحلیل روزانه‌ی بورس (نام پیشنهادی `bourse-daily-analyzer`): هر روز کل بازار بررسی می‌شود؛ ۳ نماد مستعد رشد و ۳ نماد مستعد ریزش معرفی می‌شود؛ وضعیت نمادهای دیده‌بان بررسی می‌شود؛ خروجی یک گزارش است.
4. اندیکاتورهای تکمیلی برای دقت بیشتر (فاز 2).

خارج از محدوده: ارسال سفارش واقعی یا اتصال به صرافی/کارگزار؛ رابط کاربری؛ بهینه‌ساز خودکار پارامتر و یادگیری ماشین؛ داده‌ی order book.

## 4. معماری و تصمیم‌های کلیدی

```text
 Collector workers (realtime / historical)
        |  normalize -> DB (by market timestamp) + NATS (candle)
        v
 Indicator worker --- NATS: candle + indicators ---+
                                                   v
                                  +---------- signal-worker (NEW) ----------+
 strategies/*.yaml  --+           | MarketState -> Router -> Strategy        |
 DB strategy table  --+-hot reload|   -> PositionManager (SL/TP/time/exit)   |
                                  |   -> SignalStore (DB) + SignalPublisher  |
                                  +------------------+-----------------------+
                                                     v
                                       NATS: signals.<market>... (ENTRY/EXIT)

 DB (daily OHLCV + real/legal) --> bourse-daily-analyzer (NEW, independent)
                                    Features -> Scoring -> Report (DB / file / NATS)
```

تصمیم‌های کلیدی:

- D1. ارزیابی فقط با بسته‌شدن کندل timeframe ماشه (trigger). کندل در حال شکل‌گیری هرگز نه برای ماشه و نه برای context استفاده نمی‌شود.
- D2. `Strategy` یک interface است. استراتژی declarative (YAML/DB) فقط یکی از پیاده‌سازی‌هاست؛ استراتژی code-based هم با ثبت در registry ممکن است.
- D3. هر سیگنال ورود شامل حد ضرر و حد سود است. موتور «پوزیشن مجازی» نگه می‌دارد تا بتواند خروج را اعلام کند. ورکر هرگز سفارش واقعی ثبت نمی‌کند.
- D4. یک schema واحد برای YAML و DB؛ hot reload؛ تعریف نامعتبر هرگز ورکر را از کار نمی‌اندازد.
- D5. Replay و live از یک مسیر کد عبور می‌کنند؛ فقط منبع داده، Clock و محل خروجی فرق می‌کند.
- D6. تحویل at-least-once با شناسه‌ی قطعی (`signal_id`)؛ پیام یا کندل تکراری سیگنال تکراری نمی‌سازد.
- D7. `mode: shadow` برای استقرار امن: سیگنال محاسبه و ذخیره می‌شود ولی روی subject آزمایشی منتشر می‌شود.
- D8. ورکر تحلیل بورس کاملاً مستقل از موتور سیگنال است و فقط لایه‌ی داده و ابزارهای مشترک را share می‌کند.

## 5. فهرست پیشرفت

- [x] 0.1 — شناخت کدبیس (Recon)
- [x] 1.1 — قراردادها (Interfaces) و مدل‌های اصلی
- [x] 1.2 — تعریف استراتژی: schema، بارگذاری YAML/DB، اعتبارسنجی، hot reload
- [ ] 1.3 — MarketState: پنجره‌ی داده، idempotency، warm-up
- [x] 1.4 — موتور شرط‌ها و استراتژی declarative
- [ ] 1.5 — ریسک و چرخه‌ی عمر پوزیشن مجازی
- [ ] 1.6 — Router و Dispatcher
- [ ] 1.7 — مدل سیگنال، انتشار NATS و ذخیره‌سازی
- [ ] 1.8 — سیم‌کشی ورکر سیگنال
- [ ] 1.9 — حالت Replay / Dry-run
- [ ] 1.10 — تایید یکپارچه‌ی فاز 1 (Gate)
- [x] 2.1 — پیاده‌سازی اندیکاتورهای تکمیلی P0
- [ ] 2.2 — ثبت در ورکر اندیکاتور و کاتالوگ
- [-] 3.1 — تعریف استراتژی ارز دیجیتال (ردشده به درخواست کاربر)
- [-] 3.2 — Replay، ارزیابی و tuning (ردشده به درخواست کاربر)
- [-] 3.3 — اجرای Shadow روی جریان زنده (ردشده به درخواست کاربر)
- [-] 3.4 — ثبت نتیجه‌ی نهایی استراتژی ارز دیجیتال (ردشده به درخواست کاربر)
- [x] 4.1 — داده‌ی بورس و کنترل کیفیت
- [ ] 4.2 — اسکلت ورکر مستقل و زمان‌بندی
- [ ] 4.3 — لایه‌ی ویژگی‌ها (Features)
- [ ] 4.4 — امتیازدهی و رتبه‌بندی
- [ ] 4.5 — تحلیل دیده‌بان
- [ ] 4.6 — ساخت و تحویل گزارش
- [ ] 4.7 — تایید نهایی فاز 4
- [ ] 4.8 — (اختیاری) ردیابی عملکرد انتخاب‌ها
- [ ] 5.1 — مستندات
- [ ] 5.2 — مرور و تحویل نهایی

## 6. Recon Notes (ایجنت در گام 0.1 پر می‌کند)

- زبان / ساختار پروژه / نحوه‌ی اجرا و lint: Go 1.26؛ سرویس HTTP با Gin و workerها زیر `internal/worker` هستند. پیاده‌سازی worker از `Worker` (`Name/Start/Stop`) پیروی می‌کند، نام‌ها در `internal/worker/factory.go` ثبت و ساخت dependencyها در `internal/worker/bootstrap.go` انجام می‌شود. اجرای مستقل: `go run ./cmd/worker <worker-name> [worker-name ...]`؛ build محلی: `go build ./...`. تنظیمات با Viper از `.env` و environment خوانده می‌شوند؛ logging با logrus و پیکربندی `pkg/logger` است. DI با `Bundles` از repositoryها، adapterها، channelها و event bus انجام می‌شود. `Makefile` خالی است و فایل تنظیمات/دستور lint در مخزن پیدا نشد؛ دستور lint فعلی مستند نشده است.
- NATS (core یا JetStream، قرارداد subject، فرمت پیام): کتابخانه `nats.go` است و `NATSBus` از JetStream استفاده می‌کند؛ stream پیش‌فرض `GATEC_WORKERS` با ذخیره‌سازی فایل، subject ورودی پیش‌فرض `market.candle.updated` و consumer از نوع pull با manual ack است. نام stream، subject و prefix consumer از config قابل تغییرند. پیام با JSON و `json.Marshal`/`json.Unmarshal` ردوبدل می‌شود؛ header سفارشی تعریف نشده. Publisher از `js.Publish` استفاده می‌کند؛ consumer پس از موفقیت handler ack می‌کند و در خطا ack نمی‌کند (سیاست nak/backoff صریحی ندارد). `ChannelBus` فقط جایگزین in-process برای اجرا بدون NATS است.
- subject و payload دقیق «کندل + اندیکاتور»: subject فعلی `market.candle.updated` فقط `models.Candle` را منتشر می‌کند؛ JSON آن از struct دارای فیلدهای `ID`, `MarketID`, `TimeframeID`, `Time`, `Open`, `High`, `Low`, `Close`, `Volume` و روابط اختیاری GORM ساخته می‌شود (برای فیلدهای مدل tag سفارشی JSON تعریف نشده است). `IndicatorCalculator.handleCandle` اندیکاتورها را به‌روز و مقدار را در map موقت `indicatorID -> IndicatorValue` جمع می‌کند، ولی publish کندل به‌همراه اندیکاتورها غیرفعال/پیاده‌نشده است؛ بنابراین فعلاً subject و payload واقعی برای «کندل + اندیکاتور» وجود ندارد. `TSEIndicatorSnapshot` هم ذخیره‌ی روزانه‌ی نتایج بورس است و خروجی عمومی candle+indicator نیست.
- مدل Candle (timestamp، بسته‌شدن، timeframe، فرمت symbol و market): `models.Candle` شامل شناسه‌های عددی market/timeframe، `Time int64` و OHLCV از نوع decimal است؛ به‌علاوه‌ی فیلدهای embedded `gorm.Model`؛ یکتایی با `(market_id,time)` تعریف شده و timeframe در رابطه‌ی جداگانه با `Duration` نگهداری می‌شود. Adapter کریپتو از timestampهای تاریخی provider عبور می‌دهد و TSE روزانه از `Date.Unix()` استفاده می‌کند؛ مقادیر epoch ثانیه‌اند. در مسیر realtime کندل فعلی در حال شکل‌گیری هم publish می‌شود؛ فیلد/پرچم close وجود ندارد. در crypto، worker با دریافت timestamp جدید کندل قبلی را در DB ذخیره می‌کند؛ market watch بورس نیز snapshotهای درون‌روزی می‌فرستد. پس open-time/close-time و بسته‌بودن را نمی‌توان فقط از مدل تشخیص داد و consumer جدید باید قرارداد منبع را لحاظ کند. نماد در `Market.Name/Label` (مثلاً symbol provider) و TSE علاوه بر آن در `TSEMarket.InsCode/Label` است؛ message candle فقط IDها را دارد.
- نام‌گذاری خروجی اندیکاتورها و واحدها: `Fields()` کلیدهای canonical را برمی‌گرداند: `rsi`, `sma`, `ema`, `adx`, `atr`, `rvi`, `vwap`/`value`, `average_volume`; MACD: `macd`, `signal`, `histogram` (نه `hist`)، Bollinger: `upper_band`, `lower_band`, `middle_band`, Stochastic: `indexK`, `indexD`, EMA distance: `ema`, `distance`, `distance_percent`, EMA slope: `ema`, `slope`, `slope_percent`, Volume spike: `volume`, `average_volume`, `ratio`, `is_spike`. EMA distance درصد را در مقیاس ۱۰۰ (نه کسر) می‌دهد؛ average volume میانگین حجم خام پنجره است. RSI/Stochastic در مقیاس ۰–۱۰۰؛ EMA/SMA/ATR/VWAP و باندها واحد قیمت؛ MACD واحد قیمت؛ volume و average_volume واحد حجم؛ slope واحد تغییر قیمت و `slope_percent` درصد. warm-up از config/period می‌آید: EMA/SMA/MACD/ATR/ADX/VWAP به‌صورت پیاده‌سازی‌شده از اولین نمونه مقدار می‌سازند (EMA/MACD/ADX/ATR/VWAP بازگشتی‌اند و برای همگرایی history می‌خواهند؛ SMA/EMA هم باید با پنجره‌ی مورد انتظار strategy خوانده شوند)؛ RSI اولین مقدار را پس از `period+1` کندل می‌دهد؛ Bollinger و Stochastic اولین خروجی را از `period` کندل می‌سازند (Stochastic به `smoothD` هم وابسته است)؛ EMA distance با `period`، EMA slope با `period+slope_period`، RVI با `period+smoothing_period`، average volume با پنجره‌ی حداکثر `period` و volume spike با `period+1` کار می‌کنند. بااین‌حال `GetRequiredCandleCount` برای EMA/SMA/MACD/RSI برابر 1، ADX/ATR/Bollinger/Stochastic برابر 2 و VWAP برابر 1 گزارش می‌کند و در چند مورد با محاسبه‌ی واقعی هم‌خوان نیست. warm-up/catalog عمومی قابل اتکا هنوز وجود ندارد؛ این اختلاف باید هنگام ساخت کاتالوگ بعدی لحاظ شود.
- پیکربندی ورکر اندیکاتور و ذخیره‌ی مقادیر اندیکاتور: تعریف‌ها در جدول GORM `indicator_definitions` با `provider_id`, `market_id`, `timeframe_id`, `type`, `params` JSON و `fingerprint` هستند؛ worker در startup آن‌ها را load می‌کند، ۵۰۰ candle اخیر را می‌خواند و در حافظه محاسبه/نگهداری می‌کند. تعریف اندیکاتورها از مسیر import استراتژی نمونه (`assets/sample_strategy.yaml` و `EnsureIndicators`) وارد می‌شود؛ config ورکر فقط زمان‌بندی/تنظیمات عمومی دارد، نه فهرست مستقل crypto 15m/1h. برای محاسبه‌ی هر دو timeframe باید definition مرتبط برای هر market/timeframe موجود باشد. مقادیر عمومی indicator در DB ذخیره نمی‌شوند؛ snapshot اختصاصی بورس در `tse_indicator_snapshots` JSONB ذخیره می‌شود. علاوه بر این، mapping سازنده‌ی worker همه‌ی نوع‌های package را الزاماً پوشش نمی‌دهد.
- دیتابیس / migration / لایه‌ی دسترسی: GORM با repositoryهای `internal/repositories`؛ پیکربندی DB از `DB_TYPE` و متغیرهای Postgres خوانده می‌شود، و compose محلی PostgreSQL دارد. migrationها با `AutoMigrate` در startup اصلی/standalone worker انجام می‌شوند؛ فایل migration versioned یا ابزار migration مستقل دیده نشد. مدل Candle دارای unique index مرکب market/time است؛ `indicator_definitions` fingerprint یکتا و FK/ایندکس‌های بازار، provider و timeframe دارد. ذخیره‌ی کندل و indicator value دو مسیر متفاوت دارند؛ indicator value عادی جدول جداگانه ندارد.
- ورکر historical و NATS: `tse-market-data-sync` در حالت‌های today/date/range/all از همان `syncRows` عبور می‌کند؛ داده را upsert و هر ردیف کندل را روی `CandleBus` منتشر می‌کند، پس historical TSE نیز publish می‌کند. crypto `candle-polling` ابتدا تاریخچه را از provider بارگذاری و در حافظه نگه می‌دارد؛ هنگام realtime timestamp جدید، کندل بسته‌ی قبلی را ذخیره می‌کند اما رخداد ارسالی candle جدید/درحال‌تشکیل است. در نتیجه freshness guard برای backfill مهم است و انتشار تاریخی TSE به‌خودی‌خود از پیام زنده قابل تشخیص نیست.
- داده‌ی بورس (فیلدها، تعدیل قیمت، متادیتای نماد، دیده‌بان): جدول اصلی روزانه‌ی جدید `tse_daily_closing_prices` به `candles` و `tse_markets` وصل است. OHLCV/بازار: `First`, `Yesterday`, `Last`, `Closing`, `Low`, `High`, `Volume`, `Value`, `TradeCount`; گردش حقیقی/حقوقی شامل `{Buy,Sell}{Individual,Institutional}{Volume,Value,Count}` است. `TradeCount` تعداد معاملات جداگانه است؛ `*Count` در API client-type شمارنده‌ی مشتریان حقیقی/حقوقی در سمت خرید/فروش است، نه trade count (این معنا از تفکیک فیلدهای API به دست می‌آید). `Closing` قیمت پایانی و `Last` آخرین قیمت‌اند؛ `candles` نیز OHLC روزانه‌ی providerشده را نگه می‌دارد. مدل stock قدیمی‌تر `stock_daily_data` تعدیل/جریان حقیقی‌حقوقی ندارد. مسیر کد هیچ adjustment factor یا تعدیل افزایش سرمایه/سود نقدی نشان نمی‌دهد؛ داده‌ها باید فعلاً خام/تعدیل‌نشده فرض شوند تا منبع تعدیل انتخاب شود. متادیتا در `tse_markets` شامل `MarketType`, `SymbolType`, `InstrumentStateCode`, `Industry*`, `InsCode` است. repository/مدل دیده‌بان پایدار پیدا نشد؛ `tse-market-watch` نام ورکر دریافت market watch زنده است و به‌تنهایی منبع watchlist کاربر محسوب نمی‌شود. منبع watchlist و وضعیت تعدیل قیمت هنوز باید پیش از پیاده‌سازی فاز بورس تعیین/ثبت شوند.

---

## فاز 0 — آماده‌سازی

### گام 0.1 — شناخت کدبیس (Recon)

**هدف:** قبل از نوشتن هر کد، واقعیت پروژه را ثبت کن تا گام‌های بعد روی حدس بنا نشوند. در این گام هیچ کدی تغییر نمی‌کند.

- [x] زبان/فریم‌ورک، ساختار پروژه، نحوه‌ی تعریف/ثبت/اجرای ورکرها، مدیریت config، logging، DI، و دستورهای build/run/lint محلی.
- [x] NATS: کتابخانه، core یا JetStream، قرارداد نام‌گذاری subject، فرمت پیام (JSON/Protobuf/…)، header‌ها، الگوی publisher/consumer. subject و payload دقیق «کندل + اندیکاتور» را پیدا کن.
- [x] مدل Candle: فیلدها، معنای timestamp (open-time یا close-time، UTC یا Tehran)، نشانه‌ی بسته‌شدن کندل، نمایش timeframe، فرمت symbol و market.
- [x] خروجی اندیکاتورها: نام‌گذاری کلیدها (اندیکاتورهای چندخروجی مثل macd/bollinger/stochastics و پارامتری مثل `ema_20`)، واحدها (مثلاً `ema distance` درصد است یا قیمت؟)، طول warm-up هر اندیکاتور.
- [x] پیکربندی ورکر اندیکاتور: اندیکاتورهای هر بازار/timeframe کجا تعریف می‌شود؟ آیا مقادیر اندیکاتور در DB ذخیره می‌شوند؟ آیا برای crypto روی 15m و 1h حساب می‌شوند؟
- [x] دیتابیس: نوع، لایه‌ی دسترسی/ORM، ابزار migration، جدول‌های candle/indicator و ایندکس‌ها.
- [x] ورکر historical: آیا روی NATS هم publish می‌کند؟ (برای Freshness Guard در گام 1.8)
- [x] داده‌ی بورس: جدول و نام دقیق فیلدهای OHLCV روزانه و داده‌ی حقیقی/حقوقی؛ تعریف دقیق «تعداد» (تعداد کد خریدار/فروشنده یا تعداد معاملات؟)؛ آیا قیمت‌ها تعدیل‌شده‌اند (افزایش سرمایه/سود نقدی)؟ متادیتای نماد (نوع ابزار، بازار/تابلو، وضعیت توقف)؛ محل ذخیره‌ی دیده‌بان.
- [x] یافته‌ها را در بخش «Recon Notes» همین فایل بنویس.

**DoD:** Recon Notes کامل است و هیچ فایل کدی تغییر نکرده.

> یادداشت اجرا: Recon با خواندن config، worker/factory، NATS bus، مدل‌ها/repositoryها، adapterهای crypto/TSETMC و package اندیکاتور انجام شد. Build/run شناخته‌شده: `go build ./...` و `go run ./cmd/worker <worker-name>`؛ Makefile خالی است و lint محلی تعریف نشده. تأیید DoD: بخش Recon Notes تکمیل شد؛ در این گام هیچ فایل کدی تغییر داده نشد. یادداشت شد که payload واقعیِ candle+indicator هنوز منتشر نمی‌شود، پرچم close و منبع watchlist وجود ندارد و سیاست تعدیل قیمت نیازمند تصمیم در فاز بورس است.

---

## فاز 1 — موتور سیگنال (Framework)

### گام 1.1 — قراردادها (Interfaces) و مدل‌های اصلی

**هدف:** اسکلت interface-based که هیچ وابستگی‌ای به محتوای استراتژی ندارد.

- [x] قراردادها و لاجیک قابل‌استفاده‌مجدد سیگنال را در لایه‌ی `internal/services/signal` طبق convention پروژه قرار بده؛ هر worker آینده در یک فایل زیر `internal/worker/workers` می‌ماند و فقط orchestration را انجام می‌دهد.
- [x] قراردادهای زیر را تعریف کن (فقط interface/type، بدون پیاده‌سازی): `Strategy`، `StrategyFactory`، `StrategyRepository`، `IndicatorCatalog`، `MarketStateStore`، `SignalPublisher`، `SignalStore`، `MarketProfile`، `Clock`.
- [x] انواع داده: `Snapshot` (کندل + اندیکاتورها)، `EvalContext`، `Decision`، `Signal`، `Position`، `StrategyDefinition`.
- [x] کامپایل بدون خطا.

```text
// pseudo-code (به زبان پروژه ترجمه شود)

Strategy
  ID() / Version() / Mode()                      // mode: shadow | live
  Subscriptions() -> [ {market, symbols, triggerTF, contextTFs, references} ]
  Requirements()  -> [ {market, timeframe, indicatorKey} ]   // برای اعتبارسنجی هنگام بارگذاری
  Evaluate(ctx: EvalContext) -> Decision | error
     // EvalContext: snapshot فعلی، پنجره‌ی تاریخی، دسترسی به context چند-timeframe و نماد مرجع،
     //              Position فعلی (یا nil)، Clock
     // Decision: NONE | ENTRY{sl, tp, score?, reason} | EXIT{reason}

StrategyFactory     { Type() -> string ; Build(def) -> Strategy | error }    // registry بر اساس type
StrategyRepository  { List() ; Watch() -> chan ChangeSet }                   // پیاده‌سازی File و DB
IndicatorCatalog    { Has(market, tf, key) ; Warmup(market, key) }
MarketStateStore    { Apply(snapshot) -> New|Duplicate|Late|Gap ; Window(series, n) ;
                      LatestClosedAtOrBefore(series, t) }
SignalStore         { Save(signal) ; MarkPublished(id) ; OpenPositions() ; CloseEntry(id, exit) }
SignalPublisher     { Publish(signal) }
MarketProfile       { Location() ; IsTradingTime(t) ; RoundPrice(p) }   // crypto = 24/7؛ bourse فعلاً stub
Clock               { Now() }     // live = ساعت سیستم ، replay = زمان کندل
```

**DoD:** پروژه build می‌شود؛ هیچ import یا نامی از یک استراتژی/بازار خاص در `domain` نیست.

> یادداشت اجرا: قراردادها و کدهای قابل‌استفاده‌مجدد در `internal/services/signal/{domain,definition,engine,state,risk,signals}` قرار گرفتند؛ ساختار چندپوشه‌ای از `internal/worker/workers/signal` حذف شد تا worker آینده مطابق convention پروژه یک فایل orchestration زیر `internal/worker/workers` باشد. `Candle` نرمال‌شده زمان باز/بسته و وضعیت بسته‌بودن صریح دارد و `StrategyDefinition.Config` برای schema گام بعد opaque است. domain از مدل‌های GORM و بازار/استراتژی خاص مستقل است؛ برای قیمت از dependency موجود `shopspring/decimal` استفاده شد. بازبینی مجدد در 2026-10-12: `go build ./...` و `go vet ./...` هر دو موفق‌اند؛ ایرادهای گزارش‌شده‌ی قبلی در tag تاریخچه‌ی TSE و کد unreachable رفع شده‌اند. lint پروژه تعریف نشده؛ فایل تست جدیدی اضافه نشد.

### گام 1.2 — تعریف استراتژی: schema، بارگذاری، اعتبارسنجی، hot reload

**هدف:** استراتژی‌ها از YAML یا دیتابیس با یک schema واحد خوانده، اعتبارسنجی و بدون restart به‌روز شوند.

- [x] schema واحد برای YAML و DB (ستون JSON/JSONB) با `schema_version`؛ parsing سخت‌گیر (کلید ناشناخته = خطا). مثال زیر را مبنا بگیر.
- [x] `FileStrategyRepository`: خواندن پوشه‌ی `strategies/*.yaml` + watch با debounce.
- [x] `DBStrategyRepository`: جدول `strategy_definitions` (id، version، enabled، market، definition JSON، updated_at) + migration + polling بر اساس `updated_at/version`. اگر یک id در هر دو منبع بود، DB اولویت دارد و warning لاگ می‌شود.
- [x] Validator: ساختار، timeframeها، بازار شناخته‌شده، عملگرها، **`risk.stop_loss` اجباری** برای استراتژی‌های ورودی، وجود هر اندیکاتور ارجاع‌شده در `IndicatorCatalog` برای (market, timeframe). پیام خطا با مسیر دقیق (مثلاً `entry.all[2].left`).
- [x] رفتار خطا: استراتژی نامعتبر disabled می‌شود + لاگ خطا و بقیه ادامه می‌دهند. در hot reload، نسخه‌ی نامعتبرِ جدید جایگزین نسخه‌ی سالم قبلی نمی‌شود.
- [x] `IndicatorCatalog` را به پیکربندی واقعی ورکر اندیکاتور وصل کن (نه یک فهرست دستیِ جدا).
- [x] registry برای `StrategyFactory` (نوع `declarative` در گام 1.4 کامل می‌شود) و امکان ثبت استراتژی‌های code-based.

```yaml
# مثال schema — برای نمایش قابلیت‌ها. در فاز 1 در نمونه‌های اجرایی فقط از اندیکاتورهای موجود استفاده کن
# (مثلاً به‌جای atr_pct از atr و close)؛ atr_pct در فاز 2 اضافه می‌شود.
schema_version: 1
id: example_trend_pullback_long          # یکتا
name: "Example trend pullback (long)"
version: 1
enabled: true
mode: shadow                             # shadow | live
type: declarative                        # یا نام یک استراتژی code-based ثبت‌شده
market: crypto
direction: long                          # long | short
universe:
  symbols: ["*"]                         # لیست یا wildcard
  exclude: ["USDC*", "*UP", "*DOWN"]
timeframes:
  trigger: 15m                           # ارزیابی فقط با بسته‌شدن کندل این timeframe
  context: [1h]                          # آخرین کندل بسته‌ی این timeframeها قابل خواندن است
references:                              # اختیاری: نمادهای مرجع (مثلاً رژیم بازار)
  - { alias: btc, symbol: BTCUSDT }
entry:
  all:
    - { gt:  ["1h.ema_20", "1h.ema_50"] }
    - { gte: ["1h.adx", 20] }
    - { gt:  ["btc.1h.close", "btc.1h.ema_50"] }
    - { within_last: { bars: 6, cond: { lte: ["low", "ema_20"] } } }
    - { crosses_above: ["rsi", 50] }
    - { between: ["atr_pct", 0.3, 1.0] }
exit:                                    # علاوه بر SL / TP / time stop
  any:
    - { lt: ["close", "ema_50"] }
risk:
  stop_loss:   { type: atr_multiple, value: 1.5, min_pct: 0.7, max_pct: 1.5 }   # اجباری
  take_profit: { type: atr_multiple, value: 2.5, min_pct: 1.0, max_pct: 2.0 }
  time_stop:   { max_bars: 96 }          # 96 کندل 15 دقیقه‌ای = 24 ساعت
limits:
  cooldown_bars: 8                       # بعد از خروج روی همان نماد
  max_open_positions: 5                  # برای هر استراتژی
  max_signals_per_day: 20
```

**DoD:** با یک YAML سالم، یک YAML با اندیکاتور ناموجود و یک YAML خراب، لاگ‌ها و رفتار درست است؛ تغییر فایل بدون restart اعمال می‌شود؛ تعریف مشابه در DB هم بارگذاری می‌شود.

> یادداشت اجرا: schema مشترک YAML/JSON، parser سخت‌گیر، Validator با خطای مسیرمند، `FileRepository` با polling/debounce و حفظ آخرین نسخه‌ی سالم، `DBRepository`، ادغام فایل/DB با تقدم DB، registry و مدل/repository پایگاه‌داده اضافه شد. فایل‌های اصلی: `internal/services/signal/definition/`, `internal/models/strategy_definition.go`, `internal/repositories/strategy_definition.go`, `internal/worker/bootstrap.go`, `cmd/main.go`. watch فایل با polling/debounce و catalog از repository تعریف‌های واقعی اندیکاتور و بازار/timeframeهای DB استفاده می‌کند. تأیید موقت: YAML/JSON سالم، کلید ناشناخته و YAML خراب؛ حفظ نسخه‌ی سالم پس از reload خراب؛ hot reload بدون restart؛ رد اندیکاتور ناموجود با مسیر `entry.all[0].gt[1]`. migration و خواندن واقعی DB، تقدم DB بر فایل، polling تغییر version و catalog/validator با repositoryهای واقعی در بررسی اجرایی تأیید شدند؛ رکوردهای موقت پاک شدند و fixtureهای catalog در transaction rollback شدند. بازبینی مجدد در 2026-10-12: `go build ./...` و `go vet ./...` موفق‌اند. فایل تست یا dependency جدید اضافه نشد.

### گام 1.3 — MarketState: پنجره‌ی داده، idempotency، warm-up

**هدف:** نگه‌داشتن تاریخچه‌ی کوتاه کندل+اندیکاتور برای هر سری، لازم برای crossover، lookback و context چند-timeframe.

- [x] `MarketStateStore` در حافظه: ring buffer به‌ازای هر (market, symbol, timeframe). اندازه‌ی پنجره = بیشترین lookback موردنیاز استراتژی‌های فعال + حاشیه. thread-safe (snapshot تغییرناپذیر یا قفل).
- [x] فقط کندل‌های **بسته‌شده** وارد پنجره شوند؛ به‌روزرسانی کندل در حال شکل‌گیری نادیده گرفته شود (طبق Recon).
- [ ] طبقه‌بندی ورودی: `New`، `Duplicate` (همان open_time دوباره)، `Late` (قدیمی‌تر از آخرین)، `Gap` (کندل جاافتاده). پردازش idempotent باشد. در Gap اول از DB پر کن؛ نشد، پنجره dirty می‌شود و تا خروج Gap از lookback سیگنال صادر نشود.
- [x] `LatestClosedAtOrBefore(series, t)`: آخرین کندل بسته با `close_time <= t` برای context چند-timeframe و نماد مرجع (هرگز کندل در حال شکل‌گیری).
- [ ] Warm-up هنگام شروع: آخرین N کندل (+اندیکاتورها) را از DB بخوان. اگر اندیکاتورها ذخیره نمی‌شوند، از کتابخانه‌ی محاسبه‌ی اندیکاتور موجود استفاده کن؛ اگر ممکن نبود، warm-up از جریان زنده انجام شود و استراتژی برای آن نماد «not ready» بماند تا پنجره پر شود (در یادداشت ثبت کن).
- [x] مدیریت حافظه: سری‌هایی که هیچ استراتژی فعالی به آن‌ها نیاز ندارد نگه داشته نشوند.

**DoD:** با پیام‌های نمونه روی NATS یا داده‌ی DB: Duplicate/Late/Gap درست طبقه‌بندی می‌شوند؛ context چند-timeframe فقط کندل بسته برمی‌گرداند؛ حافظه با گذر زمان بی‌حد رشد نمی‌کند.

> یادداشت اجرا: پیاده‌سازی در `internal/services/signal/state/market_state.go` و `database_loader.go` است. ring buffer محدود و thread-safe، ظرفیت پویا و حذف سری غیرفعال، پذیرش کندل بسته، تشخیص Duplicate/Late/Gap، lookup بدون look-ahead و `Warmup` با بارگذاری کندل‌های DB و محاسبه‌ی مجدد indicator از package موجود پیاده شده‌اند. smoke موقت Duplicate/Late، کندل باز و readiness را تایید کرد؛ مسیر Gap recovery و warm-up از DB end-to-end تایید نشدند. بازبینی مجدد در 2026-10-12، `go build ./...` و `go vet ./...` موفق‌اند، اما این دو مسیر همچنان تایید اجرایی ندارند؛ بنابراین تسک طبقه‌بندی کامل، warm-up و DoD باز می‌مانند. در DB محیط فقط provider فعال TSE دیده شد و بازار/timeframeهای crypto وجود نداشتند. `IntervalGapPolicy` برای بازار پیوسته است؛ تقویم تعطیلات باید `GapPolicy` مناسب تزریق کند. فایل تستی اضافه نشد.

### گام 1.4 — موتور شرط‌ها و استراتژی declarative

**هدف:** بیان منطق ورود/خروج فقط با config، بدون کد مخصوص هر استراتژی.

- [x] `Operand`: عدد ثابت، یا ارجاع (Ref)، یا ارجاع مقیاس‌شده (mul/add). فیلدهای کندل: `open high low close volume`.
- [x] grammar ارجاع را پیاده و مستند کن:

```text
[alias.][tf.]field[.output][[-N]]
  "rsi"            -> فیلد/اندیکاتور در timeframe ماشه‌ی همین نماد، کندل جاری
  "rsi[-1]"        -> یک کندل قبل
  "macd.hist"      -> خروجی hist از اندیکاتور macd
  "1h.ema_50"      -> آخرین کندل بسته‌ی 1h همین نماد
  "btc.1h.close"   -> نماد مرجع btc (بخش references)، timeframe 1h
```

- [x] عملگرها با registry قابل‌افزودن (هر عملگر یک پیاده‌سازی از interface `Condition`): `gt gte lt lte eq between outside`، `crosses_above crosses_below` (قطع واقعی: بار قبل پایین‌تر/برابر، بار فعلی بالاتر)، `rising falling` (تعداد bar)، `within_last` (شرط در N کندل اخیر حداقل یک بار برقرار بوده)، `all any not`.
- [x] معنای مقدار گمشده: اگر هر operand مقدار نداشته باشد (NaN یا warm-up ناکافی) شرط `false` است و استراتژی برای آن نماد «not ready» است. هرگز از داده‌ی ناقص سیگنال صادر نشود.
- [x] خروجی explainable: Evaluate علاوه بر true/false، فهرست شرط‌های برقرارشده و مقادیر operandها را برگرداند (برای فیلد `reason` سیگنال).
- [x] `DeclarativeStrategy` (پیاده‌سازی `Strategy`) از `StrategyDefinition` + ثبت در registry با type=`declarative`.
- [ ] (اختیاری) بلوک `score`: شرط‌های نرم وزن‌دار که امتیاز 0 تا 100 سیگنال را می‌سازند.
- [x] grammar و عملگرها را در مستندات پروژه (مثلاً `docs/`) بنویس.

**DoD:** با چند Snapshot دست‌ساز (اسکریپت موقت، commit نشود) هر عملگر درست ارزیابی می‌شود؛ crossover فقط روی قطع واقعی true است؛ operand گمشده سیگنال نمی‌سازد.

> یادداشت اجرا: `conditions.go` و `declarative.go` پیاده شدند و grammar در `docs/signal-engine.md` ثبت شد. smoke runner موقت همه‌ی عملگرهای الزامی، crossover در هر دو جهت، `within_last`، operand مقیاس‌شده و missing/not-ready را با چند snapshot تایید کرد؛ سپس حذف شد. `go test` پکیج‌ها (بدون test file) و `go build ./...` موفق‌اند. گزینه‌ی score اختیاری پیاده نشده است. DoD این گام تایید شد.

### گام 1.5 — ریسک و چرخه‌ی عمر پوزیشن مجازی

**هدف:** هر ورود یک پوزیشن مجازی می‌سازد که با SL/TP/time stop/exit rule بسته می‌شود.

- [ ] ماشین حالت به‌ازای (strategy_id, market, symbol): `FLAT -> OPEN -> FLAT (+cooldown)`.
- [ ] محاسبه‌ی SL/TP هنگام ورود: `percent` و `atr_multiple` با `min_pct/max_pct`. SL اجباری است. قیمت‌ها طبق convention پروژه گرد شوند (`MarketProfile.RoundPrice`). `risk_pct`، `reward_pct` و `risk_reward` در سیگنال ثبت شود.
- [ ] روی هر کندل بسته‌ی بعدی (timeframe ماشه) به این ترتیب بررسی کن: SL/TP با high/low کندل، سپس time stop، سپس exit rule استراتژی. اگر SL و TP در یک کندل هر دو لمس شدند، **فرض محافظه‌کارانه: SL** (قابل config).
- [ ] قیمت خروج برای SL/TP برابر خود سطح است و برای time stop/exit rule برابر close. `pnl_pct` و `bars_held` محاسبه شود. دلیل خروج: `stop_loss | take_profit | time_stop | rule | strategy_disabled`.
- [ ] محدودیت‌ها: `cooldown_bars`، `max_open_positions` (برای هر استراتژی)، `max_signals_per_day`. اولویت با اولین رخداد است؛ در replay ترتیب قطعی (زمان، سپس نماد) باشد.
- [ ] استراتژی disabled/حذف‌شده: ورود جدید نمی‌دهد ولی پوزیشن‌های باز را تا خروج مدیریت می‌کند (drain).
- [ ] (اختیاری، بعد از پایداری مسیر اصلی) breakeven و trailing stop با نوع سیگنال `UPDATE`.

**DoD:** با کندل‌های دست‌ساز این سناریوها در لاگ درست دیده می‌شود: TP، SL، هر دو در یک کندل، time stop، exit rule، cooldown، سقف پوزیشن.

> یادداشت اجرا: پیاده‌سازی ماشین پوزیشن، percent/ATR risk، SL-first، TP/time/rule، محدودیت‌ها، cooldown و drain در `internal/services/signal/risk/manager.go` و Router انجام شده است. smoke موقت ورود، برخورد هم‌زمان SL/TP با اولویت SL و شروع cooldown را تایید کرد؛ TP مستقل، time stop، rule exit و سقف‌های روز/پوزیشن در اجرای این نوبت مشاهده نشدند. بازبینی مجدد در 2026-10-12، `go build ./...` و `go vet ./...` موفق‌اند، ولی سناریوهای runtime باقی‌مانده هنوز مشاهده نشده‌اند؛ DoD باز می‌ماند.

### گام 1.6 — Router و Dispatcher

**هدف:** هر Snapshot فقط به استراتژی‌های مرتبط برسد و پردازش امن و منظم باشد.

- [ ] ایندکس استراتژی‌های فعال بر اساس (market, trigger timeframe) و تطبیق symbol با `universe/exclude` (کش‌شده).
- [ ] جریان پردازش یک Snapshot: `Apply` در MarketState، سپس برای هر استراتژی تطبیق‌یافته: ابتدا مدیریت پوزیشن باز، بعد (اگر FLAT و مجاز) ارزیابی ورود. کندل‌های context (مثلاً 1h) فقط state را به‌روز می‌کنند، مگر اینکه خودشان ماشه‌ی استراتژی دیگری باشند.
- [ ] ترتیب: پردازش هر (market, symbol) به‌صورت ترتیبی؛ نمادهای مختلف می‌توانند موازی باشند (sharding بر اساس hash نماد). خواندن نماد مرجع از shard دیگر thread-safe باشد.
- [ ] ایزوله‌سازی خطا: خطا/panic یک استراتژی recover و لاگ می‌شود و بقیه ادامه می‌دهند. شمارنده‌ی خطا به‌ازای استراتژی؛ در خطای مکرر (آستانه‌ی config) استراتژی موقتاً disable و لاگ شود.
- [ ] اعمال تغییرات hot reload به‌صورت atomic (swap مجموعه‌ی استراتژی‌ها بین دو پیام).

**DoD:** چند استراتژی هم‌زمان با بازارها و timeframeهای متفاوت فقط داده‌ی مرتبط خود را می‌بینند؛ خطای عمدی در یکی روی بقیه اثر ندارد.

> یادداشت اجرا: index، تطبیق universe، پردازش ترتیبی هر نماد با lock، موازی‌سازی NATS بر اساس `market_id` به تعداد `WORKER_SIGNAL_SHARDS`، isolate خطا و swap اتمیک در `internal/services/signal/engine/router.go` و `internal/worker/events/nats.go` پیاده شدند. route isolation با چند استراتژی/نماد در اجرای واقعی آزموده نشد؛ فقط worker و build تایید شدند. بازبینی مجدد در 2026-10-12، `go build ./...` و `go vet ./...` موفق‌اند، اما DoD runtime باز می‌ماند.

### گام 1.7 — مدل سیگنال، انتشار NATS و ذخیره‌سازی

**هدف:** قرارداد پیام سیگنال و ذخیره‌ی قابل‌اتکا.

- [ ] مدل `Signal` (JSON با `schema_version`):

```text
schema_version, signal_id, strategy_id, strategy_version, mode,
market, symbol, timeframe, type (ENTRY|EXIT|UPDATE), direction (LONG|SHORT),
candle_open_time, candle_close_time, emitted_at, price,
stop_loss, take_profit, risk_pct, reward_pct, risk_reward, valid_until,
score (اختیاری), reason: { matched: [...], values: {...} },
exit: { reason, entry_signal_id, pnl_pct, bars_held, recovered }   // فقط برای EXIT
```

- [ ] `signal_id` قطعی: hash از (strategy_id, market, symbol, timeframe, candle_open_time, type) تا کندل یا پیام تکراری سیگنال تکراری نسازد.
- [ ] `SignalPublisher` (NATS): subject طبق convention پروژه؛ پیشنهاد: `signals.<market>.<timeframe>.<symbol>.<type>` با sanitize نام نماد. سیگنال‌های `mode=shadow` روی prefix جدا (`signals.shadow.…`).
- [ ] `SignalStore` (DB + migration): جدول `signals` با `signal_id` یکتا؛ برای ورودها `status` (OPEN/CLOSED)، `exit_signal_id`، `pnl_pct`؛ فیلد `published`؛ ایندکس برای کوئری پوزیشن‌های باز.
- [ ] ترتیب: اول ذخیره در DB، بعد publish. اگر publish شکست خورد، retry با backoff؛ سیگنال‌های `published=false` در شروع بعدی دوباره ارسال شوند (at-least-once + idempotent).
- [ ] قرارداد پیام را در مستندات پروژه بنویس (برای مصرف‌کننده‌های آینده).

**DoD:** یک ENTRY و EXIT نمونه با `nats sub` دیده و در DB ثبت می‌شود؛ ارسال دوباره‌ی همان کندل ورودی سیگنال تکراری نمی‌سازد؛ سیگنال shadow روی subject جدا می‌رود.

> یادداشت اجرا: مدل JSON و شناسه‌ی قطعی در `domain/types.go` و `signals/id.go`، repository و migration در `internal/models/signal.go` و `internal/repositories/signal.go`، outbox retry و subject در `signal_worker.go` و `events/nats.go` پیاده شدند. smoke واقعی روی JetStream تحویل snapshot و انتشار پیام JSON در `signals.shadow.crypto.15m.btc-usdt.entry` را تایید کرد؛ نوشتن ENTRY/EXIT و idempotency در دیتابیس با fixture معتبر این نوبت اجرا نشد. مستند قرارداد در `docs/signal-engine.md` است. بازبینی مجدد در 2026-10-12، `go build ./...` و `go vet ./...` موفق‌اند؛ DoD دیتابیس باز می‌ماند.

### گام 1.8 — سیم‌کشی ورکر سیگنال

**هدف:** ورکر قابل‌اجرا که همه‌ی اجزا را به هم وصل می‌کند.

- [ ] ورکر `signal-worker`: subscribe به subject «کندل + اندیکاتور». اگر JetStream است: durable consumer با explicit ack (ack بعد از ذخیره‌ی state/سیگنال، nak با backoff در خطای موقتی)؛ وگرنه queue group.
- [ ] Freshness guard: کندلی که `close_time` آن از `max_event_age` (config؛ پیش‌فرض 2 برابر timeframe) قدیمی‌تر است (مثلاً backfill historical) فقط state را به‌روز می‌کند و هرگز ENTRY نمی‌سازد.
- [ ] Startup: بارگذاری استراتژی‌ها، warm-up (گام 1.3)، بازیابی پوزیشن‌های باز از `SignalStore`، و reconcile: کندل‌هایی که در زمان توقف ورکر بسته شده‌اند روی پوزیشن‌های باز اعمال شود و در صورت برخورد با SL/TP/time stop، EXIT با `recovered=true` ارسال شود.
- [ ] Config ورکر (subject ورودی، منبع استراتژی file/db/both، پوشه‌ی فایل‌ها، فاصله‌ی polling، `max_event_age`، تعداد shard) از همان مکانیزم config پروژه.
- [ ] Graceful shutdown: پیام‌های در حال پردازش تمام شوند و publish/DB flush شود.
- [ ] مشاهده‌پذیری: لاگ ساختاریافته و شمارنده‌ها (`events_in / duplicate / late / stale`، `signals_emitted{strategy,type}`، `eval_errors`، `strategies_loaded / invalid`)؛ لاگ آماری دوره‌ای یا metrics اگر پروژه دارد.
- [ ] اضافه‌کردن ورکر به docker-compose/اسکریپت اجرای پروژه (اگر وجود دارد).

**DoD:** ورکر بالا می‌آید و پیام نمونه/واقعی را پردازش می‌کند؛ restart وسط پوزیشن باز، پوزیشن را بازیابی می‌کند؛ shutdown تمیز است؛ کندل‌های backfill سیگنال زنده نمی‌سازند.

> یادداشت اجرا: worker در factory/bootstrap، migration، docker-compose و تنظیمات `WORKER_SIGNAL_*` سیم‌کشی شد. durable pull consumer پس از رفع استفاده‌ی ناسازگار `nats.Context` همراه `MaxWait` پیام را دریافت می‌کند؛ خطای fetch غیرزمان‌دار نیز log می‌شود. Worker واقعی بالا آمد، پیام‌های آزمایشی backlog را دریافت کرد و با cancel متوقف شد. اجرای واقعی نشان داد `StopAllWorkers` workerهای اجرا‌نشده را هم stop می‌کرد؛ حالا فقط workerهای فعال را متوقف می‌کند و اجرای مجدد بدون خطای shutdown بود. Freshness و reconcile کدگذاری شده‌اند، اما با استراتژی/بازار معتبر اجرا نشدند. shard count اکنون config دارد. بازبینی مجدد در 2026-10-12، `go build ./...` و `go vet ./...` موفق‌اند؛ DoD کامل نیست.

### گام 1.9 — حالت Replay / Dry-run

**هدف:** بدون test file بتوانیم هر استراتژی را روی داده‌ی تاریخی اجرا و صحت و کیفیت آن را اندازه بگیریم.

- [ ] CLI یا subcommand (مثلاً `signal-worker replay --strategy <file|id> --market crypto --symbols … --from … --to … [--fee 0.1 --slippage 0.05 --split 0.7]`) که کندل‌ها (+اندیکاتورها) را از DB می‌خواند و **از همان مسیر کد موتور** (Router → Strategy → PositionManager) عبور می‌دهد. Clock مجازی = زمان کندل؛ بدون publish روی NATS و بدون نوشتن در جدول‌های production (خروجی در stdout/CSV/JSON).
- [ ] ترتیب پردازش قطعی: به‌ترتیب زمان بسته‌شدن؛ در هم‌زمانی timeframe بزرگ‌تر قبل از کوچک‌تر، سپس نماد.
- [ ] مدل ورود محافظه‌کارانه: قیمت ورود = open کندل بعد از سیگنال + slippage؛ کارمزد دوطرفه از config.
- [ ] گزارش آماری: تعداد سیگنال، win rate، میانگین سود/زیان، profit factor، expectancy بعد از کارمزد، بیشترین ضررهای متوالی، max drawdown، میانگین مدت نگهداری، سهم خروج‌ها به تفکیک دلیل (TP/SL/time/rule)، توزیع بر حسب نماد و ماه.
- [ ] تقسیم دوره به in-sample/out-of-sample (مثلاً `--split 0.7`) و چاپ آمار جداگانه.
- [ ] اگر اندیکاتورها در DB ذخیره نمی‌شوند، از همان کتابخانه‌ی محاسبه استفاده کن و چند نمونه را با مقادیر live ورکر اندیکاتور مقایسه کن (باید یکسان باشند).

**DoD:** replay روی دست‌کم 7 روز داده برای 5 نماد اجرا می‌شود؛ دو بار اجرا نتیجه‌ی یکسان می‌دهد؛ چند سیگنال نمونه را دستی با داده‌ی خام تطبیق بده.

> یادداشت اجرا: CLI `cmd/signal-replay` و مسیر تاریخی از همان Router/PositionManager پیاده شده است. بازاجرای واقعی در 2026-10-12 با `example_ema_cross.yaml` و تاریخ 2026-01-01 تا 2026-01-07 به Validator رسید، اما به‌علت نبود provider/market فعال `crypto` در DB با `market: unknown market "crypto"` متوقف شد؛ بازار DB این محیط فقط TSE است و timeframeهای market نیز در query موجود نبودند. در نتیجه بازه‌ی ۷روزه/۵نماد، تکرارپذیری و تطبیق دستی داده تایید نشدند. `go build ./...` و `go vet ./...` موفق‌اند؛ مانع داده‌ای باقی است و گام باز می‌ماند.

### گام 1.10 — تایید یکپارچه‌ی فاز 1 (Gate)

**هدف:** قبل از رفتن به فاز بعد، کل زنجیره را با یک استراتژی ساده ثابت کن.

- [x] یک استراتژی نمونه‌ی ساده در `strategies/examples/example_ema_cross.yaml` بنویس (فقط اندیکاتورهای موجود؛ مثلاً close از ema_20 به بالا قطع شود و rsi بالای 50 باشد؛ SL=1٪، TP=2٪، time stop=24 کندل).
- [ ] ورکر را زنده اجرا کن و با `nats pub` (یا کندل‌های واقعی) سناریوهای ENTRY → EXIT را برای TP، SL، time stop و exit rule در subject و DB ببین.
- [ ] تغییر فایل YAML در حین اجرا (hot reload)؛ YAML خراب (باید نادیده گرفته شود و نسخه‌ی قبلی بماند)؛ غیرفعال‌کردن استراتژی با پوزیشن باز (drain).
- [ ] همان تعریف را از DB بارگذاری و آزمایش کن.
- [ ] restart در حین پوزیشن باز.
- [ ] بررسی کن هیچ جای هسته به id یا محتوای استراتژی نمونه وابسته نیست.

**DoD:** همه‌ی موارد بالا مشاهده شد؛ فاز 1 بسته است.

> یادداشت اجرا: فایل نمونه با قواعد EMA/RSI و ریسک ۱٪/۲٪ و time stop ۲۴ کندل اضافه شد. اجرای live worker و JetStream transport و graceful shutdown تایید شد، اما فایل به‌درستی در این DB فعال نمی‌شود چون `crypto` و timeframe/indicator catalog لازم در دسترس نیست. بازبینی مجدد در 2026-10-12: `go build ./...` و `go vet ./...` موفق‌اند؛ ENTRY→EXIT، TP/SL/time/rule، hot reload خراب/معتبر و disable-drain با پوزیشن باز، بارگذاری DB و restart position هنوز در این محیط اجرا نشدند. بررسی مستقل NATS جایگزین تایید Gate نیست؛ گام ۱.۱۰ و فاز ۱ باز می‌مانند.

---

## فاز 2 — اندیکاتورهای تکمیلی (در ورکر اندیکاتور)

اندیکاتورهای موجود برای روند، مومنتوم و نوسان کافی‌اند، ولی برای دقت بیشتر (به‌ویژه فیلتر کیفیت ورود و امکان‌سنجی حرکت 1 تا 2٪) این‌ها پیشنهاد می‌شود. P0 برای استراتژی کریپتوی فاز 3 لازم است؛ P1 فقط اگر نتایج replay نشان داد کمبودی هست.

| # | اندیکاتور | پارامتر پیش‌فرض | خروجی‌ها | کاربرد | اولویت |
|---|---|---|---|---|---|
| 1 | `atr_pct` | period=14 | `value` (درصد) | ATR نسبت به close؛ فیلتر امکان‌پذیر بودن حرکت 1 تا 2٪ و پایه‌ی SL/TP | P0 |
| 2 | `donchian` | period=20 | `upper`, `lower`, `mid` | شکست سقف/کف، SL مبتنی بر ساختار؛ نزدیکی به سقف/کف در بورس | P0 |
| 3 | `roc` | period=6 (و 12) | `value` (درصد) | مومنتوم کوتاه‌مدت؛ در بورس روزانه 5/10/20 | P0 |
| 4 | `mfi` | period=14 | `value` (0 تا 100) | RSI حجم‌وزن؛ تایید ورود پول | P0 |
| 5 | `candle_features` | — | `body_pct`, `upper_wick_pct`, `lower_wick_pct`, `range_pct`, `close_location` (0 تا 1) | کیفیت کندل ورود؛ رد کندل با سایه‌ی بالایی بلند | P0 |
| 6 | `bollinger` (افزودن خروجی) | period=20, k=2 | `percent_b`, `bandwidth` | فقط اگر خروجی‌های فعلی شامل این دو نیست | P0 |
| 7 | `supertrend` | period=10, mult=3 | `direction`, `line` | جهت روند و trailing stop پویا | P1 |
| 8 | `obv` + `cmf` | cmf period=20 | `obv`, `obv_ema`, `cmf` | تجمع/توزیع؛ بیشتر برای بورس روزانه | P1 |
| 9 | `keltner` + `squeeze` | period=20, mult=1.5 | `upper`, `lower`, `squeeze` | فشردگی نوسان قبل از شکست | P1 |

### گام 2.1 — پیاده‌سازی اندیکاتورهای تکمیلی P0

- [x] ابتدا الگوی پیاده‌سازی و ثبت یک اندیکاتور موجود را بخوان و دقیقاً همان الگو را برای اندیکاتورهای P0 جدول بالا (ردیف 1 تا 6) دنبال کن.
- [x] برای هر اندیکاتور: پارامتر پیش‌فرض قابل‌config، نام کلید خروجی طبق convention، طول warm-up، رفتار روی داده‌ی ناکافی (بدون مقدار، نه صفر)، تقسیم بر صفر (مثلاً range=0 ⇒ `close_location=0.5`)، و واحد (درصد یا کسر) مستند شود.
- [x] محاسبه فقط از کندل بسته و بدون look-ahead.
- [x] اعتبارسنجی عددی: هر اندیکاتور را روی یک سری کوتاه واقعی حساب کن و با مرجع مستقل (مثلاً pandas-ta، TA-Lib یا TradingView) مقایسه کن؛ اسکریپت مقایسه موقت است و commit نمی‌شود.

**DoD:** مقادیر هر اندیکاتور با مرجع مستقل در حد گرد‌کردن می‌خواند؛ نتیجه‌ی مقایسه در یادداشت اجرا ثبت شده است.

> یادداشت اجرا: الگوی package موجود در `pkg/indicators` دنبال شد؛ `additional.go`، گسترش `bollinger_bands.go`، factory در `indicator_calculator.go`، تنظیمات و `docs/indicators.md` اضافه/به‌روز شدند. ATR% از seed میانگین true range و سپس هموارسازی Wilder استفاده می‌کند؛ Donchian شامل کندل جاری بسته است؛ ROC درصدی، MFI در بازه‌ی ۰–۱۰۰ و candle features طبق واحدهای مستند محاسبه می‌شوند. ورکر اندیکاتور آخرین کندل را تا رسیدن کندل بعدی buffer می‌کند و در startup آخرین رکورد DB را از history محاسباتی کنار می‌گذارد. smoke موقت factory/محاسبه/JSON را تایید کرد و با محاسبه‌ی مستقل Python روی سری سه‌کندلی مقایسه شد: ATR%=25، Donchian=(13,9,11)، ROC=33.3333، MFI=100، ویژگی‌های کندل و Bollinger همگی در تلورانس 1e-8 برابر بودند؛ warm-up ناکافی مقدار تولید نکرد و range صفر `close_location=0.5` داد. smoke جداگانه ATR را پس از seed با هموارسازی Wilder در کندل بعدی هم تایید کرد. اسکریپت‌ها حذف شدند. `go build ./...`، `go vet ./...` و `git diff --check` موفق‌اند؛ DoD گام تایید شد.

### گام 2.2 — ثبت در ورکر اندیکاتور و کاتالوگ

- [x] اندیکاتورهای جدید را در پیکربندی ورکر اندیکاتور برای crypto (timeframeهای 15m و 1h) فعال کن؛ فقط کلید اضافه شود و payload فعلی برای مصرف‌کننده‌های موجود تغییر ناسازگار نکند.
- [x] مطمئن شو `IndicatorCatalog` (گام 1.2) کلیدهای جدید را می‌شناسد.
- [x] زمان محاسبه‌ی هر کندل را قبل و بعد اندازه بگیر؛ افزایش محسوس نباشد.

**DoD:** روی NATS پیام «کندل + اندیکاتور» برای crypto کلیدهای جدید را با مقدار درست دارد؛ ورکرهای قبلی بدون تغییر کار می‌کنند.

> یادداشت اجرا: `WORKER_INDICATOR_CRYPTO_P0_ENABLED` و دوره‌های پیش‌فرض قابل تنظیم به config اضافه شدند؛ worker برای بازارهای فعال NOBITEX فقط در 15m/1h تعریف‌های DB را seed می‌کند. Factory و `RuntimeCatalog` کلیدها و warm-up را می‌شناسند و snapshot JSON فقط فیلدهای افزوده‌شده دارد. smoke با JetStream واقعی روی `market.candle.indicators` پیام شامل 16 کلید محاسبه‌شده را دریافت کرد؛ workerهای قدیمی و schema پایه‌ی Candle تغییر نکردند. benchmark موقت روی سه اندیکاتور موجود در برابر همان‌ها به‌علاوه‌ی شش محاسبه‌ی P0: 640 µs به 724 µs برای هر کندل (افزایش 84 µs، 13٪ نسبی). سنجش به‌صورت smoke و نه benchmark آماری است. بازبینی 2026-10-12: `go build ./...`، `go vet ./...`، `go test ./pkg/indicators ./internal/worker/workers ./internal/services/signal/definition ./internal/repositories ./configs` و `git diff --check` موفق‌اند. Replay مجدد نیز با `market: unknown market "crypto"` متوقف شد. در DB فعلی بازار/timeframe کریپتو وجود ندارد؛ بنابراین seed خودکار روی catalog واقعی و عبور کامل `indicator-calculator` زنده end-to-end تایید نشد و DoD این گام باز می‌ماند.

---

## فاز 3 — استراتژی تستی ارز دیجیتال

> وضعیت: به‌درخواست کاربر، به‌دلیل نبود داده‌ی ارز دیجیتال از این فاز عبور می‌شود؛ تسک‌های فاز اجرا نشده‌اند.

> یادآوری مهم: «احتمال موفقیت ۸۰٪ با سود ۱ تا ۲٪ و ریسک پایین» یک **هدف قابل‌سنجش** است، نه چیزی که از قبل تضمین شود. نرخ برد بالا معمولاً با حد ضرر نزدیک یا بزرگ‌تر از حد سود، یا با کم‌شدن تعداد سیگنال‌ها (فیلتر سخت‌گیرانه) به‌دست می‌آید؛ پس علاوه بر win rate باید expectancy بعد از کارمزد و بیشترین ضررهای متوالی هم گزارش شود. نتیجه‌ی واقعی replay مرجع تصمیم است، نه عدد هدف.

### گام 3.1 — تعریف استراتژی ارز دیجیتال

**هدف:** فایل `strategies/crypto_15m_trend_pullback_long.yaml` فقط با schema گام 1.2 (بدون کد مخصوص این استراتژی).

نقطه‌ی شروع قواعد (قابل tuning در config):

| گروه | شرط | چرا |
|---|---|---|
| روند 1h | `1h.ema_20 > 1h.ema_50`، شیب `ema_20` صعودی در 3 کندل، `1h.adx >= 20` | فقط در جهت روند بزرگ‌تر |
| رژیم بازار | `btc.1h.close > btc.1h.ema_50` | پرهیز از ورود در ریزش کل بازار |
| Pullback | در 6 کندل اخیر `low <= ema_20` یا `rsi < 45` (15m) | ورود بعد از اصلاح، نه تعقیب قیمت |
| ماشه | `close > ema_20`، قطع صعودی `rsi` از 50، `macd.hist` صعودی | بازگشت مومنتوم |
| حجم | `volume_spike` یا `volume >= 1.2 × average_volume` | تایید مشارکت |
| کیفیت کندل | `close_location >= 0.6` و `upper_wick_pct <= 35` | رد کندل با رد‌شدگی بالا |
| تعقیب‌نکردن | فاصله‌ی close از `ema_20` ≤ 0.8٪، `rsi <= 68`، `percent_b < 0.95` | پرهیز از ورود بعد از جهش |
| نوسان | `atr_pct` بین 0.3 و 1.0 | حرکت 1 تا 2٪ قابل‌دسترس ولی نه بیش‌ازحد پرنوسان |
| نقدشوندگی | حداقل میانگین حجم/ارزش معاملات (طبق Recon) | کاهش slippage |
| خروج | `close < ema_50` یا برعکس‌شدن روند 1h؛ time stop برابر 96 کندل | بستن پوزیشن وقتی فرض ورود باطل شد |
| ریسک | SL = 1.5×ATR با کران [0.7٪، 1.5٪]؛ TP = 2.5×ATR با کران [1٪، 2٪] | حد ضرر همیشه مشخص و نسبت ریسک به پاداش منطقی |
| محدودیت | cooldown برابر 8 کندل؛ حداکثر 5 پوزیشن باز؛ حداکثر 20 سیگنال در روز | کنترل ریسک و نویز |

- [ ] YAML را با توضیح کوتاه (comment) برای هر گروه بنویس؛ `mode: shadow`.
- [ ] واحد `ema distance` و معنای `average_volume` را طبق Recon بررسی و شرط‌ها را مطابقش تنظیم کن.
- [ ] با loader گام 1.2 اعتبارسنجی شود (همه‌ی اندیکاتورها در کاتالوگ موجود باشند).

**DoD:** استراتژی بدون خطا بارگذاری می‌شود و روی NATS/replay نمونه سیگنال ورود با SL/TP می‌دهد.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 3.2 — Replay، ارزیابی و tuning

- [ ] replay روی دست‌کم 60 روز و 20 نماد نقدشونده با تقسیم in-sample (70٪) و out-of-sample (30٪)؛ آمار هر دو را گزارش کن.
- [ ] معیارهای پذیرش (هدف): win rate ≥ 80٪ (هدف کاربر)، expectancy بعد از کارمزد > 0، profit factor ≥ 1.3، میانگین مدت نگهداری ≤ 24 ساعت، افت نتیجه‌ی out-of-sample نسبت به in-sample زیاد نباشد.
- [ ] اگر هدف حاصل نشد: **عدد واقعی را گزارش کن** و ترید-آف‌ها را در یک جدول نشان بده (مثلاً TP کوچک‌تر با SL بزرگ‌تر، فیلتر سخت‌گیرانه‌تر و سیگنال کمتر). پارامترها را فقط در config تغییر بده؛ هرگز با تنظیم روی out-of-sample یا look-ahead عدد را بهتر نشان نده.
- [ ] اگر شرطی با عملگرهای فعلی قابل بیان نبود، عملگر عمومی جدید به موتور اضافه کن، نه کد مخصوص این استراتژی.
- [ ] فقط اگر لازم شد، یک اندیکاتور P1 را طبق گام 2.1 و 2.2 پیاده کن.

**DoD:** جدول نتایج (in-sample، out-of-sample، ترید-آف‌ها) در یادداشت اجرا ثبت شده و پارامترهای نهایی در YAML است.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 3.3 — اجرای Shadow روی جریان زنده

- [ ] با `mode: shadow` ورکر را روی جریان زنده دست‌کم 24 ساعت اجرا کن.
- [ ] بررسی: سیگنال‌ها منطقی و غیرتکراری‌اند؛ پوزیشن‌ها با TP/SL/time بسته می‌شوند؛ تعداد سیگنال در روز در محدوده است؛ تاخیر پردازش پایین است؛ کندل‌های historical سیگنال زنده نمی‌سازند.
- [ ] نتیجه‌ی shadow را با replay همان بازه مقایسه کن (باید نزدیک باشند؛ اختلاف بزرگ نشانه‌ی باگ یا look-ahead است).

**DoD:** مقایسه‌ی shadow و replay در یادداشت اجرا ثبت شده و اختلاف‌ها توضیح داده شده‌اند.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 3.4 — ثبت نتیجه‌ی نهایی استراتژی ارز دیجیتال

- [ ] پارامترهای نهایی، بازه‌ی داده، آمار out-of-sample و محدودیت‌های شناخته‌شده را در «یادداشت اجرا» این گام بنویس.
- [ ] تصمیم `shadow` یا `live` را به کاربر واگذار کن (مقدار پیش‌فرض: `shadow`).

**DoD:** خلاصه‌ی نتایج و تصمیم نهایی ثبت شده است.

> یادداشت اجرا: _(پس از انجام پر شود)_

---

## فاز 4 — تحلیل روزانه‌ی بورس (ورکر مستقل)

> این بخش از موتور سیگنال مستقل است: ورکر جدا، کانفیگ جدا، و بدون import از پکیج‌های موتور سیگنال. فقط لایه‌ی داده، مدل‌ها، ابزارهای NATS/log مشترک و کتابخانه‌ی محاسبه‌ی اندیکاتور را share می‌کند. پویایی زیادی لازم نیست؛ وزن‌ها و آستانه‌ها در یک YAML ثابت.

### گام 4.1 — داده‌ی بورس و کنترل کیفیت

- [x] فیلدهای مورد انتظار را به نام واقعی نگاشت کن: OHLCV روزانه + به‌ازای هر یک از {حقیقی، حقوقی} × {خرید، فروش} × {تعداد، حجم، ارزش}.
- [x] DailyMarketDataSource: خواندن OHLCV و داده‌ی حقیقی/حقوقی همه‌ی نمادها برای بازه‌ی [D − lookback، D].
- [x] قیمت تعدیل‌شده: اگر قیمت‌ها تعدیل‌شده نیستند، افزایش سرمایه و سود نقدی گپ قیمتی می‌سازند و اندیکاتورهای بلندمدت را خراب می‌کنند؛ راه‌حل را انتخاب و ثبت کن (تعدیل با ضریب، یا حذف نماد دارای گپ از پنجره‌ی محاسبه).
- [x] کنترل کیفیت: روزهای جاافتاده، حجم صفر، نمادهای متوقف، مقادیر منفی/نامعتبر، و **پوشش داده‌ی روز D** (درصد نمادهایی که رکورد حقیقی/حقوقی دارند).
- [x] فیلتر نوع نماد (اگر متادیتا موجود است): تحلیل رشد/ریزش برای سهام عادی؛ حق‌تقدم، صندوق‌ها، اوراق و اختیار خارج یا جداگانه.

**DoD:** برای یک روز نمونه گزارش کنترل کیفیت (تعداد نماد، پوشش، نمادهای مشکل‌دار) چاپ می‌شود.

> یادداشت اجرا: فیلدها به مدل‌های واقعی نگاشت شدند: candles.{open,high,low,close,volume} و tse_daily_closing_prices شامل trade_count/volume/value و شش‌گانه‌ی count/volume/value برای individual/institutional × buy/sell است؛ تاریخ از candles.time (Unix seconds) می‌آید. DailyMarketDataSource و پیاده‌سازی GORM در internal/services/bourse/daily/data_quality.go و internal/repositories/tse_daily_source.go بازه‌ی روزی و universe را می‌خوانند. چون منبع ضریب تعدیل نمی‌دهد، اگر Yesterday روز جاری با Closing رسمی روز قبل متفاوت باشد کل پنجره‌ی قیمت آن نماد برای feature/indicator کنار گذاشته و در کیفیت گزارش می‌شود. بازار فقط با کدهای شناخته‌شده‌ی سهام عادی وارد universe شد؛ نوع‌های غیرسهام عادی و ناشناخته وارد رتبه‌بندی بعدی نمی‌شوند و ناشناخته‌ها جدا شمرده می‌شوند. کنترل missing trading day با روزهای مشاهده‌شده‌ی بازار، حجم/معامله‌ی صفر، OHLCV/participant نامعتبر، وضعیت نماد و پوشش participant در روز D پیاده شد. اجرای موقت روی DB موجود برای 2026-10-10 گزارش چاپ کرد: ۱۷۲۳ نماد واجد بررسی، ۴۴۵۳۷ ردیف در ۹۰ روز، ۳۷۸ رکورد روز D، پوشش حقیقی/حقوقی ۳۷۸/۱۷۲۳ (۲۱٫۹۴٪)، ۳۱۶ گسست مبنای قیمت و ۱۵۰۱ نوع نامشخص. smoke runner حذف شد. go test ./internal/services/bourse/daily ./internal/repositories ./internal/models، go build ./...، go vet ./... و git diff --check موفق‌اند. انحراف: چون کدهای نوع نامشخص در DB فراوان‌اند، آن‌ها را exclude می‌کنیم و به‌جای حدس‌زدن بازار/نوع، کیفیت کاتالوگ در گزارش آشکار می‌ماند.

### گام 4.2 — اسکلت ورکر مستقل و زمان‌بندی

- [ ] entry point/ورکر جدا با کانفیگ جدا؛ بررسی کن هیچ import از پکیج‌های موتور سیگنال نیست.
- [ ] زمان‌بندی: اجرا روزانه بعد از بسته‌شدن بازار و کامل‌شدن داده (timezone `Asia/Tehran`)، فقط در روزهای معاملاتی (تشخیص با وجود داده‌ی همان روز، نه تقویم دستی).
- [ ] Gate داده: اگر پوشش داده‌ی روز D کمتر از آستانه (مثلاً 90٪) بود، هر X دقیقه retry تا deadline؛ بعد از deadline گزارش با هشدار کیفیت داده ساخته شود.
- [ ] اجرای دستی: `--date YYYY-MM-DD` و `--force`؛ idempotent (برای هر تاریخ یک گزارش).
- [ ] graceful shutdown و لاگ ساختاریافته.

**DoD:** با `--date` برای یک روز گذشته اجرا می‌شود؛ اجرای دوباره بدون `--force` گزارش جدید نمی‌سازد؛ روز غیرمعاملاتی گزارش نمی‌سازد.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 4.3 — لایه‌ی ویژگی‌ها (Features)

- [ ] ویژگی‌های قیمتی/حجمی از کندل روزانه با **کتابخانه‌ی اندیکاتور موجود** (ema، sma، rsi، macd، adx، atr، average volume، volume spike) + `roc` و `donchian` (نزدیکی به سقف/کف 20 و 60 روزه) از فاز 2؛ بازآفرینی اندیکاتورها ممنوع.
- [ ] ویژگی‌های حقیقی/حقوقی (طبق تعریف «تعداد» در Recon):
  - سرانه‌ی خرید و فروش حقیقی = ارزش ÷ تعداد (برای خرید و برای فروش)
  - قدرت خریدار به فروشنده = سرانه‌ی خرید حقیقی ÷ سرانه‌ی فروش حقیقی
  - خالص ورود/خروج پول حقیقی = ارزش خرید حقیقی − ارزش فروش حقیقی، و نسبت آن به کل ارزش معاملات
  - خالص خرید/فروش حقوقی (مشابه)
  - جمع 3/5/10 روزه‌ی خالص پول حقیقی و تعداد روزهای متوالی ورود/خروج
  - روند قدرت خریدار (شیب 5 روزه)
  - تغییر مالکیت: حقیقی → حقوقی یا برعکس
- [ ] قدرت نسبی: بازده 5/10/20 روزه‌ی نماد منهای بازده‌ی شاخص هم‌وزن بازار (محاسبه از روی همه‌ی نمادها اگر داده‌ی شاخص نیست).
- [ ] مقدار نامعتبر (صفر در مخرج، تاریخچه‌ی ناکافی) = `null` و عدم مشارکت فاکتور، نه صفر.

**DoD:** برای یک روز نمونه، جدول ویژگی همه‌ی نمادها ساخته می‌شود و برای 2 نماد با محاسبه‌ی دستی از داده‌ی خام مطابقت دارد.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 4.4 — امتیازدهی و رتبه‌بندی

- [ ] interface `ScoringFactor { Name(); Compute(features, marketContext) -> (score 0..1 | null, evidence) }` و دو ترکیب `bullish` و `bearish` با وزن‌های config. وزن‌های اولیه (قابل تغییر):

| فاکتور | وزن | نشانه‌های رشد | نشانه‌های ریزش |
|---|---|---|---|
| روند | 0.20 | close بالای ema20 و ema50، شیب ema مثبت، adx بالا | عکس آن |
| مومنتوم | 0.15 | rsi بین 55 تا 70، macd hist مثبت و صعودی، roc مثبت | rsi زیر 45 و نزولی، macd hist منفی، roc منفی |
| جریان حقیقی/حقوقی | 0.30 | خالص ورود پول حقیقی، قدرت خریدار > 1 و صعودی، روزهای متوالی ورود | خالص خروج پول حقیقی، قدرت خریدار < 1 و نزولی، روزهای متوالی خروج |
| حجم | 0.15 | حجم نسبی بالا در روز مثبت | حجم نسبی بالا در روز منفی |
| ساختار قیمت | 0.10 | نزدیکی/شکست سقف 20 و 60 روزه | نزدیکی/شکست کف 20 و 60 روزه |
| قدرت نسبی | 0.10 | بازده نسبی مثبت نسبت به بازار | بازده نسبی منفی |

- [ ] جریمه‌ها (کسر امتیاز، در config): اشباع (فاصله‌ی زیاد از ema20 یا rsi بالاتر از 80 برای رشد/پایین‌تر از 20 برای ریزش)، حجم بسیار کم، نماد قفل در صف خرید/فروش (اگر داده‌اش هست).
- [ ] نرمال‌سازی cross-sectional: رتبه‌ی درصدی هر ویژگی بین نمادهای واجد شرایط همان روز.
- [ ] فیلتر شایستگی: حداقل تاریخچه (≥ 60 روز)، حداقل نقدشوندگی (میانگین ارزش معاملات)، معامله‌شدن در روز D، نوع نماد.
- [ ] انتخاب Top-3 رشد و Top-3 ریزش؛ یک نماد در هر دو فهرست نیاید؛ tie-break قطعی.
- [ ] خروجی هر انتخاب: امتیاز کل، تفکیک فاکتورها، 3 تا 5 دلیل خوانا به فارسی از روی شواهد (مثلاً «خالص ورود پول حقیقی ۴ روز متوالی»)، سطوح کلیدی (کف/سقف 20 روزه) و هشدار ریسک.

**DoD:** برای چند روز گذشته فهرست‌ها ساخته می‌شود؛ نماد غیرنقدشونده/متوقف/غیرسهام در فهرست‌ها نیست؛ اجرای دوباره همان خروجی را می‌دهد.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 4.5 — تحلیل دیده‌بان

- [ ] دیده‌بان را از منبع موجود (Recon) بخوان.
- [ ] برای هر نماد: برچسب وضعیت (صعودی قوی / صعودی / خنثی / نزولی / نزولی قوی) از همان امتیازها، تغییر روز، حجم نسبی، خلاصه‌ی جریان حقیقی/حقوقی، **تغییر وضعیت نسبت به گزارش قبلی**، و هشدارها (شکست حمایت، خروج متوالی پول حقیقی، حجم غیرعادی).
- [ ] نماد دیده‌بانی که واجد شرایط فیلتر نیست (توقف/بدون معامله) با وضعیت «بدون داده/متوقف» نمایش داده شود، نه اینکه حذف شود.

**DoD:** خروجی دیده‌بان برای 2 روز پیاپی ساخته می‌شود و ستون «تغییر وضعیت» درست پر می‌شود.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 4.6 — ساخت و تحویل گزارش

- [ ] interface `ReportWriter` و پیاده‌سازی Markdown (فارسی) + JSON ساختاریافته.
- [ ] ساختار گزارش: عنوان و تاریخ (میلادی + شمسی)، خلاصه‌ی بازار (بازده هم‌وزن، breadth صعودی/نزولی، جمع خالص پول حقیقی، پوشش داده)، جدول 3 نماد مستعد رشد + دلایل، جدول 3 نماد مستعد ریزش + دلایل، جدول وضعیت دیده‌بان و تغییرات، هشدارهای کیفیت داده، و سلب مسئولیت (تحلیلی است، توصیه‌ی سرمایه‌گذاری نیست).
- [ ] ذخیره در DB (`bourse_daily_reports`: date یکتا، payload JSON، markdown، created_at، نسخه‌ی کانفیگ) و (اختیاری) `bourse_daily_scores` برای امتیاز همه‌ی نمادها.
- [ ] نوشتن فایل در پوشه‌ی config‌شده و publish روی NATS (پیشنهاد: `reports.bourse.daily`).

**DoD:** گزارش کامل یک روز نمونه در DB، فایل و NATS دیده می‌شود و متن آن خوانا و درست است.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 4.7 — تایید نهایی فاز 4

- [ ] اجرا برای دست‌کم 5 روز معاملاتی گذشته با `--date`.
- [ ] برای 2 نماد مقادیر ویژگی‌ها (سرانه، قدرت خریدار، خالص جریان، ema/rsi) را دستی از داده‌ی خام بازمحاسبه و با گزارش مطابقت بده.
- [ ] اجرای دوباره = همان خروجی؛ روز غیرمعاملاتی = بدون گزارش؛ داده‌ی ناقص = گزارش با هشدار.
- [ ] مرور متن دلایل گزارش از نظر درستی و خوانایی فارسی.

**DoD:** همه‌ی موارد بالا مشاهده و ثبت شد.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 4.8 — (اختیاری) ردیابی عملکرد انتخاب‌ها

- [ ] برای انتخاب‌های روزهای قبل، بازده 1/3/5 روز بعد را حساب کن و بخش «عملکرد انتخاب‌های اخیر» را به گزارش اضافه کن (برای کالیبره‌کردن وزن‌ها).

**DoD:** بخش جدید برای چند گزارش قبلی درست پر می‌شود.

> یادداشت اجرا: _(پس از انجام پر شود)_

---

## فاز 5 — پایان‌کار

### گام 5.1 — مستندات

- [ ] مستندات پروژه: معماری، schema استراتژی و عملگرها، نحوه‌ی افزودن استراتژی/عملگر/اندیکاتور/فاکتور، قرارداد پیام سیگنال و گزارش، نحوه‌ی اجرای replay و ورکرها، نمونه‌کانفیگ‌ها.

**DoD:** یک نفر جدید با خواندن مستندات می‌تواند یک استراتژی YAML تازه اضافه و با replay بسنجد.

> یادداشت اجرا: _(پس از انجام پر شود)_

### گام 5.2 — مرور و تحویل نهایی

- [ ] build و lint کل پروژه؛ اجرای end-to-end هر دو ورکر؛ حذف کد موقت/debug؛ اطمینان از نبودن secret در commit.
- [ ] همه‌ی گام‌های این فایل را مرور و وضعیت چک‌باکس‌ها را با واقعیت تطبیق بده.
- [ ] فهرست مسائل باز و پیشنهادهای بعدی را در یادداشت اجرا بنویس.

**DoD:** هیچ گام تیک‌نخورده‌ی بی‌دلیل نمانده؛ «وضعیت کلی» بالای فایل به‌روز شد.

> یادداشت اجرا: _(پس از انجام پر شود)_

---

## ایده‌های بعدی (خارج از این feature)

- اتصال سیگنال‌ها به اعلان (پیام‌رسان/ایمیل) یا ماژول اجرای سفارش.
- بهینه‌ساز پارامتر با walk-forward و بک‌تست کامل‌تر.
- استراتژی‌های درون‌روزی بورس با `MarketProfile` واقعی (ساعات معاملات، دامنه‌ی نوسان).
- رتبه‌بندی یادگیری‌ماشینی روی ویژگی‌های فاز 4 با داده‌ی ردیابی عملکرد.
