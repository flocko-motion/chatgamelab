# Color System

ChatGameLab uses Mantine's theming system with custom color palettes defined in `src/config/colors.ts`.

## Brand Identity

- **Dark bluish gradient header** → trustworthy, techy, calm
- **Neon cyan/magenta logo** → playful, game-y, creative
- **Light theme content** → modern SaaS, readable, scalable

## Theme Configuration

The app enforces a **light color scheme** with `autoContrast` enabled for readable button text.

```tsx
// In AppProviders.tsx
<MantineProvider theme={mantineTheme} forceColorScheme="light">
```

## Color Palettes

All colors use Mantine's 10-shade format (index 0-9):

| Index | Purpose |
|-------|---------|
| 0-2 | Light shades (backgrounds, subtle states) |
| 3-4 | Medium-light (hover states) |
| **5-6** | **Main color** (default, filled buttons) |
| 7-8 | Dark shades (pressed states, borders) |
| 9 | Darkest (text on light backgrounds) |

### Primary Accent - Cyan (`accent`)

**Use for:** Primary buttons, active nav, toggles, focus rings, links

```tsx
<Button color="accent">Primary Action</Button>
<ThemeIcon color="accent">...</ThemeIcon>
<Title c="accent.9">Headline</Title>
```

### Secondary Accent - Magenta (`highlight`)

**Use for:** Notifications, badges, special actions (Create, New, Pro)

⚠️ **Don't overuse** - one accent per screen is ideal.

```tsx
<Badge color="highlight">New</Badge>
```

### Semantic Colors

| Color | Use for |
|-------|---------|
| `green` | Success states |
| `red` | Error states, danger actions |
| `orange` | Warning states |
| `blue` | Info states |

### Gray (Slate-based, slightly blue-tinted)

| Index | Usage |
|-------|-------|
| `gray.0` | Main app background |
| `gray.1` | Hover/selected backgrounds |
| `gray.2` | Borders, dividers |
| `gray.5` | Muted text |
| `gray.7` | Body text (default for `<Text>`) |
| `gray.9` | Title text (default for `<Title>`) |

## Usage Examples

### Color Props (Recommended)

```tsx
// Use color prop for component theming
<Button color="accent">Primary</Button>
<Button color="red" variant="filled">Delete</Button>
<ThemeIcon color="green" variant="light">...</ThemeIcon>
<Badge color="highlight">Pro</Badge>

// Use c prop for text colors
<Title c="accent.9">Feature Title</Title>
<Text c="gray.5">Muted description</Text>
<Text c="dimmed">Also muted (Mantine built-in)</Text>
```

### CSS Variables in Styles

```tsx
<Card style={{ borderTop: '3px solid var(--mantine-color-accent-5)' }}>

// Available patterns:
// --mantine-color-{colorName}-{0-9}
// --mantine-color-accent-5
// --mantine-color-gray-2
```

### Theme Access in Components

```tsx
import { useMantineTheme } from '@mantine/core';

function MyComponent() {
  const theme = useMantineTheme();
  
  return (
    <Box style={{
      // For header/dark elements
      background: theme.other.layout.headerGradient,
      // For semantic colors
      color: theme.other.colors.textBody,
    }}>
      ...
    </Box>
  );
}
```

## Header/Dark Theme Elements

The header uses a separate dark color system via `theme.other.layout`:

```tsx
const theme = useMantineTheme();

// Available properties:
theme.other.layout.headerGradient  // Header background
theme.other.layout.bgHover         // Hover state for header elements
theme.other.layout.borderLight     // Subtle borders on dark bg
theme.other.layout.panelBg         // Panel/dropdown background
```

## Design Principles

1. **"If everything is colorful, nothing is important"**
   - Use cyan for primary actions
   - Use magenta sparingly for attention
   - Everything else: calm, neutral, predictable

2. **Auto-contrast for accessibility**
   - Button text color is automatic
   - No need for manual `textColor` props

3. **Consistent text hierarchy**
   - Use `c="gray.9"` for title text
   - Use `c="gray.7"` for body text  
   - Use `c="gray.5"` or `c="dimmed"` for muted text
   - Header/dark elements inherit `color: white` from parent styles

## File Structure

```
src/config/
├── colors.ts         # Color palettes & semantic colors
└── mantineTheme.ts   # Mantine theme configuration
```
