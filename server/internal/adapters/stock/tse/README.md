# TSE adapter

این package برای دریافت داده از TSETMC استفاده می‌شود. تمام متدها `context.Context` می‌گیرند و درخواست HTTP را با همان context لغو می‌کنند.

## ساخت Adapter

```go
adapter, err := tse.NewAdapter(&configs.TSE{}, nil, providerID)
```

client پیش‌فرض timeout سی‌ثانیه دارد. پارامتر `candleChan` برای سازگاری با factory فعلی نگه داشته شده است.

## فهرست نمادها

`UpdateInstruments` ابتدا endpoint صفحه‌بندی‌شده‌ی زیر را صدا می‌زند:

`GET https://cdn.tsetmc.com/api/Instrument/GetInstrumentSearch?pageSize=100&pageNumber=N`

خروجی به `[]Instrument` تبدیل می‌شود. اگر سرویس در دسترس نباشد، فایل‌های JSON موجود در `assets/tse/instruments` به‌عنوان cache خوانده می‌شوند.

## Market Watch

`GetMarketWatch(ctx, heven, ref)` به endpoint قدیمی زیر درخواست می‌زند:

`GET https://old.tsetmc.com/tsev2/data/MarketWatchInit.aspx?h=<heven>&r=<ref>`

خروجی متنی با جداکننده‌های `@`، `;` و `,` parse می‌شود و رکوردها به‌همراه reference بعدی برگردانده می‌شوند.

## داده‌های یک نماد

متدهای JSON از `https://cdn.tsetmc.com/api` استفاده می‌کنند:

| متد | Endpoint |
| --- | --- |
| `FetchInstrumentInfo` | `/Instrument/GetInstrumentInfo/{insCode}` |
| `FetchInstrumentStateHistory` | `/MarketData/GetInstrumentStateAll/{insCode}` |
| `FetchClosingPriceHistory` | `/ClosingPrice/GetClosingPriceDailyList/{insCode}/0` |
| `FetchClientTypeHistory` | `/ClientType/GetClientTypeHistory/{insCode}` |
| `FetchCurrentClosingPrice` | `/ClosingPrice/GetClosingPriceInfo/{insCode}` |
| `FetchCurrentClientType` | `/ClientType/GetClientType/{insCode}/1/0` |

تاریخ‌های TSETMC با قالب `YYYYMMDD` به `time.Time` در UTC تبدیل می‌شوند. خطای status غیر ۲۰۰ و خطای decode به caller برگردانده می‌شود.