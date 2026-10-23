# GATE UI Design Spec

> **هدف:** الگوی بصری برای ساخت اپلیکیشن جدید 
> **فلسفه:** Enterprise SaaS — تمیز، حرفه‌ای، با accent teal/cyan، پشتیبانی کامل از Dark Mode و RTL فارسی

---

## 1. Tech Stack پیشنهادی

| مورد | مقدار |
|------|-------|
| CSS Framework | Tailwind CSS v3+ |
| Dark Mode | `class` strategy (`next-themes`) |
| فونت | **Vazirmatn** (Google Fonts, subset: arabic) |
| جهت | RTL (`dir="rtl"`, `lang="fa"`) |
| آیکون‌ها | SVG inline (Lucide-style, stroke 2px) |

---

## 2. پالت رنگ اصلی (Brand Tokens)

### 2.1 Navy — رنگ خنثی / سطح / متن

| Token | Hex | کاربرد |
|-------|-----|--------|
| `brand-navy-50` | `#eefaf8` | پس‌زمینه صفحه (Light) |
| `brand-navy-100` | `#d8f0ed` | border ملایم، header جدول |
| `brand-navy-200` | `#b8d4d1` | border input/button |
| `brand-navy-300` | `#8aadaa` | placeholder، dot غیرفعال |
| `brand-navy-400` | `#6b8f8c` | متن ثانویه، hint |
| `brand-navy-500` | `#527370` | متن توضیحی |
| `brand-navy-600` | `#3f5a58` | label، header جدول |
| `brand-navy-700` | `#334847` | border dark input |
| `brand-navy-800` | `#2a3030` | عنوان، border card dark |
| `brand-navy-900` | `#1f2424` | پس‌زمینه card/sidebar dark |
| `brand-navy-950` | `#141818` | پس‌زمینه صفحه (Dark) |

### 2.2 Blue — accent اصلی (در واقع Teal/Cyan)

| Token | Hex | کاربرد |
|-------|-----|--------|
| `brand-blue-50` | `#e6faf9` | پس‌زمینه badge/icon |
| `brand-blue-100` | `#ccf3f1` | avatar background |
| `brand-blue-200` | `#99e6e3` | — |
| `brand-blue-300` | `#5fd4d0` | — |
| `brand-blue-400` | `#2ec4c0` | hover dark mode |
| `brand-blue-500` | `#26b0b0` | **Primary accent** — CTA، focus ring |
| `brand-blue-600` | `#1e9696` | دکمه primary (Light) |
| `brand-blue-700` | `#1a7879` | hover primary |
| `brand-blue-800` | `#175f60` | chart secondary |
| `brand-blue-900` | `#144e4f` | — |
| `brand-blue-950` | `#0c3233` | badge dark background |

---

## 3. رنگ‌های Landing Page (مارکتینگ)

| Token | Hex | کاربرد |
|-------|-----|--------|
| `landing-bg-dark` | `#061f21` | hero، header، section تیره |
| `landing-bg-darker` | `#041618` | footer، input background |
| `landing-bg-light` | `#f4f8f8` | section روشن |
| `landing-border` | `#d7e3e3` | جداکننده grid |
| `landing-text-primary` | `#061f21` | عنوان |
| `landing-text-body` | `#3d5658` | متن بدنه |
| `landing-text-muted` | `#5a7274` | caption |
| `landing-accent` | `#26b0b0` | CTA، eyebrow |
| `landing-accent-hover` | `#1e9696` | hover CTA |
| `landing-accent-light` | `#7bdcb5` | متن accent روی پس‌زمینه تیره |
| `landing-success-dot` | `#00d084` | bullet list |

---

## 4. رنگ‌های Semantic (Tailwind defaults)

برای وضعیت‌ها از پالت استاندارد Tailwind استفاده می‌شود:

