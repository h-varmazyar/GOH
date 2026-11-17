# Deep Forest Green — Design System

> A calm, intellectual, timeless visual system for web applications,
> dashboards, and mobile products.

**Version:** 1.1
**Status:** Refined specification (all v1.0 brand values preserved)
**Primary brand color:** `#263B38`

---

## 1. Design principles

1. **Quiet confidence** — use deep greens, restrained contrast, and deliberate visual hierarchy rather than loud decoration.
2. **Clarity before ornament** — components should make information easier to scan and actions easier to understand.
3. **Nature-inspired, not rustic** — forest hues provide the identity; typography, spacing, and controls remain contemporary.
4. **Consistent semantics** — success, warning, error, and information colors communicate status consistently and are not replaced by brand colors.
5. **Accessible by default** — use sufficient contrast, visible focus states, clear labels, and non-color status cues.

---

## 2. Color system

The system has two layers:

- **Primitive tokens** (sections 2.1–2.5) — the raw palette. Brand primitives **never change between themes**; they are the identity.
- **Semantic tokens** (section 3) — roles such as `bg`, `text`, `action`, `border-strong`. Components use **only** semantic tokens; themes remap them.

Contrast ratios below are WCAG 2.x values, measured on the stated background.

### 2.1 Brand palette

| Token | Hex | RGB | Intended use |
|---|---|---|---|
| `color.brand.primary` | `#263B38` | 38, 59, 56 | Primary brand color, navigation, key actions |
| `color.brand.primary-light` | `#3F5F5A` | 63, 95, 90 | Hover surfaces, secondary emphasis, links on light surfaces |
| `color.brand.primary-dark` | `#1A2A28` | 26, 42, 40 | Pressed states, dark emphasis |
| `color.brand.mint` | `#4E8B7F` | 78, 139, 127 | Accent, selected indicators, focus ring, subtle highlights |
| `color.brand.mint-light` | `#6AA397` | 106, 163, 151 | Dark-theme action and accent (already used in v1.0 dark tokens; now documented) |

**Mint usage rule.** Mint is a *non-text* accent on light surfaces: on white it reaches 3.94:1 — enough for focus rings, indicators, and icons (≥ 3:1), but **not** for body text or for white button labels (< 4.5:1). For text in light theme use `primary` or `primary-light`.

### 2.2 Neutral palette

| Token | Hex | Intended use |
|---|---|---|
| `color.neutral.950` | `#1B1F1E` | Light-theme primary text; dark-theme app background |
| `color.neutral.800` | `#34413E` | Strong secondary text; dark-theme elevated surface |
| `color.neutral.700` | `#4A5654` | Secondary text on light surfaces; dark-theme decorative border |
| `color.neutral.600` | `#5F6E6B` | **New.** Subtle text, placeholders, captions on light surfaces (5.34:1 on white, 4.86:1 on `#F1F5F3`) |
| `color.neutral.500` | `#7A8A87` | Icons, disabled text, strong (input) borders — **not** for readable text on light surfaces (3.61:1) |
| `color.neutral.300` | `#C7D0CD` | Decorative borders, dividers; dark-theme secondary text |
| `color.neutral.100` | `#E7EDEA` | Subtle surfaces |
| `color.neutral.50` | `#F1F5F3` | Light application background; dark-theme primary text |
| `color.white` | `#FFFFFF` | Cards and elevated surfaces |

`neutral.600` fills the only gap in the scale and sits on the same green-gray hue line as its neighbours.

### 2.3 Semantic colors

| Token | Light theme | Dark theme | Meaning |
|---|---|---|---|
| `color.success` | `#2E7D32` | `#81C784` | Successful operations, positive status |
| `color.info` | `#1976D2` | `#90CAF9` | Informational messages |
| `color.warning` | `#F9A825` | `#F9A825` | Caution, pending attention |
| `color.error` | `#D32F2F` | `#EF9A9A` | Errors, destructive actions |

Rules:

- Light-theme values are unchanged from v1.0. The light semantic values only reach about 3.3:1 on dark surfaces, so dark theme uses lighter tints of the same hues for text and icons.
- **Warning is never paired with white text** (1.97:1). Text on a warning fill is always `#1B1F1E` (8.45:1).
- White text on `success`, `info`, and `error` fills passes AA (5.13, 4.60, 4.98:1).
- Semantic colors must be paired with an icon, label, or other non-color indicator. Do not communicate status by color alone.

