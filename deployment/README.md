# gozarsaz

مخزن استقرار (deployment) پروژه‌ی گذر. این مخزن کنار دو مخزن برنامه قرار می‌گیرد:

```
Gozar/
├── gozargah/   بک‌اند (Go)
├── gozarban/   پنل ادمین (Next.js)
└── gozarsaz/   همین مخزن: docker-compose، nginx و اسکریپت‌های استقرار
```

ساختار مخزن flat است و همه‌چیز در ریشه‌ی gozarsaz قرار دارد:

```
gozarsaz/
├── deploy.conf               تنها فایل وابسته به پروژه برای Makefile و scripts/ (پایین را ببینید)
├── docker-compose.yml
├── .env                      تنظیمات (در git نیست؛ از .env.example کپی کنید)
├── .env.example
├── nginx/gozar.conf.template
├── gozargah/secrets/         کلیدها و فایل‌های config بک‌اند (mount به /app/configs)
├── gozargah/uploads/         فایل‌های آپلودشده
└── scripts/                  اسکریپت‌ها (lib.sh توابع مشترک آن‌هاست)
```

همه‌ی دستورها مقادیر را از `gozarsaz/.env` می‌خوانند (نه از `.env` مخزن‌های برنامه).
`.env` هرگز `source` نمی‌شود، پس مقدارهای دارای فاصله (مثل `DAY_OTP_MESSAGE_SUBJECT`) بدون کوتیشن هم مشکلی ندارند.

---

## `deploy.conf` و استفاده در پروژه‌های دیگر

`Makefile` و `scripts/` هیچ نامی از این پروژه ندارند؛ هر چیز وابسته به پروژه در `deploy.conf` است.
برای پروژه‌ی دیگری با همین الگو (مخزن‌های برنامه کنار مخزن استقرار):

1. `Makefile` و پوشه‌ی `scripts/` را کپی کنید.
2. یک `deploy.conf` بنویسید (این فایل را الگو قرار دهید).
3. `docker-compose.yml` ، `.env` و در صورت نیاز `nginx/<PROJECT>.conf.template` خود پروژه را بگذارید.

| کلید | کاربرد |
|---|---|
| `PROJECT` | نام پروژه؛ template در `nginx/$PROJECT.conf.template` و مقصد نصب `/etc/nginx/conf.d/$PROJECT.conf` |
| `IMAGES` | ایمیج‌های `make build-image`: نام، مخزن (پوشه‌ی کناری)، مسیر Dockerfile و build arg ها |
| `TARGETS` | مقادیر `APP` در `make deploy`: ایمیج‌هایی که در حالت `local` ساخته می‌شوند، سرویس‌های compose و پوشه‌های الزامی |
| `project_checks` | تابع اختیاری برای چک‌های اختصاصی `make check` (با `check_http` و `check_tcp`) |

build arg ها در `IMAGES` به سه شکل نوشته می‌شوند: `KEY=VAL` (مقدار ثابت)، `KEY` (از `.env`، الزامی)،
`KEY?` (از `.env`؛ اگر کلید نباشد پیش‌فرض Dockerfile می‌ماند).

نام متغیرها از روی نام‌ها ساخته می‌شود (حروف بزرگ و `-` به `_`):

| الگو | مثال | کاربرد |
|---|---|---|
| `<IMAGE>_IMAGE` | `GOZARGAH_WORKER_IMAGE` | تگ ایمیج در `.env` و `docker-compose.yml` |
| `<IMAGE>_SHA` | `GOZARBAN_SHA` | SHA کامیت ساخته‌شده (حالت `local`) |
| `<REPO>_DIR` | `GOZARGAH_DIR` | مسیر دیگر برای مخزن برنامه (پیش‌فرض `../<repo>`) |
| `<REPO>_BRANCH` | `GOZARBAN_BRANCH` | شاخه‌ی pull در حالت `local` (پیش‌فرض `main`) |

