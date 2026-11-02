# TSE adapter

این package برای دریافت داده از TSETMC استفاده می‌شود. تمام متدها `context.Context` می‌گیرند و درخواست HTTP را با همان context لغو می‌کنند.

## ساخت Adapter

```go
adapter, err := tse.NewAdapter(&configs.TSE{}, nil, providerID)
```

client پیش‌فرض timeout سی‌ثانیه دارد. پارامتر `candleChan` برای سازگاری با factory فعلی نگه داشته شده است.

## فهرست نمادها

`UpdateInstruments` سهام بورس تهران، صندوق‌های ETF بورس و سهام بازار اول و دوم فرابورس را برمی‌گرداند. نمادهای اختیار معامله، حق تقدم، اوراق، بازار پایه و نمادهای لغو پذیرش‌شده حذف می‌شوند.

| منبع | کاربرد |
| --- | --- |
| `https://webgw.tse.ir/cmssite/api/v1/EDataListingCashMarket/List/fa` | سهام بورس (فقط `IRO1…`)، `instrumentId`، تاریخ درج و تاریخ عرضه |
| `https://webgw.tse.ir/InstrumentProvider/api/v1/ArchiveMarket/ETFMarket` | صندوق‌های ETF (`IRT1…`)، تاریخ شروع پذیره‌نویسی به‌عنوان تاریخ عرضه |
| `https://www.ifb.ir/markets.aspx?id=1` و `id=2` | فهرست نمادهای بازار اول و دوم فرابورس و وضعیت (مجاز / ممنوع-متوقف) |
| `https://www.ifb.ir/omidnameh.aspx` | تاریخ درج، عرضه و پذیرش نمادهای فرابورس (شمسی، به میلادی تبدیل می‌شود) |
| فایل‌های `assets/tse/instruments/*.json` | `insCode` و کد گروه و زیرگروه صنعت |
| `cdn.tsetmc.com/api/StaticData/GetStaticData` | نام گروه صنعت |

تطبیق: هر نماد با نماد نرمال‌شده (ی/ک عربی، نیم‌فاصله، فاصله، ارقام فارسی)، نوع ابزار TSETMC (`300` سهام بورس، `303` سهام فرابورس، `305` صندوق) و فعال‌بودن با فایل‌های JSON تطبیق داده می‌شود و نامزدها بر اساس وضعیت معاملاتی و شباهت نام مرتب می‌شوند. اگر بیش از یک نامزد باشد، `instrumentID` برگشتی از `GetInstrumentInfo` با `instrumentId` منبع مقایسه می‌شود. صفحات فرابورس `instrumentId` ندارند؛ برای آن‌ها مقدار از `GetInstrumentInfo` گرفته می‌شود. نمادی که در فایل‌ها نباشد با `GetInstrumentSearch` جست‌وجو می‌شود. عدم تطابق نام و نماد در لاگ هشدار داده می‌شود.

`MarketType` یکی از `BOURSE`، `ETF`، `IFB_FIRST` و `IFB_SECOND` است. تاریخ‌ها (`InsertDate`، `ReleaseDate`، `AcceptanceDate`) می‌توانند خالی باشند.

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