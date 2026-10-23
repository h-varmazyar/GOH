# راهنمای راه‌اندازی Gatec

این سند مراحل آماده‌سازی و اجرای Gatec را در محیط توسعه یا سرور توضیح می‌دهد. تمام فرمان‌ها را از ریشه پروژه اجرا کنید؛ زیرا برنامه فایل `.env` و مسیرهای نسبی را بر اساس پوشه جاری پیدا می‌کند.

## پیش‌نیازها

- Go نسخه `1.25.3`
- PostgreSQL
- n8n برای اجرای workflowهای موردنیاز راه‌اندازی
- ابزارهای build زبان C/C++ و CGO
- TDLib به همراه OpenSSL و zlib؛ پیکربندی فعلی build، headerها را در `/usr/local/include` و کتابخانه‌ها را در `/usr/local/lib` جست‌وجو می‌کند
- دسترسی Telegram API شامل `api_id` و `api_hash`، در صورت استفاده از امکانات تلگرام

> فایل‌های secret، کلیدهای JWT، فایل `.env` و دیتابیس sessionهای Telegram نباید در Git ثبت شوند.

## ۱. دریافت و آماده‌سازی تنظیمات

از ریشه پروژه فایل تنظیمات محلی را ایجاد کنید:

```bash
cp .env.example .env
```

سپس مقادیر `.env` را متناسب با محیط تکمیل کنید. مهم‌ترین گروه‌های تنظیمات عبارت‌اند از:

- `SERVER_*`: آدرس و پورت HTTP
- `POSTGRES_*`, `DB_TYPE`, `IS_SSL_ENABLE`: اتصال دیتابیس
- `JWT_*`, `SIGNING_METHOD`: کلیدهای امضا و اعتبارسنجی JWT
- `SERVICE_AUTH_*`: تنظیمات احراز هویت
- `SERVICE_TELEGRAM_*`: اعتبارنامه و محل نگهداری sessionهای Telegram
- `ADAPTER_*`: اتصال سرویس‌های Nobitex و Tabdeal
- `WORKER_*`: فعال‌سازی و تنظیم workerها

مقادیر duration باید با قالب Go باشند؛ برای مثال `250ms`، `30s` یا `1m`.

برای PostgreSQL مقدار `DB_TYPE` را دقیقاً برابر مقدار زیر قرار دهید:

```dotenv
DB_TYPE=postgreSQL
```

پوشه مشخص‌شده در `SERVICE_TELEGRAM_DATABASE_DIR` باید موجود، قابل‌نوشتن و روی سرور پایدار باشد. برای نمونه:

```dotenv
SERVICE_TELEGRAM_DATABASE_DIR=.tdlib2
```
## ۲. ساخت کلیدهای JWT

یک جفت کلید سازگار با `SIGNING_METHOD` بسازید و فایل‌های آن را خارج از Git نگه دارید. مسیر فایل‌ها را در `.env` تنظیم کنید:

```dotenv
JWT_PUBLIC_KEY_PATH=configs/secrets/jwt-public.secret
JWT_PRIVATE_KEY_PATH=configs/secrets/jwt-private.secret
SIGNING_METHOD=RS256
```

برنامه محتوای دو فایل کلید را هنگام شروع می‌خواند؛ بنابراین مسیر اشتباه یا فایل خوانده‌نشدنی باعث توقف برنامه می‌شود. سایر secretها، از جمله hashing salt و توکن Tabdeal، از متغیرهای محیطی مربوط به خود خوانده می‌شوند و صرف قرار دادن آن‌ها در `configs/secrets` کافی نیست.

## ۳. آماده‌سازی PostgreSQL

یک database و user بسازید و اطلاعات آن را در `.env` قرار دهید. اگر database وجود نداشته باشد، برنامه تلاش می‌کند آن را ایجاد کند؛ در این حالت user باید مجوز `CREATE DATABASE` داشته باشد. در غیر این صورت database را پیش از اجرای برنامه دستی بسازید.

در هر بار شروع، migrationهای GORM به‌صورت خودکار اجرا می‌شوند و دستور migration جداگانه‌ای لازم نیست. برای محیط production پیش از اولین اجرا از دیتابیس نسخه پشتیبان بگیرید.

## ۴. آماده‌سازی workflowهای n8n

فایل‌های workflow بخشی از ملزومات راه‌اندازی هستند، اما برنامه Go آن‌ها را خودکار در n8n ثبت نمی‌کند. کاربر در اولین راه‌اندازی باید برای هر فایل مراحل زیر را انجام دهد:

1. n8n را اجرا و پنل آن را باز کند.
2. از بخش **Workflows** گزینه **Import from File** را انتخاب کند.
3. فایل‌های JSON را به ترتیب شماره import کند.
4. credentialها، URLها، tokenها و webhookهای استفاده‌شده در nodeها را متناسب با محیط تنظیم کند.
5. هر workflow را یک بار به‌صورت دستی آزمایش کند.
6. workflowهایی را که باید خودکار اجرا شوند روی حالت **Active** قرار دهد.

اگر n8n داخل Docker اجرا می‌شود، می‌توان پوشه را read-only داخل container mount کرد؛ برای نمونه:

```yaml
volumes:
  - ./deployments/n8n/workflows:/bootstrap/workflows:ro
```

سپس فایل‌های `/bootstrap/workflows/*.json` را از پنل import کنید یا در اسکریپت bootstrap اختصاصی n8n به کار ببرید. فایل export نباید credential، token یا secret واقعی داشته باشد؛ credentialها را در خود n8n تعریف کنید و nodeها را به آن‌ها متصل کنید.

> هر زمان workflow جدیدی به راه‌اندازی اولیه اضافه شد، فایل JSON آن را در همین مسیر قرار دهید، شماره و نام معنادار انتخاب کنید و ترتیب اجرای آن را در این بخش ثبت کنید.

## ۵. تنظیم workerها برای اولین اجرا

در `.env.example` چند worker به‌صورت پیش‌فرض فعال‌اند. در اولین اجرا بهتر است workerهای وابسته به provider، credential یا سرویس خارجی را غیرفعال کنید و پس از تکمیل تنظیمات، آن‌ها را مرحله‌ای فعال کنید:

```dotenv
WORKER_CANDLE_POLLING_ENABLED=false
WORKER_MARKET_SYNC_ENABLED=false
WORKER_PROXY_HEALTH_CHECKER_ENABLED=false
WORKER_CONNECTION_MONITOR_ENABLED=false
WORKER_INDICATOR_CALCULATOR_ENABLED=false
WORKER_STOCK_MARKET_SYNC_ENABLED=false
WORKER_TSE_MARKET_HISTORY_SYNC_ENABLED=false
WORKER_TSE_MARKET_DAILY_SYNC_ENABLED=false
```

برای فعال‌سازی TSE Daily باید `WORKER_TSE_MARKET_DAILY_SYNC_TELEGRAM_BOT_TOKEN` نیز مقدار داشته باشد. حالت `WORKER_STOCK_MARKET_SYNC_MODE` فقط می‌تواند `daily` یا `historical` باشد.
## ۶. build و اجرای برنامه

وابستگی‌های Go داخل repository به‌صورت vendor موجودند. ابتدا build را بررسی کنید:

```bash
go build -mod=vendor -o ./bin/gatec ./cmd
```

سپس برنامه را از ریشه پروژه اجرا کنید:

```bash
./bin/gatec
```

برای اجرای مستقیم در محیط توسعه نیز می‌توانید از فرمان زیر استفاده کنید:

```bash
go run -mod=vendor ./cmd
```

برنامه به آدرس `${SERVER_ADDR}:${SERVER_PORT}` گوش می‌دهد. مستندات Swagger پس از اجرا در مسیر زیر در دسترس است:

```text
http://<SERVER_ADDR>:<SERVER_PORT>/docs/api/index.html
```

برای مثال، اگر پورت `8575` است:

```text
http://localhost:8575/docs/api/index.html
```

## ۷. اقدامات اولین اجرا

ترتیب پیشنهادی برای راه‌اندازی اولیه:

1. PostgreSQL و n8n را اجرا کنید.
2. workflowهای `deployments/n8n/workflows/` را در n8n import، آزمایش و فعال کنید.
3. فایل `.env`، کلیدهای JWT و credentialهای سرویس‌ها را آماده کنید.
4. برنامه Gatec را اجرا و موفق‌بودن اتصال و migration دیتابیس را در logها بررسی کنید.
5. از endpoint زیر کاربر اولیه را ثبت کنید:

   ```text
   POST /v1/auth/register
   ```

6. با endpoint زیر وارد شوید و JWT دریافت کنید:

   ```text
   POST /v1/auth/login
   ```

7. برای endpointهای محافظت‌شده header زیر را ارسال کنید:

   ```http
   Authorization: Bearer <token>
   ```

8. credentialها و providerهای موردنیاز را تکمیل و workerها را یکی‌یکی فعال کنید.
9. در صورت استفاده از Telegram، session را از endpointهای `/v1/telegram/sessions` ایجاد و مراحل phone، code و در صورت نیاز password را تکمیل کنید.

## بررسی خطاهای رایج

- **فایل `.env` پیدا نمی‌شود:** برنامه را از ریشه repository اجرا کنید.
- **خطای خواندن کلید JWT:** وجود فایل، permission و مقدار `JWT_*_KEY_PATH` را بررسی کنید.
- **خطای اتصال یا ساخت database:** اطلاعات PostgreSQL و مجوزهای user را بررسی کنید.
- **خطای TDLib هنگام build:** نصب TDLib، compiler، CGO، OpenSSL و zlib و مسیرهای `/usr/local/include` و `/usr/local/lib` را بررسی کنید.
- **worker هنگام شروع خطا می‌دهد:** worker را موقتاً غیرفعال و credential/provider موردنیازش را تکمیل کنید.
- **workflow اجرا نمی‌شود:** فعال‌بودن workflow، credentialهای n8n، URL برنامه و تنظیم webhook را بررسی کنید.

> در نسخه فعلی route سلامت HTTP فعال نیست؛ برای بررسی اولیه از logهای برنامه، Swagger و یک درخواست register/login استفاده کنید.