مسیر `deploy.conf` را با متغیر `CONFIG_FILE` هم می‌توان عوض کرد.

---

## Makefile

همه‌ی دستورها از ریشه‌ی gozarsaz اجرا می‌شوند. لیست دستورها:

```bash
make help
```

### متغیرهای مشترک

| متغیر | پیش‌فرض | کاربرد |
|---|---|---|
| `APP` | `all` | برنامه‌ی هدف (مقادیر مجاز در هر دستور فرق دارد) |
| `VERSION` | — | تگ ایمیج، مثل `1.4.0` (فقط `build-image`) |
| `ARGS` | — | فلگ‌های اضافه که مستقیم به اسکریپت داده می‌شود |
| `SUDO` | `sudo` | برای دستورهای nginx؛ اگر root هستید `SUDO=` بگذارید |

هر فلگی که اسکریپت پشتیبانی کند را می‌توان با `ARGS` فرستاد، مثلاً `ARGS="--latest --network host"`.

---

### `make build-image`

ایمیج داکر یک برنامه را از مخزن کناری آن می‌سازد. اسکریپت: `scripts/build-image.sh`

| پارامتر | الزامی | توضیح |
|---|---|---|
| `APP` | بله | یکی از `IMAGES` در `deploy.conf`: `gozargah` ، `gozargah-worker` یا `gozarban` |
| `VERSION` | بله | تگ ایمیج؛ خروجی `<APP>:<VERSION>` |

فلگ‌های قابل ارسال با `ARGS`:

| فلگ | توضیح |
|---|---|
| `--latest` | علاوه بر نسخه، تگ `<APP>:latest` هم می‌زند |
| `--build-arg KEY=VAL` | build arg اضافه (قابل تکرار)؛ بر مقدار `.env` اولویت دارد |
| `--network MODE` | شبکه‌ی مراحل `RUN`، مثلاً `host` (یا متغیر `DOCKER_BUILD_NETWORK`) |

نکات:
- مخزن، Dockerfile و build arg های هر ایمیج در `IMAGES` در `deploy.conf` تعریف شده است.
- `gozargah-worker` از همان Dockerfile بک‌اند با `CMD_PATH=./cmd/worker` ساخته می‌شود.
- برای `gozarban` این مقادیر از `.env` خوانده و در bundle قرار می‌گیرند:
  - `NEXT_PUBLIC_API_BASE_URL` — **الزامی**؛ آدرس بک‌اند که مرورگر صدا می‌زند.
  - `NEXT_BASE_PATH` — اگر در فایل نباشد پیش‌فرض Dockerfile (`/admin`) استفاده می‌شود؛ اگر خالی باشد بدون base path.
- `.env` خود مخزن gozarban در build استفاده **نمی‌شود** (در `.dockerignore` است).
- مسیر مخزن‌ها را با `<REPO>_DIR` (مثلاً `GOZARGAH_DIR` و `GOZARBAN_DIR`) می‌توان عوض کرد.

```bash
make build-image APP=gozargah VERSION=1.4.0
make build-image APP=gozargah-worker VERSION=1.4.0 ARGS="--latest"
make build-image APP=gozarban VERSION=1.4.0

# شبکه‌ی host برای npm ci / go mod download پشت پروکسی
make build-image APP=gozarban VERSION=1.4.0 ARGS="--network host"

# override یک مقدار .env فقط برای همین build
make build-image APP=gozarban VERSION=1.4.0 ARGS="--build-arg NEXT_PUBLIC_API_BASE_URL=https://example.com/api"
```

---

### `make deploy`

برنامه‌ها را با docker compose بالا می‌آورد و سپس `gozar.conf` را روی nginx میزبان نصب می‌کند
(همان کار `make install-nginx`؛ اگر `nginx/<PROJECT>.conf.template` نباشد این مرحله رد می‌شود).
همه‌چیز از ریشه‌ی gozarsaz خوانده می‌شود. اسکریپت: `scripts/deploy.sh`

