# deployment

پوشه‌ی استقرار (deployment) پروژه‌ی gatec. این پوشه کنار دو پوشه‌ی برنامه در همان مخزن GOH قرار دارد:

```
GOH/
├── server/       بک‌اند (Go): API، workerها، airdrop-catcher، signal-replay
├── frontend/     پنل (React + Vite)
└── deployment/   همین پوشه: docker-compose، nginx و اسکریپت‌های استقرار
```

ساختار flat است و همه‌چیز در ریشه‌ی deployment قرار دارد:

```
deployment/
├── deploy.conf               تنها فایل وابسته به پروژه برای Makefile و scripts/ (پایین را ببینید)
├── docker-compose.yml
├── .env                      تنظیمات (در git نیست؛ از .env.example کپی کنید)
├── .env.example
├── nginx/gatec.conf.template
├── server/secrets/           کلیدهای JWT و فایل‌های config بک‌اند (mount به /app/configs/secrets)
├── server/logs/              لاگ همه‌ی سرویس‌های بک‌اند (در git نیست)
├── server/tdlib/             سشن‌های تلگرام (TDLib)، مشترک بین server و airdrop-catcher و چند worker (در git نیست)
└── scripts/                  اسکریپت‌ها (lib.sh توابع مشترک آن‌هاست)
```

همه‌ی دستورها مقادیر را از `deployment/.env` می‌خوانند (نه از `.env` پوشه‌های server یا frontend).
`.env` هرگز `source` نمی‌شود، پس مقدارهای دارای فاصله بدون کوتیشن هم مشکلی ندارند.
همین `.env` از طریق `env_file` به همه‌ی کانتینرهای بک‌اند داده می‌شود؛ همه‌ی کلیدهای `configs.Load`
باید در آن باشند (حتی خالی)، وگرنه برنامه panic می‌کند.

`Makefile` و `scripts/` همان نسخه‌ی gozarsaz هستند، با یک تفاوت: در حالت `local` مخزن برنامه
می‌تواند یک clone جدا باشد (مثل Gozar) یا یک پوشه داخل clone مشترک (مثل اینجا که server و frontend
و deployment در یک مخزن‌اند). در حالت دوم clone مشترک در هر deploy فقط یک بار pull می‌شود.

---

## `deploy.conf` و استفاده در پروژه‌های دیگر

`Makefile` و `scripts/` هیچ نامی از این پروژه ندارند؛ هر چیز وابسته به پروژه در `deploy.conf` است.

| کلید | کاربرد |
|---|---|
| `PROJECT` | نام پروژه (`gatec`)؛ template در `nginx/gatec.conf.template` و مقصد نصب `/etc/nginx/conf.d/gatec.conf` |
| `IMAGES` | ایمیج‌های `make build-image`: نام، مخزن (پوشه‌ی کناری)، مسیر Dockerfile و build arg ها |
| `TARGETS` | مقادیر `APP` در `make deploy`: ایمیج‌هایی که در حالت `local` ساخته می‌شوند، سرویس‌های compose و پوشه‌های الزامی |
| `TOOLS` | ابزارهای یک‌باره‌ی `make tool`: ایمیج و پوشه‌های الزامی |
| `project_checks` | تابع اختیاری برای چک‌های اختصاصی `make check` (با `check_http` و `check_tcp`) |

build arg ها در `IMAGES` به سه شکل نوشته می‌شوند: `KEY=VAL` (مقدار ثابت)، `KEY` (از `.env`، الزامی)،
`KEY?` (از `.env`؛ اگر کلید نباشد پیش‌فرض Dockerfile می‌ماند).

نام متغیرها از روی نام‌ها ساخته می‌شود (حروف بزرگ و `-` به `_`):