### 2.4 Optional accent colors

Use sparingly for data visualization, category distinctions, or product-specific accents.

| Token | Hex | Text on fill |
|---|---|---|
| `color.accent.teal` | `#00BFA5` | `#1B1F1E` only |
| `color.accent.cyan` | `#00BCD4` | `#1B1F1E` only |
| `color.accent.indigo` | `#3F51B5` | `#FFFFFF` |
| `color.accent.purple` | `#7E57C2` | `#FFFFFF` |

Accents are for marks (chart lines, bars, dots, category tags), not large surfaces or body text.

### 2.5 Gradients

| Token | Value | Use |
|---|---|---|
| `gradient.brand` | `linear-gradient(110deg, #263B38 0%, #4E8B7F 100%)` | Hero areas, banners, occasional emphasis |

Avoid gradients behind dense text or essential controls. White text holds AA only over the dark half of this gradient; at the mint end it drops to 3.94:1. Place text on the `#263B38` side, use large text (≥ 24px, or ≥ 18.66px bold), or add an overlay.

---

## 3. Themes (semantic tokens)

Components reference these roles, never raw hex values.

| Role token | Light | Dark | Notes |
|---|---|---|---|
| `bg` | `#F1F5F3` | `#1B1F1E` | App background |
| `surface` | `#FFFFFF` | `#263B38` | Cards, main panels |
| `surface-muted` | `#E7EDEA` | `#34413E` | Subtle / elevated surface |
| `text` | `#1B1F1E` | `#F1F5F3` | 15.1:1 / 10.8:1 on surface |
| `text-muted` | `#4A5654` | `#C7D0CD` | Secondary text |
| `text-subtle` | `#5F6E6B` | `#C7D0CD` | Captions, placeholders, metadata |
| `text-disabled` | `#7A8A87` | `#7A8A87` | Disabled only (exempt from contrast rules) |
| `link` | `#3F5F5A` | `#82B5AA` | Underline links in running text (5.18:1 on dark surface) |
| `border` | `#C7D0CD` | `#4A5654` | Decorative dividers, card outlines |
| `border-strong` | `#7A8A87` | `#8A9A97` | **Input and control boundaries** (≥ 3:1 required on every surface) |
| `action` | `#263B38` | `#6AA397` | Primary button fill |
| `action-hover` | `#3F5F5A` | `#82B5AA` | |
| `action-pressed` | `#1A2A28` | `#5C978B` | |
| `on-action` | `#FFFFFF` | `#1B1F1E` | Text/icon on action fill |
| `focus-ring` | `#4E8B7F` | `#6AA397` | 2px ring, 2px offset |
| `selected` | `#4E8B7F` | `#6AA397` | Indicators, active nav marker |

### Why the dark-theme action changed

