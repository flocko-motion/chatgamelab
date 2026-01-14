# Color System

ChatGameLab uses a centralized color system defined in `src/config/colors.ts`. All colors flow through this file to the Mantine theme in `src/config/mantineTheme.ts`.

## Brand Identity

- **Dark bluish gradient header** → trustworthy, techy, calm
- **Neon cyan/magenta logo** → playful, game-y, creative
- **Light theme content** → modern SaaS, readable, scalable

## Color Roles

### Primary Accent - Cyan (`accent`)
```
#22E6F3 (colors.accent[500])
```

**Use for:**
- Primary buttons
- Active navigation items
- Toggles, sliders
- Focus rings
- Links

```tsx
<Button color="accent">Primary Action</Button>
<ThemeIcon color="accent">...</ThemeIcon>
<Anchor c="accent">Link</Anchor>
```

### Secondary Accent - Magenta (`highlight`)
```
#FF4D9D (colors.highlight[500])
```

**Use for:**
- Notifications
- Badges
- Special actions (Create, New, Pro, etc.)
- Attention-grabbing elements

⚠️ **Don't overuse** - one accent per screen is ideal.

```tsx
<Badge color="highlight">New</Badge>
<ThemeIcon color="highlight">...</ThemeIcon>
```

### Surface Colors

| Role | Color | Usage |
|------|-------|-------|
| Background Main | `#F6F8FB` | Main app background |
| Surface | `#FFFFFF` | Cards, modals, dropdowns |
| Hover | `#EEF2F7` | Hover/selected states |

### Typography Colors

| Role | Color | Usage |
|------|-------|-------|
| Title | `#0F172A` | Headlines (dark blue, not pure black) |
| Body | `#334155` | Body text |
| Muted | `#64748B` | Secondary/muted text |
| Inverse | `#E6F0FF` | Text on dark backgrounds (header) |

### Border & Icon Colors

| Role | Color | Usage |
|------|-------|-------|
| Border | `#E2E8F0` | Subtle borders, dividers |
| Icon Default | `#475569` | Default icon color |
| Icon Active | Cyan | Active icons |

## Usage in Components

### Mantine Color Props

Use the color name directly with Mantine components:

```tsx
// Primary accent (cyan)
<Button color="accent">...</Button>
<ThemeIcon color="accent">...</ThemeIcon>
<Badge color="accent">...</Badge>

// Secondary accent (magenta)
<Badge color="highlight">Pro</Badge>
<ThemeIcon color="highlight">...</ThemeIcon>

// Semantic colors
<Badge color="green">Success</Badge>
<Badge color="red">Error</Badge>
<Badge color="orange">Warning</Badge>
```

### Accessing Theme Colors in Styles

```tsx
import { useMantineTheme } from '@mantine/core';

function MyComponent() {
  const theme = useMantineTheme();
  
  return (
    <Box
      style={{
        borderColor: theme.colors.accent[6],
        color: theme.other.colors.textBody,
      }}
    >
      ...
    </Box>
  );
}
```

### CSS Variables

Available via `cssVariables` export:

```css
.my-class {
  color: var(--color-text-body);
  background: var(--color-bg-surface);
  border-color: var(--color-border);
}
```

### Semantic Colors Object

Access via `theme.other.colors` or import directly:

```tsx
import { semanticColors } from '@/config/colors';

// In styles
color: semanticColors.textTitle
background: semanticColors.bgMain
```

## Design Principles

1. **If everything is colorful, nothing is important**
   - Use cyan for actions
   - Use magenta for attention
   - Everything else: calm, neutral, predictable

2. **Keep the header + logo as the star**
   - Header stays with gradient
   - Use muted, neutral surfaces elsewhere
   - Reserve neon colors strictly for accents & actions

3. **Typography is boring (on purpose)**
   - Let color come from structure, not decoration
   - Dark blue titles instead of pure black for softer feel

## File Structure

```
src/config/
├── colors.ts         # Color definitions & exports
└── mantineTheme.ts   # Mantine theme configuration
```