| الگو | مثال | کاربرد |
|---|---|---|
| `<IMAGE>_IMAGE` | `SERVER_IMAGE` | تگ ایمیج در `.env` و `docker-compose.yml` |
| `<IMAGE>_SHA` | `FRONTEND_SHA` | SHA کامیت ساخته‌شده (حالت `local`) |
| `<REPO>_DIR` | `SERVER_DIR` | مسیر دیگر برای پوشه‌ی برنامه (پیش‌فرض `../<repo>`) |
| `<REPO>_BRANCH` | `SERVER_BRANCH` | شاخه‌ی pull در حالت `local` (پیش‌فرض `main`) |

### ایمیج‌ها

| ایمیج | Dockerfile | توضیح |
|---|---|---|
| `tdlib` | `server/deploy/tdlib.Dockerfile` | ایمیج پایه‌ی TDLib. کامپایلش طولانی است ولی تا Dockerfile عوض نشود از cache می‌آید |
| `server` | `server/deploy/Dockerfile` | همه‌ی باینری‌ها: `gatec` (API)، `gatec-worker`، `gatec-airdrop-catcher`، `gatec-signal-replay` |
| `frontend` | `frontend/Dockerfile` | build استاتیک Vite که nginx داخل کانتینر روی پورت 3000 سرو می‌کند |

- متغیر ایمیج `tdlib` در `.env` اسمش `TDLIB_IMAGE` است، یعنی همان build arg ـی که `server/deploy/Dockerfile`
  می‌خواهد. پس در `make deploy` (حالت `local`) اول `tdlib:local` ساخته و در `.env` نوشته می‌شود و
  سپس build ـِ `server` همان را برمی‌دارد.
- workerها ایمیج جدا ندارند: همه‌ی سرویس‌های بک‌اند از `SERVER_IMAGE` استفاده می‌کنند و باینری با
  `entrypoint` در `docker-compose.yml` انتخاب می‌شود (`gatec-worker <name>` برای workerها).

مسیر `deploy.conf` را با متغیر `CONFIG_FILE` هم می‌توان عوض کرد.

---

## Makefile

همه‌ی دستورها از ریشه‌ی deployment اجرا می‌شوند. لیست دستورها:

```bash
make help
```

### متغیرهای مشترک

| متغیر | پیش‌فرض | کاربرد |
|---|---|---|
| `APP` | `all` | برنامه‌ی هدف (مقادیر مجاز در هر دستور فرق دارد) |
| `VERSION` | — | تگ ایمیج، مثل `1.4.0` (فقط `build-image`) |
| `CMD` | — | دستور کانتینر (فقط `tool`) |
| `ARGS` | — | فلگ‌های اضافه که مستقیم به اسکریپت داده می‌شود |
| `SUDO` | `sudo` | برای دستورهای nginx؛ اگر root هستید `SUDO=` بگذارید |

---

### `make build-image`

ایمیج داکر یک برنامه را از پوشه‌ی کناری آن می‌سازد. اسکریپت: `scripts/build-image.sh`

| پارامتر | الزامی | توضیح |
|---|---|---|
| `APP` | بله | یکی از `IMAGES` در `deploy.conf`: `tdlib` ، `server` یا `frontend` |
| `VERSION` | بله | تگ ایمیج؛ خروجی `<APP>:<VERSION>` |

فلگ‌های قابل ارسال با `ARGS`:

| فلگ | توضیح |
|---|---|
| `--latest` | علاوه بر نسخه، تگ `<APP>:latest` هم می‌زند |
| `--build-arg KEY=VAL` | build arg اضافه (قابل تکرار)؛ بر مقدار `.env` اولویت دارد |
| `--network MODE` | شبکه‌ی مراحل `RUN`، مثلاً `host` (یا متغیر `DOCKER_BUILD_NETWORK`) |

build arg هایی که از `.env` خوانده می‌شوند:
- `server`: `TDLIB_IMAGE` — **الزامی**؛ ایمیج TDLib که باید از قبل روی سرور باشد.
  `GO_IMAGE` و `RUNTIME_IMAGE` اختیاری‌اند (وقتی registry پیش‌فرض در دسترس نیست).