In v1.0 the dark-theme primary action was white text on `#4E8B7F` — only 3.94:1, which fails AA for button labels. The fix keeps the same mint family: the fill moves one step lighter to `#6AA397` (already part of v1.0's dark tokens) and the label becomes `#1B1F1E` (5.79:1). Hover and pressed stay within the same hue.

Also fixed: v1.0 set the dark pressed state to `#263B38`, which is identical to the dark `surface`, so a pressed button disappeared into its card.

### Theme switching

- Apply dark theme automatically via `prefers-color-scheme: dark`, and allow a manual override with `data-theme="light"` / `data-theme="dark"` on `<html>`.
- Set `color-scheme: light dark` so native controls (scrollbars, date pickers) follow the theme.

---

## 4. Typography

Use a modern, highly legible sans-serif family:

```css
font-family: Inter, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
```

For Persian and Arabic interfaces use **Vazirmatn**:

```css
font-family: Vazirmatn, Tahoma, sans-serif;
```

For mixed Persian/Latin content, put Vazirmatn first; it contains Latin glyphs and keeps baselines consistent.

### Type scale

| Token | Size | Weight | Line-height | Typical use |
|---|---|---|---|---|
| `font.display` | 36px / 2.25rem | 700 | 1.15 | Landing-page hero only |
| `font.h1` | 30px / 1.875rem | 700 | 1.2 | Page title |
| `font.h2` | 24px / 1.5rem | 650 | 1.25 | Main section title |
| `font.h3` | 20px / 1.25rem | 600 | 1.3 | Card or subsection title |
| `font.h4` | 16px / 1rem | 600 | 1.3 | Component heading |
| `font.body-lg` | 16px / 1rem | 400 | 1.6 | Lead paragraph |
| `font.body` | 14px / 0.875rem | 400 | 1.5 | Default interface text |
| `font.label` | 13px / 0.8125rem | 600 | 1.4 | Form labels, compact controls |
| `font.caption` | 12px / 0.75rem | 400 | 1.4 | Helper text, metadata |

- Weight 650 requires a variable font (Inter and Vazirmatn both ship variable versions). With static fonts it falls back to 600 — acceptable.
- Persian script needs more vertical room: add **+0.1** to body line-heights (e.g. 1.5 → 1.6, 1.6 → 1.7) in RTL interfaces.
- Do not go below 12px for any readable text; in Persian prefer 13px minimum for captions.
- Use `font-variant-numeric: tabular-nums` for prices, counts, metrics, and table columns.
- Avoid long passages in all caps. (Persian has no case — use weight, not letter-spacing, for emphasis.)
- Never apply `letter-spacing` to Persian/Arabic text; it breaks letter joining.

---

## 5. Spacing and layout

Use a 4px base spacing unit.

| Token | Value |
|---|---|
| `space.1` | 4px |
| `space.2` | 8px |
| `space.3` | 12px |
| `space.4` | 16px |
| `space.5` | 20px |
| `space.6` | 24px |
| `space.8` | 32px |
| `space.10` | 40px |
| `space.12` | 48px |
| `space.16` | 64px |

### Breakpoints

| Token | Min width | Typical layout |
|---|---|---|
| `bp.sm` | 640px | Large phones, single column |
| `bp.md` | 768px | Tablets, two columns |
| `bp.lg` | 1024px | Laptops, sidebar + content |
| `bp.xl` | 1280px | Desktop, max container width |

### Layout guidance

- Centered content container, max width 1200–1280px for desktop applications.
- Page gutters: 16px on mobile, 24px on tablet, 32px on desktop.
- Reading-oriented content and forms: max ~680px (≈ 70 characters per line).
- Use consistent gaps within repeated component groups.
- Do not use spacing alone to imply relationships that should be expressed through grouping or headings.

### RTL

- Use CSS logical properties everywhere (`margin-inline-start`, `padding-inline-end`, `inset-inline-start`, `border-inline-start`) instead of left/right. One stylesheet then serves both directions.
- Mirror directional icons (arrows, chevrons, "back") in RTL; do not mirror non-directional icons (search, check, clock) or media controls.
- Numbers, phone numbers, and code stay LTR inside RTL text — wrap them with `dir="ltr"` or `unicode-bidi: isolate` when needed.

---

## 6. Shape, borders, and elevation

### Radius

| Token | Value | Use |
|---|---|---|
| `radius.sm` | 6px | Compact controls, tags |
| `radius.md` | 10px | Inputs, buttons |
| `radius.lg` | 16px | Cards, panels |
| `radius.xl` | 24px | Feature panels, dialogs |
| `radius.pill` | 999px | Pills and fully rounded badges |

Nested radius rule: inner radius = outer radius − padding (e.g. a 10px button inside a 16px card with 6px inset).

### Borders

- **Decorative** (dividers, card outlines): `1px solid var(--color-border)` — `#C7D0CD` light, `#4A5654` dark.
- **Functional** (inputs, checkboxes, toggles, secondary-button outlines): `1px solid var(--color-border-strong)` — `#7A8A87`. Control boundaries must reach 3:1 (WCAG 1.4.11); `#C7D0CD` reaches only 1.58:1 on white.
- Prefer subtle borders over heavy outlines.
- Use the focus ring, not a thicker border, for focus.

### Shadows

```css
--shadow-sm: 0 1px 2px rgb(27 31 30 / 8%);
--shadow-md: 0 4px 12px rgb(27 31 30 / 10%);
--shadow-lg: 0 12px 32px rgb(27 31 30 / 14%);
```

Use shadows to establish hierarchy, not as decoration on every element. In dark theme, shadows are barely visible; express elevation with a lighter surface (`surface` → `surface-muted`) instead.

### Layering (z-index)

| Token | Value | Use |
|---|---|---|
| `z.base` | 0 | Page content |
| `z.sticky` | 100 | Sticky headers, toolbars |
| `z.dropdown` | 200 | Menus, popovers |
| `z.overlay` | 300 | Modal scrim |
| `z.modal` | 400 | Dialogs, drawers |
| `z.toast` | 500 | Toasts, notifications |

Modal scrim: `rgb(27 31 30 / 48%)` in both themes.

---

## 7. Component specifications

### Buttons

| Variant | Fill | Text | Border | Hover | Pressed |
|---|---|---|---|---|---|
| Primary | `action` | `on-action` | — | `action-hover` | `action-pressed` |
| Secondary | transparent | `#263B38` (dark: `text`) | `border-strong` | `surface-muted` | `border` fill |
| Ghost | transparent | `#263B38` (dark: `text`) | — | `surface-muted` | `border` fill |
| Destructive | `#D32F2F` | `#FFFFFF` | — | `#B71C1C` | `#B71C1C` |

- Radius `10px`; height **40px** (desktop, dense UI) or **44px** (touch, default on mobile).
- Horizontal padding 16px (40px height) / 20px (44px height); icon–label gap 8px.
- **Disabled:** 40% opacity on the whole button, `cursor: not-allowed`, no hover effect; keep it in the tab order only if a tooltip explains why.
- **Loading:** keep the button width fixed, replace the label with a spinner + "Saving…" text for screen readers (`aria-busy="true"`).
- **Focus-visible:** 2px `focus-ring`, 2px offset — on every variant.
- Use verbs that describe the action ("Save changes", not "OK").
- Do not use color alone to distinguish primary and destructive actions — the destructive label states the consequence ("Delete project").
- One primary button per view or section.

### Cards

- Background: `surface` (`#FFFFFF` light; `#263B38` or `#34413E` dark).
- Radius `16px`; padding 20–24px (16px on mobile).
- Border: `1px solid var(--color-border)`; shadow optional (`shadow-sm`).
- Structure: title → optional supporting text → predictable action area at the bottom.
- An entirely clickable card needs one clear link target and a visible hover/focus state.

### Inputs

- Height: 44px (touch) / 40px (dense desktop).
- Background `surface`; text `text`; placeholder `text-subtle` (`#5F6E6B`, 5.34:1).
- Border: `border-strong` (`#7A8A87` light, 3.61:1 on white; `#8A9A97` dark, ≥ 3.6:1 on dark surfaces).
- Focus: 2px `focus-ring` (`#4E8B7F`), 2px offset.
- Error: `error` border + error icon + message below the field, linked with `aria-describedby`; set `aria-invalid="true"`.
- Disabled: `surface-muted` background, `text-disabled` text.
- Labels stay visible above the field at all times. Do not rely on placeholders as labels.
- Helper text uses `font.caption` in `text-subtle`.

### Navigation

- Either a strong brand navigation area (`#263B38` background, `#F1F5F3` text 10.8:1, `#C7D0CD` secondary 7.55:1) or a light surface with dark-green active indicators.
- On a `#263B38` navigation bar, inactive items use `#C7D0CD`, **never** `#7A8A87` (3.29:1 — fails).
- Active items must have both a color change and a secondary cue: a 3px `selected` indicator bar (`border-inline-start` for vertical nav), a weight change, or a filled icon.
- Mark the current page with `aria-current="page"`.
- Keep navigation hierarchy shallow and labels concise.

### Alerts and status messages

| Type | Color | Required treatment | Background (light) |
|---|---|---|---|
| Success | `success` | Check icon + concise confirmation | success at 8% opacity |
| Info | `info` | Info icon + explanation | info at 8% opacity |
| Warning | `warning` | Warning icon + recommended action | warning at 12% opacity |
| Error | `error` | Error icon + recovery guidance | error at 8% opacity |

- Alert body text is always `text`, not the semantic color; the semantic color goes to the icon and a 3px `border-inline-start`.
- For warning, the icon itself is `#1B1F1E` on a warning fill or `#F9A825` on dark surfaces — never yellow on white (1.97:1).
- Errors and urgent status changes use `role="alert"`; passive status uses `role="status"`.

### Tables and data-dense interfaces

- Neutral surfaces for the table body; a restrained brand treatment for headers (`surface-muted` background, `font.label`) or selected rows (`selected` at 10% + `border-inline-start`).
- Align numeric values to the end side (right in LTR, left in RTL) using `text-align: end`.
- Use tabular numerals for prices, counts, and metrics.
- Row height: 40px default, 32px compact, 48px comfortable.
- Separate rows with subtle dividers (`border`) rather than heavy boxes.
- Provide clear empty, loading (skeleton rows), error, and pagination states.
- Do not encode a data series using only shades of green; use distinguishable colors, labels, and/or patterns.

### Data visualization palette

Recommended categorical order (validated to be distinguishable from each other and from neutrals):

1. `#263B38` (brand primary — dark theme: `#6AA397`)
2. `#00BCD4` (cyan)
3. `#7E57C2` (purple)
4. `#F9A825` (warning hue — only when not signalling status)
5. `#3F51B5` (indigo)

Use at most 5 categories before grouping the rest as "Other". Always add direct labels or a legend.

### Badges and chips

- Compact pill shapes (`radius.pill`), height 24px, `font.caption` at weight 600.
- Keep labels short (1–2 words).
- Use semantic colors only when the badge communicates a real status; neutral badges use `surface-muted` + `text-muted`.
- Avoid a different bright color for every category without a clear information architecture.

---

## 8. Iconography and imagery

- Prefer simple outline icons with consistent stroke weight (1.5px at 24px, scaled proportionally).
- Use one icon family throughout the product.
- Default icon sizes: 16px, 20px, and 24px; align icon size with text size (16px icons with 14px text).
- Icons that trigger actions must have accessible names or visible labels; decorative icons get `aria-hidden="true"`.
- Icon-only buttons still need a 44 × 44px touch target, even if the icon is 20px.
- Use natural imagery selectively: forest textures, mist, foliage, and wood tones can support the brand, but should not compete with application data.
- Avoid placing text directly over busy photography without a strong overlay (`#1B1F1E` at ≥ 60%).

---

## 9. Accessibility requirements

Target **WCAG 2.2 AA**.

- Normal text: at least 4.5:1. Large text (≥ 24px, or ≥ 18.66px bold): at least 3:1.
- Control boundaries, focus indicators, and meaningful graphics: at least 3:1 against adjacent colors.
- All interactive elements must be keyboard accessible, in a logical order.
- Focus indicators must be clearly visible and must not be removed; use `:focus-visible`.
- Provide labels, instructions, and programmatically associated error messages for form fields.
- Use semantic HTML and correct heading order; set `lang` and `dir` on `<html>`.
- Support reduced-motion preferences.
- Touch targets: preferably at least 44 × 44px (minimum 24 × 24px per WCAG 2.2 SC 2.5.8).
- Test both themes and all component states with a contrast checker; do not infer compliance from the palette alone.

### Verified pairings (quick reference)

| Foreground | Background | Ratio | Status |
|---|---|---|---|
| `#FFFFFF` | `#263B38` | 11.90 | ✅ Text |
| `#FFFFFF` | `#3F5F5A` | 7.01 | ✅ Text |
| `#1B1F1E` | `#F1F5F3` | 15.14 | ✅ Text |
| `#4A5654` | `#FFFFFF` | 7.64 | ✅ Text |
| `#5F6E6B` | `#FFFFFF` | 5.34 | ✅ Text |
| `#5F6E6B` | `#E7EDEA` | 4.50 | ✅ Text (minimum) |
| `#F1F5F3` | `#263B38` | 10.82 | ✅ Text |
| `#C7D0CD` | `#263B38` | 7.55 | ✅ Text |
| `#1B1F1E` | `#6AA397` | 5.79 | ✅ Text (dark action) |
| `#1B1F1E` | `#F9A825` | 8.45 | ✅ Text |
| `#4E8B7F` | `#FFFFFF` | 3.94 | ⚠️ Non-text only |
| `#7A8A87` | `#FFFFFF` | 3.61 | ⚠️ Non-text only |
| `#FFFFFF` | `#4E8B7F` | 3.94 | ❌ Avoid for labels |
| `#7A8A87` | `#263B38` | 3.29 | ❌ Avoid for text |
| `#FFFFFF` | `#F9A825` | 1.97 | ❌ Never |
| `#C7D0CD` | `#FFFFFF` | 1.58 | ❌ Decorative only |

---

## 10. Motion

| Token | Value | Use |
|---|---|---|
| `duration.fast` | 120ms | Hover, color change |
| `duration.base` | 180ms | Focus, selection, toggles |
| `duration.slow` | 240ms | Small panels, dialogs |
| `ease.standard` | `cubic-bezier(0.2, 0, 0, 1)` | Most transitions |
| `ease.exit` | `cubic-bezier(0.4, 0, 1, 1)` | Elements leaving the screen |

- Prefer opacity and transform over animating layout dimensions.
- Avoid unnecessary motion in dashboards and data-heavy workflows.
- Respect `prefers-reduced-motion`: reduce durations to near zero and remove movement, keeping only opacity changes.

---

## 11. CSS design tokens

```css
:root {
  color-scheme: light dark;

  /* ── Primitives: brand (constant across themes) ── */
  --color-brand-primary: #263B38;
  --color-brand-primary-light: #3F5F5A;
  --color-brand-primary-dark: #1A2A28;
  --color-brand-mint: #4E8B7F;
  --color-brand-mint-light: #6AA397;

  /* ── Primitives: neutral ── */
  --color-neutral-950: #1B1F1E;
  --color-neutral-800: #34413E;
  --color-neutral-700: #4A5654;
  --color-neutral-600: #5F6E6B;
  --color-neutral-500: #7A8A87;
  --color-neutral-300: #C7D0CD;
  --color-neutral-100: #E7EDEA;
  --color-neutral-50: #F1F5F3;
  --color-white: #FFFFFF;

  /* ── Primitives: accent ── */
  --color-accent-teal: #00BFA5;
  --color-accent-cyan: #00BCD4;
  --color-accent-indigo: #3F51B5;
  --color-accent-purple: #7E57C2;

  /* ── Semantic: light theme (default) ── */
  --color-bg: var(--color-neutral-50);
  --color-surface: var(--color-white);
  --color-surface-muted: var(--color-neutral-100);
  --color-text: var(--color-neutral-950);
  --color-text-muted: var(--color-neutral-700);
  --color-text-subtle: var(--color-neutral-600);
  --color-text-disabled: var(--color-neutral-500);
  --color-link: var(--color-brand-primary-light);
  --color-border: var(--color-neutral-300);
  --color-border-strong: var(--color-neutral-500);

  --color-action: var(--color-brand-primary);
  --color-action-hover: var(--color-brand-primary-light);
  --color-action-pressed: var(--color-brand-primary-dark);
  --color-on-action: var(--color-white);
  --color-focus-ring: var(--color-brand-mint);
  --color-selected: var(--color-brand-mint);

  --color-success: #2E7D32;
  --color-info: #1976D2;
  --color-warning: #F9A825;
  --color-on-warning: var(--color-neutral-950);
  --color-error: #D32F2F;
  --color-error-hover: #B71C1C;

  --color-scrim: rgb(27 31 30 / 48%);

  /* ── Typography ── */
  --font-family: Inter, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  --font-family-fa: Vazirmatn, Tahoma, sans-serif;

  --font-size-display: 2.25rem;  /* 36px */
  --font-size-h1: 1.875rem;      /* 30px */
  --font-size-h2: 1.5rem;        /* 24px */
  --font-size-h3: 1.25rem;       /* 20px */
  --font-size-h4: 1rem;          /* 16px */
  --font-size-body-lg: 1rem;     /* 16px */
  --font-size-body: 0.875rem;    /* 14px */
  --font-size-label: 0.8125rem;  /* 13px */
  --font-size-caption: 0.75rem;  /* 12px */

  --font-weight-regular: 400;
  --font-weight-semibold: 600;
  --font-weight-h2: 650;
  --font-weight-bold: 700;

  --line-height-tight: 1.2;
  --line-height-heading: 1.3;
  --line-height-body: 1.5;
  --line-height-relaxed: 1.6;

  /* ── Spacing ── */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;
  --space-16: 64px;

  /* ── Sizing ── */
  --control-height-sm: 40px;
  --control-height-md: 44px;
  --focus-ring-width: 2px;
  --focus-ring-offset: 2px;

  /* ── Shape and elevation ── */
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 16px;
  --radius-xl: 24px;
  --radius-pill: 999px;

  --shadow-sm: 0 1px 2px rgb(27 31 30 / 8%);
  --shadow-md: 0 4px 12px rgb(27 31 30 / 10%);
  --shadow-lg: 0 12px 32px rgb(27 31 30 / 14%);

  --gradient-brand: linear-gradient(110deg, #263B38 0%, #4E8B7F 100%);

  /* ── Layering ── */
  --z-sticky: 100;
  --z-dropdown: 200;
  --z-overlay: 300;
  --z-modal: 400;
  --z-toast: 500;

  /* ── Motion ── */
  --duration-fast: 120ms;
  --duration-base: 180ms;
  --duration-slow: 240ms;
  --ease-standard: cubic-bezier(0.2, 0, 0, 1);
  --ease-exit: cubic-bezier(0.4, 0, 1, 1);
}

/* ── Dark theme: shared declarations ── */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --color-bg: var(--color-neutral-950);
    --color-surface: var(--color-brand-primary);
    --color-surface-muted: var(--color-neutral-800);
    --color-text: var(--color-neutral-50);
    --color-text-muted: var(--color-neutral-300);
    --color-text-subtle: var(--color-neutral-300);
    --color-link: #82B5AA;
    --color-border: var(--color-neutral-700);
    --color-border-strong: #8A9A97;

    --color-action: var(--color-brand-mint-light);
    --color-action-hover: #82B5AA;
    --color-action-pressed: #5C978B;
    --color-on-action: var(--color-neutral-950);
    --color-focus-ring: var(--color-brand-mint-light);
    --color-selected: var(--color-brand-mint-light);

    --color-success: #81C784;
    --color-info: #90CAF9;
    --color-error: #EF9A9A;

    --shadow-sm: 0 1px 2px rgb(0 0 0 / 30%);
    --shadow-md: 0 4px 12px rgb(0 0 0 / 35%);
    --shadow-lg: 0 12px 32px rgb(0 0 0 / 45%);
  }
}

:root[data-theme="dark"] {
  --color-bg: var(--color-neutral-950);
  --color-surface: var(--color-brand-primary);
  --color-surface-muted: var(--color-neutral-800);
  --color-text: var(--color-neutral-50);
  --color-text-muted: var(--color-neutral-300);
  --color-text-subtle: var(--color-neutral-300);
  --color-link: #82B5AA;
  --color-border: var(--color-neutral-700);
  --color-border-strong: #8A9A97;

  --color-action: var(--color-brand-mint-light);
  --color-action-hover: #82B5AA;
  --color-action-pressed: #5C978B;
  --color-on-action: var(--color-neutral-950);
  --color-focus-ring: var(--color-brand-mint-light);
  --color-selected: var(--color-brand-mint-light);

  --color-success: #81C784;
  --color-info: #90CAF9;
  --color-error: #EF9A9A;

  --shadow-sm: 0 1px 2px rgb(0 0 0 / 30%);
  --shadow-md: 0 4px 12px rgb(0 0 0 / 35%);
  --shadow-lg: 0 12px 32px rgb(0 0 0 / 45%);
}

/* ── Persian / RTL ── */
:root[lang="fa"],
:root[dir="rtl"] {
  --font-family: var(--font-family-fa);
  --line-height-body: 1.6;
  --line-height-relaxed: 1.7;
}

/* ── Reduced motion ── */
@media (prefers-reduced-motion: reduce) {
  :root {
    --duration-fast: 1ms;
    --duration-base: 1ms;
    --duration-slow: 1ms;
  }
}

/* ── Base ── */
body {
  background: var(--color-bg);
  color: var(--color-text);
  font-family: var(--font-family);
  font-size: var(--font-size-body);
  line-height: var(--line-height-body);
}

:focus-visible {
  outline: var(--focus-ring-width) solid var(--color-focus-ring);
  outline-offset: var(--focus-ring-offset);
}
```

> **Note on dark mode:** brand primitives keep their values in both themes. In v1.0, `--color-brand-primary` itself was redefined to mint in dark mode, which meant "brand primary" meant two different colors. Theme changes now happen only in semantic tokens (`--color-action`, `--color-surface`, …).

---

## 12. Usage do's and don'ts

### Do

- Use deep forest green as a recognizable anchor, not as the background of every element.
- Keep large content areas neutral to preserve readability.
- Use mint as a restrained accent for focus, selection, and highlights.
- Build components from semantic tokens so themes work automatically.
- Keep spacing, radii, and typography consistent across products.
- Validate contrast in real component states.

### Don't

- Use bright accent colors as large background surfaces.
- Use semantic colors as decorative brand colors.
- Put low-contrast gray text on green surfaces (`#7A8A87` on `#263B38` fails).
- Use mint (`#4E8B7F`) for body text or with white button labels.
- Put white text on warning yellow.
- Apply gradients to every card or button.
- Use color as the only indicator of state.
- Use `left`/`right` in CSS where a logical property exists.

---

## 13. Implementation checklist

- [ ] Define tokens centrally; components reference semantic tokens only, no one-off hex values.
- [ ] Implement light and dark themes, both via `prefers-color-scheme` and the `data-theme` override.
- [ ] Build buttons, inputs, cards, alerts, badges, navigation, and tables from shared tokens.
- [ ] Add hover, pressed, disabled, focus-visible, loading, and error states to every interactive component.
- [ ] Use `border-strong` for every input and control boundary.
- [ ] Verify WCAG contrast for every foreground/background pairing actually used (see section 9 table).
- [ ] Test responsive layouts at each breakpoint, keyboard navigation, RTL (with real Persian content), and reduced motion.
- [ ] Document any product-specific extension without changing the core brand tokens.

---

## 14. Changelog

### 1.1

**Preserved:** every v1.0 hex value, the brand palette, the type scale, spacing, radii, shadows, and the gradient.

**Added**
- `color.brand.mint-light` (`#6AA397`) and `color.neutral.600` (`#5F6E6B`) as documented primitives.
- Semantic token layer (`text-subtle`, `text-disabled`, `link`, `border-strong`, `action-*`, `on-action`, `focus-ring`, `selected`, `on-warning`).
- Dark-theme tints for success, info, and error.
- Line-height and weight tokens; tokens for display, label, and every type-scale step.
- Breakpoints, z-index scale, motion tokens, control-height tokens.
- RTL guidance (logical properties, icon mirroring, bidi isolation, Persian line-height).
- Ghost button, disabled and loading states, data-viz palette order, verified-contrast table.
- Automatic dark mode via `prefers-color-scheme`.

**Fixed**
- Dark-theme primary button: white on `#4E8B7F` (3.94:1, fails AA) → `#1B1F1E` on `#6AA397` (5.79:1).
- Dark-theme pressed state no longer equals the surface color.
- Dark-theme links and control borders lifted to stay above 4.5:1 and 3:1 on every dark surface.
- Placeholder/muted text: `#7A8A87` (3.61:1, fails) → `#5F6E6B` (5.34:1); `#7A8A87` reserved for icons, disabled, and control borders.
- Input borders: `#C7D0CD` (1.58:1, fails non-text contrast) → `#7A8A87` (3.61:1).
- Semantic colors on dark surfaces (≈ 3.3:1) → dark-theme tints (≥ 5:1).
- Warning text color rule made explicit (dark text only).
- Inconsistent button (40px) vs. touch target (44px) heights resolved into two sizes.
- Brand tokens no longer redefined inside the dark theme.
- Formatting: broken inline button specs and grid tables rewritten as standard Markdown.

---

**Design intent:** composed, thoughtful, quietly confident, and enduring. The interface should feel like a modern tool grounded in nature: focused enough for serious work, warm enough to be approachable.