| پارامتر | پیش‌فرض | توضیح |
|---|---|---|
| `APP` | `all` | یکی از مقادیر جدول زیر (`TARGETS` در `deploy.conf` به‌اضافه‌ی `nginx` و `all`) |

| `APP` | سرویس‌های compose | ایمیجی که در حالت `local` ساخته می‌شود |
|---|---|---|
| `gozargah` | `gozargah` | `gozargah:local` |
| `workers` | `vpn-conn-monitor` ، `clickhouse-sink` | `gozargah-worker:local` |
| `gozarban` | `gozarban` | `gozarban:local` |
| `infra` | `nats` ، `postgres` ، `clickhouse` ، `ch-ui` | — |
| `nginx` | — (فقط nginx) | — |
| `all` | همه | هر سه |

وابستگی‌ها (`depends_on`) خودکار بالا می‌آیند؛ مثلاً `APP=gozargah` اگر postgres خاموش باشد آن را هم روشن می‌کند.

فلگ‌های قابل ارسال با `ARGS`:

| فلگ | توضیح |
|---|---|
| `--mode auto\|local\|ci\|images` | `auto` (پیش‌فرض): در CI حالت `ci`، در غیر این صورت `local` |
| `--branch NAME` | شاخه‌ای که در حالت `local` pull می‌شود (پیش‌فرض `<REPO>_BRANCH` مثل `GOZARGAH_BRANCH` ، یا `main`) |
| `--image NAME=REF` | `<NAME>_IMAGE` را در `.env` می‌نویسد (حالت `ci`؛ قابل تکرار)، مثلاً `--image gozargah-worker=REF` |
| `--<name>-image REF` | همان `--image <name>=REF`، مثلاً `--gozargah-image REF` یا `--gozargah-worker-image REF` |
| `--skip-nginx` | مرحله‌ی nginx را اجرا نکن |

حالت‌ها:
- **`local`**: مخزن برنامه روی همین سرور `git pull` می‌شود و ایمیج با `build-image.sh` ساخته می‌شود
  (همان build arg های `make build-image`). مسیر مخزن‌ها به‌طور پیش‌فرض کنار gozarsaz است
  (`../gozargah` و `../gozarban`) و با `GOZARGAH_DIR` / `GOZARBAN_DIR` در `.env` قابل تغییر است.
  هر مخزن در هر اجرا فقط یک بار pull می‌شود.
- **`ci`**: ایمیج‌ها از registry کشیده می‌شوند (`compose pull`).
- **`images`**: نه build نه pull؛ همان ایمیج‌هایی که در `.env` آمده و روی سرور موجودند
  (مثلاً با `make build-image` روی همین سرور ساخته شده‌اند) اجرا می‌شوند.

در پایان، `DEPLOYED_AT` و (در حالت `local`) تگ و SHA ایمیج‌ها (`<IMAGE>_IMAGE` و `<IMAGE>_SHA`) در `.env` نوشته می‌شود.

```bash
make deploy                                   # همه، حالت local، با nginx
make deploy APP=gozargah
make deploy APP=workers
make deploy APP=gozarban ARGS="--branch develop"
make deploy APP=infra ARGS="--skip-nginx"
make deploy APP=nginx                         # فقط nginx
make deploy APP=all ARGS="--mode images"      # ایمیج‌هایی که از قبل روی سرور هستند

# CI / registry
make deploy APP=gozargah ARGS="--mode ci --gozargah-image registry.example.com/gozargah:1.4.0"
```

---

### `make render-nginx`

`nginx/<PROJECT>.conf.template` (اینجا `nginx/gozar.conf.template`) را با مقادیر `.env` پر می‌کند و **فقط چاپ** می‌کند (چیزی نصب نمی‌شود). اسکریپت: `scripts/render-nginx.sh`