- `tdlib`: `GO_IMAGE` و `TDLIB_REF` اختیاری.
- `frontend`: `NEXT_PUBLIC_API_BASE_URL` — **الزامی**؛ آدرس بک‌اند که مرورگر صدا می‌زند
  (در Dockerfile به `VITE_API_BASE_URL` تبدیل و داخل bundle قرار می‌گیرد)، مثلاً `https://gatec.ir/api`.

```bash
make build-image APP=tdlib VERSION=1 ARGS="--latest"    # یک بار؛ TDLIB_IMAGE=tdlib:latest
make build-image APP=server VERSION=1.4.0
make build-image APP=frontend VERSION=1.4.0

# شبکه‌ی host برای npm ci / go build پشت پروکسی
make build-image APP=frontend VERSION=1.4.0 ARGS="--network host"

# override یک مقدار .env فقط برای همین build
make build-image APP=frontend VERSION=1.4.0 ARGS="--build-arg NEXT_PUBLIC_API_BASE_URL=https://example.com/api"
```

---

### `make deploy`

برنامه‌ها را با docker compose بالا می‌آورد و سپس `gatec.conf` را روی nginx میزبان نصب می‌کند
(همان کار `make install-nginx`). اسکریپت: `scripts/deploy.sh`

| `APP` | سرویس‌های compose | ایمیجی که در حالت `local` ساخته می‌شود |
|---|---|---|
| `server` | `server` ، `airdrop-catcher` | `tdlib:local` ، `server:local` |
| `airdrop` | `airdrop-catcher` | `tdlib:local` ، `server:local` |
| `workers` | `candle-polling` ، `market-sync` ، `tse-market-data-sync` ، `tse-trade-history` ، `indicator-watch` ، `proxy-health-checker` ، `connection-monitor` ، `stock-market-sync` ، `stock-instrument-sync` | `tdlib:local` ، `server:local` |
| `frontend` | `frontend` | `frontend:local` |
| `infra` | `nats` ، `postgres` ، `clickhouse` ، `ch-ui` | — |
| `nginx` | — (فقط nginx) | — |
| `all` | همه | همه |

- وابستگی‌ها (`depends_on`) خودکار بالا می‌آیند؛ مثلاً `APP=server` اگر postgres خاموش باشد آن را هم روشن می‌کند.
- `server` ، `airdrop` و `workers` یک ایمیج مشترک دارند: deploy یکی از آن‌ها ایمیج را دوباره می‌سازد،
  ولی بقیه تا deploy بعدی خودشان روی build قبلی می‌مانند. برای به‌روزرسانی همه‌ی بک‌اند `APP=all` بزنید.

فلگ‌های قابل ارسال با `ARGS`:

| فلگ | توضیح |
|---|---|
| `--mode auto\|local\|ci\|images` | `auto` (پیش‌فرض): در CI حالت `ci`، در غیر این صورت `local` |
| `--branch NAME` | شاخه‌ای که در حالت `local` pull می‌شود (پیش‌فرض `<REPO>_BRANCH` مثل `SERVER_BRANCH` ، یا `main`) |
| `--image NAME=REF` | `<NAME>_IMAGE` را در `.env` می‌نویسد (حالت `ci`؛ قابل تکرار)، مثلاً `--image server=REF` |
| `--<name>-image REF` | همان `--image <name>=REF`، مثلاً `--server-image REF` |
| `--skip-nginx` | مرحله‌ی nginx را اجرا نکن |

حالت‌ها:
- **`local`**: مخزن GOH روی همین سرور `git pull` می‌شود (یک بار برای server و frontend، چون یک clone
  مشترک‌اند) و ایمیج‌ها با `build-image.sh` ساخته می‌شوند. چون خود deployment هم در همین مخزن است،
  pull اسکریپت‌ها و compose را هم به‌روز می‌کند.
- **`ci`**: ایمیج‌ها از registry کشیده می‌شوند (`compose pull`).
- **`images`**: نه build نه pull؛ همان ایمیج‌هایی که در `.env` آمده و روی سرور موجودند اجرا می‌شوند.

