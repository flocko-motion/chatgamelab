# Button Components

ChatGameLab uses **semantic button components** that encode design intent. Choose a button based on **what the action is**, not how you want it to look.

## Available Buttons

| Component | When to Use |
|-----------|-------------|
| `ActionButton` | Primary CTA - the main thing the user should do |
| `MenuButton` | Action lists/menus in sidebars or panels |
| `TextButton` | Secondary/subtle actions (View All, Cancel) |
| `DangerButton` | Destructive actions (Delete, Remove) |

## ActionButton

**Primary call-to-action button** for the main action on a page.

```tsx
import { ActionButton } from '@components/buttons';

// Hero CTA
<ActionButton onClick={handleLogin}>Get Started</ActionButton>

// Form submission
<ActionButton type="submit" loading={isSubmitting}>
  Submit
</ActionButton>

// With icon
<ActionButton leftSection={<IconRocket size={20} />}>
  Launch
</ActionButton>
```

**Use when:**
- Main action on a page (Login, Sign Up, Get Started)
- Hero section CTAs
- Form submission buttons

**Props:**
- `size`: 'sm' | 'md' | 'lg' (default: 'lg')
- `loading`: boolean
- `disabled`: boolean
- `fullWidth`: boolean
- `type`: 'button' | 'submit' | 'reset'
- `leftSection` / `rightSection`: ReactNode

## MenuButton

**Action button for menus and action lists** - full-width, left-aligned with icons.

```tsx
import { MenuButton } from '@components/buttons';

<Stack gap="sm">
  <MenuButton leftSection={<IconPlus size={16} />}>
    Create New Game
  </MenuButton>
  <MenuButton leftSection={<IconBuilding size={16} />}>
    Create Room
  </MenuButton>
</Stack>
```

**Use when:**
- Quick action panels/sidebars
- Action lists within cards
- Navigation-like buttons in secondary areas

**Props:**
- `leftSection` / `rightSection`: ReactNode
- `disabled`: boolean

## TextButton

**Subtle, link-like button** for secondary actions.

```tsx
import { TextButton } from '@components/buttons';

// Navigation
<TextButton onClick={handleViewAll}>View All</TextButton>

// Cancel action
<TextButton onClick={handleCancel}>Cancel</TextButton>
```

**Use when:**
- Secondary navigation ("View All", "See More")
- Cancel/dismiss actions
- Low-emphasis actions that shouldn't compete with primary CTAs

**Props:**
- `size`: 'xs' | 'sm' | 'md' (default: 'sm')
- `leftSection` / `rightSection`: ReactNode
- `disabled`: boolean

## DangerButton

**Button for destructive or irreversible actions.**

```tsx
import { DangerButton } from '@components/buttons';

// Filled (default)
<DangerButton onClick={handleDelete}>Delete Account</DangerButton>

// Outline variant
<DangerButton variant="outline" onClick={handleRetry}>
  Try Again
</DangerButton>
```

**Use when:**
- Delete/remove actions
- Destructive operations
- Error recovery actions

**Props:**
- `variant`: 'filled' | 'outline' (default: 'filled')
- `loading`: boolean
- `disabled`: boolean
- `fullWidth`: boolean
- `type`: 'button' | 'submit' | 'reset'

## Migration Guide

Replace direct Mantine `<Button>` usage with semantic components:

```tsx
// ❌ Before
<Button variant="filled" color="accent" size="lg">
  Get Started
</Button>

// ✅ After
<ActionButton>Get Started</ActionButton>
```

```tsx
// ❌ Before
<Button variant="subtle" size="sm">View All</Button>

// ✅ After
<TextButton>View All</TextButton>
```

```tsx
// ❌ Before
<Button variant="light" color="accent" fullWidth justify="start">
  Create Game
</Button>

// ✅ After
<MenuButton leftSection={<IconPlus />}>Create Game</MenuButton>
```

## File Structure

```
src/common/components/buttons/
├── index.ts          # Exports all button components
├── ActionButton.tsx  # Primary CTA
├── MenuButton.tsx    # Menu/list actions
├── TextButton.tsx    # Subtle/link actions
└── DangerButton.tsx  # Destructive actions
```