| معنا | Light text | Light bg | Dot/Bar | Dark text | Dark bg |
|------|-----------|----------|---------|-----------|---------|
| **Success** | `emerald-700` | `emerald-50` | `emerald-500` | `emerald-400` | `emerald-950/40` |
| **Warning** | `amber-700` | `amber-50` | `amber-500` | `amber-400` | `amber-950/40` |
| **Error/Danger** | `red-700` | `red-50` | `red-500` | `red-400` | `red-950/40` |
| **Info** | `sky-700` | `sky-50` | `sky-500` | `sky-400` | `sky-950/40` |
| **Neutral/Disabled** | `brand-navy-500` | `brand-navy-50` | `brand-navy-300` | `brand-navy-400` | `brand-navy-800` |
| **Violet** | `violet-700` | `violet-50` | — | `violet-300` | `violet-950/40` |
| **Orange** | `orange-700` | `orange-50` | — | `orange-400` | `orange-950/40` |
| **Teal** | `teal-700` | `teal-50` | — | `teal-400` | `teal-950/40` |

---

## 5. رنگ‌های Chart / Data Visualization

```text
Primary line/bar:     #26b0b0  (brand-blue-500)
Secondary:            #175f60  (brand-blue-800)
Tertiary:             #2ec4c0  (brand-blue-400)
Success metric:       #059669  (emerald-600)
Warning metric:       #d97706  (amber-600)
Danger metric:        #dc2626  (red-600)
Purple metric:        #7c3aed  (violet-600)
Neutral metric:       #527370  (brand-navy-500)

Geo map gradient:     from #0b3a45 → via #0d4a52 → to #0a2f38
Map dots success:     #10b981
Map dots warning:     #f59e0b
Map dots error:       #ef4444
```

---

## 6. Shadow & Gradient

```css
/* Shadows */
shadow-card:      0 4px 24px -4px rgba(20, 78, 79, 0.12)
shadow-card-dark: 0 4px 28px -4px rgba(0, 0, 0, 0.55), 0 0 24px -8px rgba(38, 176, 176, 0.12)
shadow-glow:      0 0 20px rgba(38, 176, 176, 0.35)

/* Login gradient (Light) */
bg-login-gradient:
  radial-gradient(ellipse 80% 60% at 20% 40%, rgba(38,176,176,0.18) 0%, transparent 60%),
  radial-gradient(ellipse 60% 50% at 80% 80%, rgba(38,176,176,0.10) 0%, transparent 55%),
  linear-gradient(180deg, #eefaf8 0%, #d5f2ef 100%)

/* Login gradient (Dark) */
bg-login-gradient-dark:
  radial-gradient(ellipse 70% 50% at 50% 20%, rgba(38,176,176,0.16) 0%, transparent 55%),
  radial-gradient(ellipse 60% 50% at 80% 80%, rgba(4,245,215,0.06) 0%, transparent 55%),
  linear-gradient(180deg, #1f2424 0%, #141818 100%)
```

---

## 7. Typography Scale

| نقش | Class | وزن |
|-----|-------|-----|
| Page title | `text-xl font-bold text-brand-navy-800 dark:text-white` | 700 |
| Section title | `text-base font-bold` یا `text-sm font-semibold` | 700/600 |
| Card value (KPI) | `text-xl sm:text-2xl font-bold tabular-nums` | 700 |
| Hero KPI | `text-2xl sm:text-3xl font-bold` | 700 |
| Body | `text-sm` | 400 |
| Label | `text-sm font-medium text-brand-navy-700 dark:text-brand-navy-200` | 500 |
| Caption/Hint | `text-xs text-brand-navy-500 dark:text-brand-navy-400` | 400 |
| Micro | `text-[11px]` یا `text-[10px] text-brand-navy-400` | 400 |
| Landing H1 | `text-3xl sm:text-4xl font-bold tracking-tight` | 700 |
| Landing eyebrow | `text-sm font-semibold tracking-wide text-[#26b0b0]` | 600 |

**نکته:** اعداد با `tabular-nums` و متن LTR (ایمیل، IP) با `dir="ltr"`.

---

## 8. Border Radius

| المان | Radius |
|--------|--------|
| Input, Button, Card | `rounded-xl` (12px) |
| Modal | `rounded-xl` |
| Login card | `rounded-2xl` (16px) |
| Badge pill | `rounded-full` |
| Badge rect | `rounded-md` |
| Tab/Filter chip | `rounded-lg` |
| Avatar/Icon box | `rounded-lg` |
| Progress bar | `rounded-full` |
| Landing CTA | `rounded-md` |