در پایان، `DEPLOYED_AT` و (در حالت `local`) تگ و SHA ایمیج‌ها (`<IMAGE>_IMAGE` و `<IMAGE>_SHA`) در `.env` نوشته می‌شود.

```bash
make deploy                                   # همه، حالت local، با nginx
make deploy APP=server
make deploy APP=workers
make deploy APP=frontend ARGS="--branch develop"
make deploy APP=infra ARGS="--skip-nginx"
make deploy APP=nginx                         # فقط nginx
make deploy APP=all ARGS="--mode images"      # ایمیج‌هایی که از قبل روی سرور هستند

# CI / registry
make deploy APP=server ARGS="--mode ci --server-image registry.example.com/gatec-server:1.4.0"
```

---

### `make tool`

یک ابزار یک‌باره را با `compose run --rm` اجرا می‌کند؛ کانتینر کارش را انجام می‌دهد و تمام می‌شود.
اسکریپت: `scripts/run-tool.sh`

| پارامتر | الزامی | توضیح |
|---|---|---|
| `APP` | بله | یکی از `TOOLS` در `deploy.conf`: `worker` یا `signal-replay` |
| `CMD` | خیر | دستور و فلگ‌های کانتینر |

| فلگ (`ARGS`) | توضیح |
|---|---|
| `--no-build` | build نمی‌کند؛ همان ایمیجی که در `.env` (`SERVER_IMAGE`) آمده اجرا می‌شود |

- `worker`: هر worker را یک بار با `gatec-worker` اجرا می‌کند (مثلاً backfill یا sync دستی).
- `signal-replay`: backtest یک استراتژی روی داده‌ی ذخیره‌شده (`server/cmd/signal-replay`)؛
  فایل‌های YAML از `server/strategies` به `/app/strategies` mount می‌شوند.
- به‌طور پیش‌فرض ایمیج `server` از checkout فعلی (بدون git pull) با تگ `server:local` ساخته می‌شود؛
  این کار به `TDLIB_IMAGE` موجود در `.env` نیاز دارد.
- سرویس ابزارها در profile `tools` است، پس `make deploy` و `docker compose up` آن‌ها را بالا نمی‌آورند.

```bash
make tool APP=worker CMD="tse-market-history-sync"
make tool APP=worker CMD="tse-market-data-sync --collection-method ins_codes --ins-codes 123,456" ARGS="--no-build"
make tool APP=signal-replay CMD="-strategy my-strategy -from 2026-01-01 -to 2026-06-30 -symbols BTCUSDT"
```

---

### `make render-nginx`

`nginx/gatec.conf.template` را با مقادیر `.env` پر می‌کند و **فقط چاپ** می‌کند. اسکریپت: `scripts/render-nginx.sh`

هر متغیر به شکل `${VAR}` (با آکولاد) جایگزین می‌شود و باید در `.env` باشد، وگرنه خطا می‌دهد.
متغیرهای خود nginx بدون آکولاد نوشته می‌شوند (`$host`) و دست نمی‌خورند. متغیرهای فعلی:

| کلید | کاربرد در nginx |
|---|---|
| `NGINX_SERVER_NAME` | دامنه (`server_name`)؛ پورت 80 به 443 redirect می‌شود |
| `NGINX_SSL_CERTIFICATE` / `NGINX_SSL_CERTIFICATE_KEY` | مسیر گواهی و کلید SSL روی میزبان |
| `SERVER_BIND_ADDRESS` / `SERVER_HOST_PORT` | upstream مسیر `/api/` (پیشوند `/api` حذف می‌شود) |
| `FRONTEND_BIND_ADDRESS` / `FRONTEND_HOST_PORT` | upstream مسیر `/` |

کلیدهای upstream در `docker-compose.yml` هم برای publish پورت‌ها استفاده می‌شوند، پس nginx و داکر همیشه یکسان‌اند.
ch-ui از nginx منتشر نمی‌شود؛ با SSH tunnel به `127.0.0.1:9052` وصل شوید.

