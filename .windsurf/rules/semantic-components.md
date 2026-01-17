# Semantic Component Guidelines

This document defines how to use intent-based wrapper components in ChatGameLab. We do NOT use Mantine components directly - instead we create semantic wrappers that encode design intent.

## Core Principle

> Choose components based on **WHAT the action/content is**, not **how you want it to look**.

## Button Components

Import from `@components/buttons`
| Component | Intent | When to Use |
|-----------|--------|-------------|
| `ActionButton` | Primary CTA | Main action on a page (Login, Submit, Get Started) |
| `MenuButton` | Menu/list action | Quick action panels, sidebars, action lists |
| `TextButton` | Subtle secondary | View All, Cancel, low-emphasis actions |
| `DangerButton` | Destructive action | Delete, Remove, error recovery |

### Examples

```tsx
// ❌ Wrong - deciding style ad-hoc
<Button variant="filled" color="accent" size="lg">Get Started</Button>

// ✅ Correct - selecting by intent
<ActionButton>Get Started</ActionButton>
```

```tsx
// ❌ Wrong
<Button variant="subtle" size="sm">View All</Button>

// ✅ Correct
<TextButton>View All</TextButton>
```

```tsx
// ❌ Wrong
<Button variant="light" color="accent" fullWidth justify="start">
  Create Game
</Button>

// ✅ Correct
<MenuButton leftSection={<IconPlus />}>Create Game</MenuButton>
```

## Typography Components

Import from `@components/typography`

### Headings

| Component | Intent | HTML | When to Use |
|-----------|--------|------|-------------|
| `PageTitle` | Main page heading | h1 | Top-level heading, one per page |
| `SectionTitle` | Section heading | h2 | Grouping content within a page |
| `CardTitle` | Card/panel heading | h3 | Card headers, panel titles |

### Text

| Component | Intent | When to Use |
|-----------|--------|-------------|
| `BodyText` | Primary content | Paragraphs, descriptions |
| `Label` | Form/metadata label | Form field labels, stat labels |
| `HelperText` | Muted secondary | Hints, descriptions, footer text |
| `ErrorText` | Error message | Validation errors, API errors |

### Examples

```tsx
// ❌ Wrong
<Title order={2} c="accent.9">Settings</Title>
<Text size="sm" c="dimmed">Enter your email</Text>

// ✅ Correct
<SectionTitle accent>Settings</SectionTitle>
<HelperText>Enter your email</HelperText>
```

## Color System

### DO use theme colors

```tsx
// CSS variables
color: 'var(--mantine-color-accent-5)'
background: 'var(--mantine-color-red-0)'
border: '1px solid var(--mantine-color-accent-2)'

// Mantine color props
c="accent.9"
c="gray.5"
color="accent"
```

### DON'T use hardcoded colors

```tsx
// ❌ Wrong
color: '#29D0DE'
background: 'rgba(139, 92, 246, 0.2)'
c="#fef2f2"
```

### Accent Color Palette

The primary accent is cyan (`#29D0DE` at index 5):

| Index | Usage |
|-------|-------|
| 0-2 | Light backgrounds, subtle states |
| 3-4 | Hover states |
| **5** | **Main accent (buttons, primary actions)** |
| 6-7 | Hover/pressed states |
| 8-9 | Text on light backgrounds |

## Inline Styles

### Acceptable inline styles

- Layout (flex, grid, positioning)
- Transitions/animations
- Dynamic values that can't be CSS variables

### Move to CSS modules

- Complex hover/focus states
- Multiple related styles
- Reusable patterns

```tsx
// ✅ OK - simple layout
style={{ display: 'flex', alignItems: 'center' }}

// ❌ Move to CSS module - complex styling
style={{
  background: 'var(--mantine-color-accent-0)',
  border: '1px solid var(--mantine-color-accent-2)',
  borderRadius: 'var(--mantine-radius-md)',
  '&:hover': { ... }
}}
```

## File Structure

```
src/common/components/
├── buttons/
│   ├── index.ts           # Exports all buttons
│   ├── ActionButton.tsx   # Primary CTA
│   ├── MenuButton.tsx     # Menu/list actions
│   ├── TextButton.tsx     # Subtle/link actions
│   └── DangerButton.tsx   # Destructive actions
├── typography/
│   ├── index.ts           # Exports all typography
│   ├── PageTitle.tsx      # h1 - main page heading
│   ├── SectionTitle.tsx   # h2 - section heading
│   ├── CardTitle.tsx      # h3 - card heading
│   ├── BodyText.tsx       # Primary content
│   ├── Label.tsx          # Form/metadata labels
│   ├── HelperText.tsx     # Muted secondary text
│   └── ErrorText.tsx      # Error messages
└── index.ts               # Re-exports buttons + typography
```

## Import Patterns

```tsx
// Buttons
import { ActionButton, MenuButton, TextButton, DangerButton } from '@components/buttons';

// Typography
import { PageTitle, SectionTitle, CardTitle, BodyText, Label, HelperText, ErrorText } from '@components/typography';

// Or from main index
import { ActionButton, SectionTitle, BodyText } from '@components';
```

## Component Props Reference

### ActionButton

```tsx
interface ActionButtonProps {
  children: ReactNode;
  onClick?: () => void;
  leftSection?: ReactNode;
  rightSection?: ReactNode;
  loading?: boolean;
  disabled?: boolean;
  type?: 'button' | 'submit' | 'reset';
  fullWidth?: boolean;
  size?: 'sm' | 'md' | 'lg'; // default: 'lg'
}
```

### MenuButton

```tsx
interface MenuButtonProps {
  children: ReactNode;
  onClick?: () => void;
  leftSection?: ReactNode;
  rightSection?: ReactNode;
  disabled?: boolean;
}
```

### TextButton

```tsx
interface TextButtonProps {
  children: ReactNode;
  onClick?: () => void;
  leftSection?: ReactNode;
  rightSection?: ReactNode;
  disabled?: boolean;
  size?: 'xs' | 'sm' | 'md'; // default: 'sm'
}
```

### DangerButton

```tsx
interface DangerButtonProps {
  children: ReactNode;
  onClick?: () => void;
  leftSection?: ReactNode;
  rightSection?: ReactNode;
  loading?: boolean;
  disabled?: boolean;
  type?: 'button' | 'submit' | 'reset';
  variant?: 'filled' | 'outline'; // default: 'filled'
  fullWidth?: boolean;
}
```

### Typography Components

```tsx
// All accept children: ReactNode

// Titles - optional accent prop for cyan color
<PageTitle>...</PageTitle>
<SectionTitle accent>...</SectionTitle>
<CardTitle accent>...</CardTitle>

// Text
<BodyText size="sm" | "md" | "lg" | "xl">...</BodyText>
<Label uppercase>...</Label>
<HelperText>...</HelperText>
<ErrorText>...</ErrorText>
```

## Review Checklist

When reviewing code, check:

- [ ] Uses semantic button components, not `<Button>` with manual props
- [ ] Uses semantic typography, not `<Title>` or `<Text>` with manual styling
- [ ] No hardcoded hex colors - uses theme CSS variables
- [ ] Complex inline styles moved to CSS modules
- [ ] Imports from `@components/buttons` or `@components/typography`