---

## 9. Layout & Spacing

```text
Page padding:           p-4 sm:p-6
Section gap:            space-y-6
Card padding:           p-4
Modal padding:          px-5 py-4 (header/footer)
Grid KPI:               grid-cols-2 lg:grid-cols-4 gap-3 lg:gap-4
Sidebar width:          w-64 (256px)
Header height:          h-16 (dashboard) / h-14 (portal)
Main bg:                bg-brand-navy-50 dark:bg-brand-navy-950
Card bg:                bg-white dark:bg-brand-navy-900
Card border:            border border-brand-navy-100 dark:border-brand-navy-800
```

---

## 10. Component Patterns

### 10.1 Button — Primary (Full Width / Form)

```css
.btn-primary {
  w-full rounded-xl bg-brand-blue-600 px-4 py-2.5 text-sm font-medium text-white
  transition-all hover:bg-brand-blue-700
  focus:outline-none focus:ring-2 focus:ring-brand-blue-500/40
  active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60
}
/* Dark */
.dark .btn-primary {
  bg-brand-blue-500 shadow-glow hover:bg-brand-blue-400
}
```

### 10.2 Button — Action Primary (Inline CTA)

```css
.btn-action-primary {
  inline-flex items-center justify-center gap-2
  rounded-xl bg-brand-blue-600 px-4 py-2.5 text-sm font-medium text-white
  transition-all hover:bg-brand-blue-700
  focus:ring-2 focus:ring-brand-blue-500/40 active:scale-[0.98]
}
/* Dark: same as btn-primary */
```

### 10.3 Button — Secondary / Ghost

```css
rounded-lg border border-brand-navy-200 px-3 py-2 text-xs font-medium
text-brand-navy-700 hover:bg-brand-navy-50
dark:border-brand-navy-700 dark:text-brand-navy-200 dark:hover:bg-brand-navy-800
```

### 10.4 Button — Danger

```css
text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/30
/* یا border hover: hover:border-red-300 hover:text-red-600 */
```

### 10.5 Input Field

```css
.input-field {
  w-full rounded-xl border border-brand-navy-200 bg-white px-4 py-2.5
  text-sm text-brand-navy-900 outline-none transition-all
  placeholder:text-brand-navy-300
  focus:border-brand-blue-500 focus:ring-2 focus:ring-brand-blue-500/25
}
/* Dark */
.dark .input-field {
  border-brand-navy-700 bg-brand-navy-900 text-brand-navy-50
  placeholder:text-brand-navy-500
  focus:border-brand-blue-400 focus:ring-brand-blue-400/25
}
```

### 10.6 Card (Standard)

```css
rounded-xl border border-brand-navy-100 bg-white
dark:border-brand-navy-800 dark:bg-brand-navy-900
```

### 10.7 Card (Elevated / Login)

```css
rounded-2xl border border-brand-navy-100/80 bg-white/90 p-8 shadow-card backdrop-blur-md
dark:border-brand-navy-800 dark:bg-brand-navy-900/90 dark:shadow-card-dark
```

### 10.8 Modal

```css
/* Backdrop */
fixed inset-0 bg-brand-navy-950/50 backdrop-blur-sm

/* Panel */
relative max-h-[90vh] w-full overflow-hidden rounded-xl
border border-brand-navy-100 bg-white shadow-card
dark:border-brand-navy-800 dark:bg-brand-navy-900 dark:shadow-card-dark

/* Z-index layers */
base: z-50 | stack: z-[60] | top (confirm): z-[70]
```

### 10.9 Table

```css
/* Container */
overflow-hidden rounded-xl border border-brand-navy-100 bg-white
dark:border-brand-navy-800 dark:bg-brand-navy-900

/* Header row */
border-b border-brand-navy-100 bg-brand-navy-50/50
th: px-4 py-3 text-start text-xs font-semibold text-brand-navy-600 dark:text-brand-navy-300

/* Body row */
hover:bg-brand-navy-50/50 dark:hover:bg-brand-navy-800/30
td: px-4 py-3 text-xs

/* Empty state */
rounded-xl border border-dashed border-brand-navy-200 bg-white py-16
dark:border-brand-navy-700 dark:bg-brand-navy-900
```