```bash
make render-nginx
make render-nginx > /tmp/gatec.conf   # ذخیره برای بررسی
```

---

### `make install-nginx`

مثل `render-nginx`، ولی خروجی را نصب می‌کند، `nginx -t` می‌گیرد و nginx را reload می‌کند. اگر `nginx -t` خطا بدهد
(مثلاً فایل گواهی SSL وجود ندارد)، فایل قبلی برگردانده می‌شود. با `sudo` اجرا می‌شود.

| فلگ (`ARGS`) | توضیح |
|---|---|
| `--dest PATH` | مسیر نصب (پیش‌فرض `/etc/nginx/conf.d/gatec.conf` یا متغیر `NGINX_DEST`) |

```bash
make install-nginx
make install-nginx SUDO=          # وقتی خودتان root هستید
```

---

### `make check`

بررسی سلامت. اسکریپت: `scripts/check.sh`

- همه‌ی سرویس‌های `docker-compose.yml` (به‌جز ابزارها) در حال اجرا باشند.
- nginx میزبان فعال باشد، `nginx -t` بگذرد و فایل نصب‌شده با خروجی فعلی `render-nginx` یکی باشد.
- چک‌های `project_checks` در `deploy.conf`: API مستقیم و از طریق nginx، frontend مستقیم و از طریق nginx،
  مانیتور NATS، ping کلیک‌هاوس، ch-ui و پورت postgres.

```bash
make check
make check ARGS="--verbose"
```

---

## پورت‌ها

| سرویس | آدرس روی میزبان | کلید `.env` |
|---|---|---|
| nats / مانیتور | `127.0.0.1:9022` / `127.0.0.1:9023` | `NATS_HOST_PORT` / `NATS_MONITOR_HOST_PORT` |
| postgres | `127.0.0.1:9032` | `POSTGRES_HOST_PORT` |
| clickhouse native / http | `127.0.0.1:9042` / `127.0.0.1:9043` | `CLICKHOUSE_NATIVE_HOST_PORT` / `CLICKHOUSE_HTTP_HOST_PORT` |
| ch-ui | `127.0.0.1:9052` | `CH_UI_BIND_ADDRESS` / `CH_UI_HOST_PORT` |
| server (API) | `127.0.0.1:8575` | `SERVER_BIND_ADDRESS` / `SERVER_HOST_PORT` |
| frontend | `127.0.0.1:8091` | `FRONTEND_BIND_ADDRESS` / `FRONTEND_HOST_PORT` |

داخل شبکه‌ی داکر، کانتینرها infra را با نام سرویس صدا می‌زنند (`postgres` ، `nats://nats:4222` ، `http://clickhouse:8123`).

---

## سناریوهای رایج

### راه‌اندازی اولیه روی سرور

```bash
git clone <GOH-url> /opt/gatec && cd /opt/gatec/deployment
sudo scripts/bootstrap.sh            # یک بار: Docker + nginx
cp .env.example .env                 # پسوردها، دامنه و ... را تنظیم کنید
# کلیدهای JWT را در server/secrets/ بگذارید (مسیرها در JWT_*_KEY_PATH)
# گواهی SSL را در مسیر NGINX_SSL_CERTIFICATE / NGINX_SSL_CERTIFICATE_KEY بگذارید
make render-nginx                    # بررسی خروجی nginx
make deploy                          # pull + build (tdlib, server, frontend) + compose up + nginx
make check
```

### انتشار نسخه‌ی مشخص

```bash
make build-image APP=server   VERSION=1.4.0
make build-image APP=frontend VERSION=1.4.0

# تگ‌ها را در .env بگذارید
#   SERVER_IMAGE=server:1.4.0
#   FRONTEND_IMAGE=frontend:1.4.0
make deploy APP=all ARGS="--mode images"
make check
```

### فقط تغییر پورت یا آدرس یک سرویس

```bash
# .env را ویرایش کنید، مثلاً SERVER_HOST_PORT=8576
make deploy APP=server   # داکر با پورت جدید و سپس nginx با همان پورت
```