هر متغیر به شکل `${VAR}` (با آکولاد) در template جایگزین می‌شود و باید در `.env` باشد، وگرنه خطا می‌دهد.
متغیرهای خود nginx بدون آکولاد نوشته می‌شوند (`$host`) و دست نمی‌خورند. متغیرهای فعلی:

| کلید | کاربرد در nginx |
|---|---|
| `GOZARGAH_BIND_ADDRESS` / `GOZARGAH_HOST_PORT` | upstream مسیر `/api/` |
| `GOZARBAN_BIND_ADDRESS` / `GOZARBAN_HOST_PORT` | upstream مسیر `/admin` |
| `CH_UI_BIND_ADDRESS` / `CH_UI_HOST_PORT` | upstream مسیر `/ch-ui/` |

همین کلیدها در `docker-compose.yml` هم برای publish پورت‌ها استفاده می‌شوند، پس nginx و داکر همیشه یکسان‌اند.

```bash
make render-nginx
make render-nginx > /tmp/gozar.conf   # ذخیره برای بررسی
```

---

### `make install-nginx`

مثل `render-nginx`، ولی خروجی را نصب می‌کند، `nginx -t` می‌گیرد و nginx را reload می‌کند. اگر `nginx -t` خطا بدهد، فایل قبلی برگردانده می‌شود. با `sudo` اجرا می‌شود.

| پارامتر | پیش‌فرض | توضیح |
|---|---|---|
| `SUDO` | `sudo` | اگر root هستید: `SUDO=` |

فلگ‌های قابل ارسال با `ARGS`:

| فلگ | توضیح |
|---|---|
| `--dest PATH` | مسیر نصب (پیش‌فرض `/etc/nginx/conf.d/<PROJECT>.conf` یا متغیر `NGINX_DEST`) |

```bash
make install-nginx
make install-nginx ARGS="--dest /etc/nginx/sites-enabled/gozar.conf"
make install-nginx SUDO=          # وقتی خودتان root هستید
```

---

### `make check`

بررسی سلامت. اسکریپت: `scripts/check.sh`

- همه‌ی سرویس‌های `docker-compose.yml` در حال اجرا باشند.
- nginx میزبان فعال باشد، `nginx -t` بگذرد و فایل نصب‌شده با خروجی فعلی `render-nginx` یکی باشد
  (اگر نباشد یعنی `.env` یا template عوض شده و `make install-nginx` لازم است).
- چک‌های اختصاصی پروژه از تابع `project_checks` در `deploy.conf` (پاسخ HTTP مسیرها، باز بودن پورت postgres).

فلگ‌های قابل ارسال با `ARGS`:

| فلگ | توضیح |
|---|---|
| `--verbose` | جزئیات بیشتر |

```bash
make check
make check ARGS="--verbose"
```

---

## سناریوهای رایج

### راه‌اندازی اولیه روی سرور

```bash
sudo scripts/bootstrap.sh            # یک بار: Docker + nginx
cp .env.example .env                 # مقادیر را تنظیم کنید
# کلیدهای JWT و فایل‌های config را در gozargah/secrets/ بگذارید
make render-nginx                    # بررسی خروجی nginx
make deploy                          # pull + build + compose up + nginx
make check
```

### انتشار نسخه‌ی مشخص

```bash
make build-image APP=gozargah        VERSION=1.4.0
make build-image APP=gozargah-worker VERSION=1.4.0
make build-image APP=gozarban        VERSION=1.4.0

# تگ‌ها را در .env بگذارید
#   GOZARGAH_IMAGE=gozargah:1.4.0
#   GOZARGAH_WORKER_IMAGE=gozargah-worker:1.4.0
#   GOZARBAN_IMAGE=gozarban:1.4.0
make deploy APP=all ARGS="--mode images"
make check
```

### فقط تغییر پورت یا آدرس یک سرویس

```bash
# .env را ویرایش کنید، مثلاً GOZARGAH_HOST_PORT=8089
make deploy APP=gozargah   # داکر با پورت جدید و سپس nginx با همان پورت
```