### 10.10 Badge — Pill (Status)

```css
inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset
/* مثال active: */
bg-emerald-50 text-emerald-700 ring-emerald-600/20
dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-500/30
```

### 10.11 Badge — Dot Status

```css
inline-flex items-center gap-1.5 text-xs font-medium
/* dot: h-1.5 w-1.5 rounded-full bg-emerald-500 */
/* active states: animate-pulse */
```

### 10.12 Badge — Rect (Tag)

```css
inline-flex rounded-md px-2 py-0.5 text-[10px] font-medium
bg-brand-blue-50 text-brand-blue-700 dark:bg-brand-blue-950/40 dark:text-brand-blue-300
```

### 10.13 Progress Bar

```css
/* Track */
h-1.5 overflow-hidden rounded-full bg-brand-navy-100 dark:bg-brand-navy-800
/* Fill */
h-full rounded-full transition-all bg-brand-blue-600
/* Warning: bg-amber-500 | Danger: bg-red-500 */
```

### 10.14 Tab (Primary)

```css
/* Active */
rounded-lg px-4 py-2 text-sm font-medium bg-brand-blue-600 text-white dark:bg-brand-blue-500
/* Inactive */
bg-white text-brand-navy-600 shadow-sm ring-1 ring-brand-navy-100 hover:bg-brand-navy-50
dark:bg-brand-navy-900 dark:text-brand-navy-300 dark:ring-brand-navy-800
```

### 10.15 Filter Chip

```css
/* Active (primary filter) */
rounded-lg px-3 py-1.5 text-xs font-medium bg-brand-blue-600 text-white
/* Active (secondary filter) */
bg-brand-navy-700 text-white dark:bg-brand-navy-600
/* Inactive */
bg-brand-navy-50 text-brand-navy-600 hover:bg-brand-navy-100
dark:bg-brand-navy-800 dark:text-brand-navy-300
```

### 10.16 Toast / Alert Banner

```css
/* Success/Info toast */
rounded-xl border border-brand-blue-200 bg-brand-blue-50 px-4 py-3 text-sm text-brand-blue-800
dark:border-brand-blue-900 dark:bg-brand-blue-950/40 dark:text-brand-blue-200

/* Error */
rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700
dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300

/* Warning */
rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900
dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200
```

### 10.17 Sidebar Nav Item

```css
/* Active */
flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium
bg-brand-blue-50 text-brand-blue-700 dark:bg-brand-blue-950/50 dark:text-brand-blue-300

/* Inactive */
text-brand-navy-600 hover:bg-brand-navy-50 hover:text-brand-navy-900
dark:text-brand-navy-300 dark:hover:bg-brand-navy-800 dark:hover:text-white
```

### 10.18 Header (Sticky)

```css
sticky top-0 z-30 h-16 flex items-center justify-between
border-b border-brand-navy-100 bg-white/80 backdrop-blur-md
dark:border-brand-navy-800 dark:bg-brand-navy-950/80
```

### 10.19 Theme Toggle

```css
h-9 w-9 rounded-lg border border-brand-navy-200 bg-white/80 backdrop-blur-sm
text-brand-navy-600 hover:border-brand-blue-400 hover:text-brand-blue-600
dark:border-brand-navy-700 dark:bg-brand-navy-900/80 dark:text-brand-navy-200
```

### 10.20 KPI Card

```css
rounded-xl border border-brand-navy-100 bg-white p-4 dark:border-brand-navy-800 dark:bg-brand-navy-900
/* Icon box */
rounded-lg p-2 text-brand-blue-600 bg-brand-blue-50 dark:bg-brand-blue-950/40 dark:text-brand-blue-400
/* Value */
text-xl sm:text-2xl font-bold tabular-nums text-brand-navy-800 dark:text-white
```

### 10.21 Hero KPI (Executive — با gradient accent)

```css
relative overflow-hidden rounded-xl border ... p-4
/* Top gradient bar */
absolute inset-x-0 top-0 h-16 bg-gradient-to-b from-brand-blue-500/20 to-transparent
/* accent variants: teal-500/20, emerald-500/20, violet-500/20, brand-navy-500/15 */
```

---

## 11. Dark Mode Rules

