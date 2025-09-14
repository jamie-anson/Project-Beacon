# Catppuccin Color Mapping Strategy
## Project Beacon Portal Migration

### Current → Catppuccin Class Mappings

#### Background Colors
- `bg-slate-50` → `bg-ctp-base` (main background)
- `bg-white` → `bg-ctp-surface0` (card/panel backgrounds)
- `bg-slate-100` → `bg-ctp-surface1` (hover states)
- `bg-beacon-50` → `bg-ctp-surface2` (active/selected states)

#### Text Colors
- `text-slate-900` → `text-ctp-text` (primary text)
- `text-slate-600` → `text-ctp-subtext1` (secondary text)
- `text-slate-500` → `text-ctp-subtext0` (muted text)
- `text-slate-400` → `text-ctp-overlay2` (placeholder text)
- `text-beacon-600` → `text-ctp-peach` (accent text/links)

#### Border Colors
- `border-slate-200` → `border-ctp-surface1` (subtle borders)
- `border-b` → `border-b border-ctp-surface1`

#### Status Colors
- `bg-green-100 text-green-700` → `bg-ctp-green/20 text-ctp-green`
- `bg-red-100 text-red-700` → `bg-ctp-red/20 text-ctp-red`
- `bg-green-500` → `bg-ctp-green`
- `bg-red-500` → `bg-ctp-red`

#### Interactive States
- `hover:text-slate-900` → `hover:text-ctp-text`
- `hover:bg-slate-50` → `hover:bg-ctp-surface0`

### Semantic Color Usage
- **Primary Accent**: `text-ctp-peach` (navigation, CTAs)
- **Success**: `text-ctp-green` (online status, success states)
- **Error**: `text-ctp-red` (offline status, errors)
- **Warning**: `text-ctp-yellow` (warnings, alerts)
- **Info**: `text-ctp-blue` (informational content)

### Custom Beacon Colors (Preserved)
- `beacon-50`: Light peach tint for subtle accents
- `beacon-400`: Catppuccin peach for primary brand color
- `beacon-600`: Catppuccin pink for secondary brand color