```text
Body Light:  bg-brand-navy-50  text-brand-navy-900
Body Dark:   bg-brand-navy-950  text-brand-navy-50

Primary button Light: brand-blue-600 → hover 700
Primary button Dark:  brand-blue-500 + shadow-glow → hover 400

Card Light: white + border-navy-100
Card Dark:  navy-900 + border-navy-800

Transition: transition-colors duration-300 (body)
```

---

## 12. Page Structure Template

```text
DashboardShell
├── Sidebar (w-64, hidden lg:flex)
│   ├── Brand (logo + title)
│   ├── Nav items
│   └── Footer meta
├── Main column
│   ├── Header (sticky, h-16)
│   └── Main (p-4 sm:p-6)
│       └── PageContent (space-y-6)
│           ├── PageHeader (title + description + CTA)
│           ├── Toast (optional)
│           ├── Stats/KPIs grid
│           ├── Tabs/Filters bar
│           └── Table/Content card
```

---

## 13. Tailwind Config (کپی مستقیم)

```typescript
// tailwind.config.ts — colors section
colors: {
  brand: {
    navy: {
      50: "#eefaf8", 100: "#d8f0ed", 200: "#b8d4d1", 300: "#8aadaa",
      400: "#6b8f8c", 500: "#527370", 600: "#3f5a58", 700: "#334847",
      800: "#2a3030", 900: "#1f2424", 950: "#141818",
    },
    blue: {
      50: "#e6faf9", 100: "#ccf3f1", 200: "#99e6e3", 300: "#5fd4d0",
      400: "#2ec4c0", 500: "#26b0b0", 600: "#1e9696", 700: "#1a7879",
      800: "#175f60", 900: "#144e4f", 950: "#0c3233",
    },
  },
},
fontFamily: {
  sans: ["var(--font-vazirmatn)", "system-ui", "sans-serif"],
},
boxShadow: {
  card: "0 4px 24px -4px rgba(20, 78, 79, 0.12)",
  "card-dark": "0 4px 28px -4px rgba(0, 0, 0, 0.55), 0 0 24px -8px rgba(38, 176, 176, 0.12)",
  glow: "0 0 20px rgba(38, 176, 176, 0.35)",
},
darkMode: "class",
```

---

## 14. Prompt آماده برای ایجنت

می‌توانید این بلوک را مستقیم به ایجنت بدهید:

```markdown
Build UI following this design system:

- RTL Persian app with Vazirmatn font
- Tailwind CSS with custom brand colors (navy neutrals + teal accent #26b0b0)
- Dark mode via class strategy
- Style: enterprise SaaS, clean cards, subtle borders, teal glow in dark mode

Colors:
- Page bg: navy-50 / navy-950 (dark)
- Cards: white / navy-900 (dark), border navy-100 / navy-800
- Primary CTA: blue-600 (light) / blue-500 + glow (dark)
- Text: navy-800 titles, navy-500 captions
- Status: emerald=success, amber=warning, red=error, sky=info

Components:
- Border radius: xl for cards/inputs/buttons, 2xl for login cards
- Buttons: btn-primary (full), btn-action-primary (inline), secondary with border
- Inputs: rounded-xl, focus ring brand-blue-500/25
- Tables: rounded-xl container, subtle header bg navy-50/50
- Badges: pill with ring-1 for status, dot badges with animate-pulse for active
- Modals: backdrop blur + navy-950/50, shadow-card
- KPI cards: icon in colored box + bold tabular-nums value
- Progress bars: h-1.5 rounded-full

Layout:
- Sidebar 256px, sticky header with backdrop-blur
- Page spacing: space-y-6, padding p-4 sm:p-6
- Page header: text-xl font-bold + text-sm description + action button
```

---

## خلاصه

| بخش | منبع اصلی در پروژه |
|-----|-------------------|
| توکن‌های رنگ | `tailwind.config.ts` |
| کلاس‌های utility | `src/app/globals.css` |
| Layout dashboard | `src/components/layout/dashboard-shell.tsx` |
| Modal | `src/components/ui/modal.tsx` |
| Badge patterns | `src/components/*/*-badges.tsx` |
| Landing colors | `src/components/landing/*.tsx` |
