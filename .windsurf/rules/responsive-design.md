# Rules for AI Agents:  React SPA with Mobile & Desktop Support (Mantine v8)

## 0.  Scope & Goals

You are helping build a **React Single Page Application** that must work well on **mobile and desktop** and uses **Mantine v8** as the UI framework.

Your primary goals: 

1. Ensure **responsive, mobile-first UI** using Mantine primitives. 
2. Keep **navigation and interaction patterns** consistent and touch-friendly.
3. Adapt **dynamic content** (tables, lists, complex layouts) to different screen sizes.
4. Promote **shared, reusable patterns** instead of ad-hoc responsive hacks.
5. Maintain **good performance**, especially on mobile. 
6. Ensure **accessibility** compliance (WCAG 2.1 AA minimum).
7. For the top header with centered logo and left/right items, choose **mobile switch breakpoint based on real content width and 20-language support** (see section 2.3).

---
Agents MUST use Mantine v8 APIs.  If you encounter deprecated patterns from earlier versions, update them to v8 equivalents. 

---

## 1. Layout & Responsiveness

### 1.1 General Principles

- Always assume **mobile-first design**: 
  - Base styles should work on small screens. 
  - Use breakpoints to enhance layouts on larger screens.
- Prefer **flexible layouts** (`flex`, `%`, `maxWidth`) instead of fixed `px` dimensions.
- Use Mantine's **responsive props** and **breakpoints** instead of raw CSS where possible. 
- Default Mantine breakpoints (reference only—always use `theme.breakpoints`):
  - `xs`: 576px
  - `sm`: 768px
  - `md`: 992px
  - `lg`: 1200px
  - `xl`: 1408px

### 1.2 Required Mantine Primitives

When designing or refactoring layout, you MUST prefer: 

- `AppShell` for overall shell:  header, navbar, main, footer, aside. 
- `Grid`, `SimpleGrid`, `Flex`, `Stack`, `Group` for internal layout.
- `Container` for constraining content width with consistent padding.
- `useMantineTheme` and `theme. breakpoints` for breakpoint-aware behavior. 
- `visibleFrom` / `hiddenFrom` props for conditional rendering based on viewport.

Example (canonical pattern for layout):

```tsx
import { AppShell, Burger, Group, Box } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';

function Layout({ children }:  { children: React. ReactNode }) {
  const [opened, { toggle }] = useDisclosure();

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{
        width:  { base: 260, md: 300 },
        breakpoint: 'sm',
        collapsed: { mobile: ! opened, desktop: false },
      }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md">
          <Burger
            opened={opened}
            onClick={toggle}
            hiddenFrom="sm"
            size="sm"
            aria-label="Toggle navigation"
          />
          <Box>Logo</Box>
        </Group>
      </AppShell. Header>

      <AppShell. Navbar p="md">
        {/* Navigation items */}
      </AppShell. Navbar>

      <AppShell.Main>
        {children}
      </AppShell.Main>
    </AppShell>
  );
}
```

### 1.3 Layout Rules

Agents MUST: 

- Avoid custom media queries unless Mantine props cannot express a requirement.
- Avoid layouts that rely on precise pixel-perfect positioning that can break on resize.
- Use `Container` with `size` prop to limit content width on large screens.
- Prefer responsive prop objects (e.g., `{{ base: 'sm', md: 'lg' }}`) over conditional rendering for simple style changes. 

Agents MUST NOT: 

- Hard-code pixel breakpoint values in CSS or inline styles.
- Use `position: fixed` for navigation elements without considering iOS Safari scroll behavior.
- Create layouts that require horizontal scrolling of the main content area on mobile.

### 1.4 Container Queries (Modern CSS)

Mantine v8 supports **CSS container queries**, which allow styles to adapt based on a **container's size** rather than the viewport. This is ideal for reusable components that may appear in different layout contexts (sidebars, modals, cards, etc.).

**When to use container queries vs media queries:**

| Use Case | Approach |
|----------|----------|
| Page layout, navigation | Media queries (viewport-based) |
| Reusable component that adapts to its container | Container queries |
| Card in a grid that may be wide or narrow | Container queries |
| Sidebar content that may be collapsed | Container queries |

**Implementation with CSS Modules:**

```css
/* styles.module.css */
.container {
  container-type: inline-size;
}

.content {
  display: flex;
  flex-direction: column;
  gap: var(--mantine-spacing-sm);
  
  /* When container is at least 500px wide, switch to row layout */
  @container (min-width: 500px) {
    flex-direction: row;
    gap: var(--mantine-spacing-md);
  }
  
  /* When container is at least 800px wide, add more space */
  @container (min-width: 800px) {
    gap: var(--mantine-spacing-xl);
  }
}
```

```tsx
import classes from './styles.module.css';

function AdaptiveCard({ children }: { children: React.ReactNode }) {
  return (
    <div className={classes.container}>
      <div className={classes.content}>
        {children}
      </div>
    </div>
  );
}
```

**Important notes:**

- Container queries are supported in [all modern browsers](https://caniuse.com/css-container-queries).
- You can use `rem` and `em` functions from `postcss-preset-mantine` in container queries.
- CSS variables **do not work** inside container query conditions.
- If you rely on [rem scaling](https://mantine.dev/styles/rem/), define container query breakpoints in `px` units.

**Example: Responsive card that adapts to container width:**

```css
/* ResponsiveCard.module.css */
.wrapper {
  container-type: inline-size;
}

.card {
  padding: var(--mantine-spacing-sm);
}

.cardContent {
  display: flex;
  flex-direction: column;
  gap: var(--mantine-spacing-xs);
  
  @container (min-width: 400px) {
    flex-direction: row;
    align-items: center;
    gap: var(--mantine-spacing-md);
  }
}

.cardImage {
  width: 100%;
  aspect-ratio: 16 / 9;
  
  @container (min-width: 400px) {
    width: 120px;
    aspect-ratio: 1;
  }
}

.cardTitle {
  font-size: var(--mantine-font-size-sm);
  
  @container (min-width: 400px) {
    font-size: var(--mantine-font-size-md);
  }
}
```

Agents SHOULD:

- Use container queries for **reusable components** that appear in multiple layout contexts.
- Prefer media queries for **page-level layout** decisions.
- Set `container-type: inline-size` on the wrapper element.
- Test components in different container widths (sidebar, main content, modal).

Agents MUST NOT:

- Use CSS variables inside `@container` conditions (they won't work).
- Rely on container queries for overall page navigation layout (use media queries instead).

---

## 2. Navigation & Interaction Patterns

### 2.1 Single Route Structure, Adaptive Shell

- There MUST be **one route structure** (React Router or similar) that is reused for both mobile and desktop. 
- The **navigation presentation** (sidebar, header, drawer) CAN change by breakpoint, but the **route paths** MUST remain consistent. 

### 2.2 Desktop vs Mobile Patterns

On **desktop**: 

- Use `AppShell.Navbar` as a persistent sidebar for primary navigation.
- Header can contain logo, search, user menu, and secondary actions. 

On **mobile**: 

- Replace sidebar with: 
  - `Burger` + collapsible `AppShell. Navbar`, or
  - `Burger` + `Drawer` for navigation, or
  - A bottom navigation bar for a small number of primary sections (≤5 items).
- Make hit areas large (**minimum 44×44px** touch target).
- Do NOT rely on hover for any critical interaction. 

Example (canonical responsive shell):

```tsx
import { AppShell, Burger, Group, Box, NavLink, ScrollArea } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';

const navItems = [
  { label: 'Dashboard', href: '/' },
  { label: 'Projects', href: '/projects' },
  { label: 'Settings', href: '/settings' },
];

function ResponsiveShell({ children }:  { children: React. ReactNode }) {
  const [opened, { toggle, close }] = useDisclosure();

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{
        width: 280,
        breakpoint: 'md',
        collapsed:  { mobile: !opened, desktop: false },
      }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group gap="sm">
            <Burger
              opened={opened}
              onClick={toggle}
              hiddenFrom="md"
              size="sm"
              aria-label="Toggle navigation"
            />
            <Box component="span" fw={700}>Logo</Box>
          </Group>

          {/* Desktop-only header actions */}
          <Group gap="sm" visibleFrom="md">
            {/* Search, notifications, user menu */}
          </Group>
        </Group>
      </AppShell. Header>

      <AppShell. Navbar p="md">
        <AppShell.Section grow component={ScrollArea}>
          {navItems.map((item) => (
            <NavLink
              key={item.href}
              href={item.href}
              label={item.label}
              onClick={close} // Close mobile nav on selection
            />
          ))}
        </AppShell.Section>
      </AppShell. Navbar>

      <AppShell. Main>
        {children}
      </AppShell.Main>
    </AppShell>
  );
}
```

Agents MUST: 

- Keep navigation items **semantically the same** between mobile and desktop. 
- Only change **presentation**, not **routing logic**.
- Close mobile navigation when a link is activated.
- Avoid building separate "mobile routes" unless explicitly required.

### 2.3 Header with Centered Logo & Left/Right Items (Multi-language)

**Context:**
The header has a **centered logo**, with items **to the left** and **to the right**. The app supports **20+ languages**, so label lengths vary significantly.

#### 2.3.1 Make the desktop header intrinsically flexible

Agents MUST design the desktop header so it can survive moderate width changes *before* fully switching to mobile:

- Use `Group` with `wrap="nowrap"` to keep a single row. 
- Use `visibleFrom` / `hiddenFrom` to hide less important labels on smaller widths (e.g., show only icons or shorter labels).
- Prefer icons or abbreviated labels for controls that appear in every language. 
- Use `flex` properties to allow sections to shrink gracefully. 

Example flexible desktop header:

```tsx
import { Group, Box, ActionIcon, Text, Tooltip } from '@mantine/core';
import { IconLanguage, IconUser } from '@tabler/icons-react';

function DesktopHeader() {
  return (
    <Group h={60} px="md" justify="space-between" wrap="nowrap">
      {/* Left section */}
      <Group gap="md" wrap="nowrap" style={{ flex: 1 }}>
        <Text component="a" href="/products" visibleFrom="sm">Products</Text>
        <Text component="a" href="/pricing" visibleFrom="md">Pricing</Text>
        <Text component="a" href="/docs" visibleFrom="lg">Documentation</Text>
      </Group>

      {/* Center logo - fixed width to maintain centering */}
      <Box style={{ flexShrink: 0 }}>
        <img src="/logo.svg" alt="Company Logo" height={32} />
      </Box>

      {/* Right section */}
      <Group gap="sm" wrap="nowrap" justify="flex-end" style={{ flex: 1 }}>
        <Tooltip label="Change language">
          <ActionIcon variant="subtle" aria-label="Change language">
            <IconLanguage size={20} />
          </ActionIcon>
        </Tooltip>
        <Text component="a" href="/login" visibleFrom="md">Login</Text>
        <ActionIcon variant="filled" radius="xl" aria-label="Open account menu">
          <IconUser size={18} />
        </ActionIcon>
      </Group>
    </Group>
  );
}
```

Rules: 

- On **narrower desktop widths**, some labels may disappear (icons remain).
- Use the **longest language translation** as the reference when testing.
- Left and right sections should have equal `flex` values to keep logo centered. 

#### 2.3.2 How to choose when to switch to "mobile view"

The switch to a mobile header (burger + drawer) MUST NOT be a hard-coded number like "768px" without context. It MUST be based on **when the header actually starts breaking** with your real labels.

**Process for choosing the breakpoint:**

1. **Implement the full desktop header** with: 
   - All left-section items. 
   - Centered logo.
   - Right-section items (e.g., language selector, login, user menu).
   - **Longest translations** for all labels (e.g., German, Finnish, or other verbose languages).

2. **Test in the browser's responsive mode**:
   - Gradually shrink the viewport width.
   - Observe when: 
     - Labels start wrapping into two lines.
     - Left/right groups overlap the logo.
     - Elements become too cramped or touch targets too small (<44px).

3. **Identify the "breaking width"** (the smallest width where the header is still clean & readable).

4. **Map that width to the nearest Mantine breakpoint**:
   | Breaking Width | Use Breakpoint |
   |----------------|----------------|
   | ~500-576px | `xs` |
   | ~700-768px | `sm` |
   | ~900-992px | `md` |

5. Use that breakpoint as the **mobile switch breakpoint** for: 
   - Showing a **Burger** instead of full nav.
   - Moving all navigation into a collapsible navbar or **Drawer**.
   - Hiding desktop-only header layouts.

**Rule of thumb:**

- If the full header (with longest translations) looks good down to ~768px: 
  → Use `breakpoint="sm"` as the switch to mobile header.
- If you have **many header items** or particularly long labels, and layout breaks earlier:
  → Use `breakpoint="md"`.

#### 2.3.3 Implementation pattern for the header switch

Agents MUST follow this pattern: 

- **Above the chosen breakpoint (desktop)**:
  - Show full header:  centered logo, left items, right items. 
- **Below the chosen breakpoint (mobile)**:
  - Show a simplified header:  logo + burger. 
  - Put all nav items and language selector into a collapsible `AppShell.Navbar` or `Drawer`.

Example: 

```tsx
import {
  AppShell,
  Burger,
  Group,
  Box,
  Text,
  ActionIcon,
  NavLink,
  Tooltip,
  Divider,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconLanguage, IconLogin } from '@tabler/icons-react';

function HeaderShell({ children }: { children: React.ReactNode }) {
  const [opened, { toggle, close }] = useDisclosure();
  
  // Choose 'md' or 'sm' based on real content testing (see 2.3.2)
  const mobileBreakpoint = 'md';

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{
        width: 280,
        breakpoint: mobileBreakpoint,
        collapsed: { mobile:  !opened, desktop:  true }, // Hidden on desktop, toggle on mobile
      }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between" wrap="nowrap">
          {/* Mobile:  burger + logo */}
          <Group gap="sm" wrap="nowrap">
            <Burger
              opened={opened}
              onClick={toggle}
              hiddenFrom={mobileBreakpoint}
              size="sm"
              aria-label="Toggle navigation menu"
            />
            <Box component="span" fw={700}>Logo</Box>
          </Group>

          {/* Desktop: left nav items */}
          <Group gap="lg" visibleFrom={mobileBreakpoint} style={{ flex: 1 }} justify="center">
            <Text component="a" href="/products">Products</Text>
            <Text component="a" href="/pricing">Pricing</Text>
            <Text component="a" href="/docs">Docs</Text>
          </Group>

          {/* Desktop:  right items */}
          <Group gap="sm" visibleFrom={mobileBreakpoint} wrap="nowrap">
            <Tooltip label="Change language">
              <ActionIcon variant="subtle" aria-label="Change language">
                <IconLanguage size={20} />
              </ActionIcon>
            </Tooltip>
            <Text component="a" href="/login">Login</Text>
          </Group>
        </Group>
      </AppShell. Header>

      {/* Mobile navigation */}
      <AppShell. Navbar p="md">
        <NavLink href="/products" label="Products" onClick={close} />
        <NavLink href="/pricing" label="Pricing" onClick={close} />
        <NavLink href="/docs" label="Docs" onClick={close} />
        <Divider my="sm" />
        <NavLink
          href="#"
          label="Change language"
          leftSection={<IconLanguage size={18} />}
          onClick={close}
        />
        <NavLink
          href="/login"
          label="Login"
          leftSection={<IconLogin size={18} />}
          onClick={close}
        />
      </AppShell.Navbar>

      <AppShell.Main>{children}</AppShell.Main>
    </AppShell>
  );
}
```

**Canonical rule for AI agents:**

> "Always choose the header's mobile switch breakpoint (`breakpoint` on `AppShell.navbar` and `visibleFrom`/`hiddenFrom` on header elements) as the nearest Mantine breakpoint **at or just above** the width where the full desktop header with the **longest language labels** starts wrapping or overlapping.  Typically, this will be `sm` or `md`. Never hard-code a number; always tie the breakpoint choice to observed header behavior with real translations."

---

## 3. Dynamic Content Adaptation

Dynamic content includes:  cards, lists, tables, dashboards, complex forms, etc.  Agents MUST ensure these adapt well across screen sizes.

### 3.1 Lists & Grids

For card-like item collections: 

- Use `SimpleGrid` or `Grid` with responsive `cols` and `spacing`.
- On small screens, **collapse to 1 column**. 
- Limit text (e.g., via `lineClamp`) to avoid mismatched card heights on narrow screens. 

Example:

```tsx
import { SimpleGrid, Card, Text, Group, Badge } from '@mantine/core';

interface Item {
  id:  string;
  title: string;
  description: string;
  status: string;
}

function ItemGrid({ items }: { items: Item[] }) {
  return (
    <SimpleGrid
      cols={{ base: 1, sm: 2, md: 3, lg: 4 }}
      spacing={{ base: 'sm', md: 'lg' }}
    >
      {items.map((item) => (
        <Card key={item.id} shadow="sm" padding="lg" withBorder>
          <Group justify="space-between" mb="xs">
            <Text fw={500} lineClamp={1}>{item.title}</Text>
            <Badge size="sm">{item.status}</Badge>
          </Group>
          <Text size="sm" c="dimmed" lineClamp={2}>
            {item.description}
          </Text>
        </Card>
      ))}
    </SimpleGrid>
  );
}
```

Agents SHOULD:

- Prefer `cols={{ base: 1, sm: 2, ...  }}` instead of hard-coded widths. 
- Use `visibleFrom` / `hiddenFrom` for significantly different card layouts (e.g., horizontal vs vertical).
- Use `lineClamp` to prevent text overflow from breaking grid alignment.

### 3.2 Tables

Tables are problematic on mobile.  Agents MUST choose one of these patterns: 

**Pattern A (Preferred for complex/detailed data):**
Use **table on desktop**, but **card/stack layout on mobile**.

```tsx
import {
  Table,
  Card,
  Stack,
  Group,
  Text,
  Badge,
} from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';

interface Row {
  id:  string;
  name: string;
  status: string;
  description: string;
  date: string;
}

function ResponsiveTable({ rows }: { rows: Row[] }) {
  const isMobile = useMediaQuery('(max-width:  48em)'); // ~768px, matches 'sm'

  if (isMobile) {
    return (
      <Stack gap="sm">
        {rows. map((row) => (
          <Card key={row.id} withBorder padding="md">
            <Group justify="space-between" mb="xs">
              <Text fw={500}>{row.name}</Text>
              <Badge size="sm" variant="light">{row.status}</Badge>
            </Group>
            <Text size="sm" c="dimmed" mb="xs">
              {row.description}
            </Text>
            <Text size="xs" c="dimmed">{row.date}</Text>
          </Card>
        ))}
      </Stack>
    );
  }

  return (
    <Table striped highlightOnHover>
      <Table. Thead>
        <Table.Tr>
          <Table. Th>Name</Table.Th>
          <Table. Th>Status</Table.Th>
          <Table. Th>Description</Table.Th>
          <Table. Th>Date</Table.Th>
        </Table. Tr>
      </Table.Thead>
      <Table. Tbody>
        {rows.map((row) => (
          <Table.Tr key={row.id}>
            <Table.Td>{row.name}</Table. Td>
            <Table.Td><Badge size="sm">{row.status}</Badge></Table. Td>
            <Table.Td>{row.description}</Table.Td>
            <Table. Td>{row.date}</Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  );
}
```

**Pattern B (For wide but simple data):**
Keep table but wrap in a horizontally scrollable container. 

```tsx
import { Table, ScrollArea } from '@mantine/core';

function ScrollableTable({ rows }: { rows: Row[] }) {
  return (
    <ScrollArea>
      <Table miw={700} striped highlightOnHover>
        <Table. Thead>
          <Table.Tr>
            <Table.Th>Name</Table.Th>
            <Table.Th>Status</Table.Th>
            <Table.Th>Description</Table.Th>
            <Table.Th>Date</Table.Th>
            <Table.Th>Actions</Table. Th>
          </Table.Tr>
        </Table.Thead>
        <Table. Tbody>
          {/* rows */}
        </Table.Tbody>
      </Table>
    </ScrollArea>
  );
}
```

Agents MUST:

- Pick **Pattern A** for complex, many-column, or rich-row data.
- Only use **Pattern B** if the table must truly remain tabular and is still readable when scrolled. 
- Add visual indication that horizontal scroll is available (e. g., fade edge or scroll hint).

### 3.3 Forms

For forms, agents MUST:

- Use **vertical stacking on mobile**. 
- Avoid side-by-side fields on small screens unless extremely simple (e.g., city/postcode).
- Group related fields with clear visual hierarchy.
- For long/complex forms: 
  - Consider `Stepper` for wizard patterns.
  - Use `Fieldset` or `Paper` for grouping with headings. 

Implementation rules:

- Use `Stack` as the default form layout.
- Use `Grid`/`SimpleGrid` sparingly and ensure `cols` → `1` on small screens.
- Use Mantine form components (`TextInput`, `NumberInput`, `Select`, `DatePickerInput`, etc.) for accessibility and touch-friendliness.
- Ensure all inputs have visible labels (not just placeholders).

Example: 

```tsx
import { Stack, TextInput, Select, Group, Button, SimpleGrid } from '@mantine/core';
import { useForm } from '@mantine/form';

function ContactForm() {
  const form = useForm({
    initialValues: {
      firstName: '',
      lastName: '',
      email: '',
      country: '',
    },
  });

  return (
    <form onSubmit={form. onSubmit((values) => console.log(values))}>
      <Stack gap="md">
        {/* Side-by-side on larger screens, stacked on mobile */}
        <SimpleGrid cols={{ base:  1, sm:  2 }} spacing="md">
          <TextInput
            label="First name"
            placeholder="John"
            required
            {...form. getInputProps('firstName')}
          />
          <TextInput
            label="Last name"
            placeholder="Doe"
            required
            {... form.getInputProps('lastName')}
          />
        </SimpleGrid>
        
        <TextInput
          label="Email"
          placeholder="john@example. com"
          type="email"
          required
          {...form.getInputProps('email')}
        />
        
        <Select
          label="Country"
          placeholder="Select your country"
          data={['United States', 'United Kingdom', 'Germany', 'France']}
          searchable
          {...form.getInputProps('country')}
        />
        
        <Group justify="flex-end" mt="md">
          <Button type="submit">Submit</Button>
        </Group>
      </Stack>
    </form>
  );
}
```

---

## 4. Accessibility Requirements

Agents MUST ensure all UI components meet **WCAG 2.1 Level AA** requirements. 

### 4.1 Core Requirements

| Requirement | Standard | Implementation |
|-------------|----------|----------------|
| Color contrast | 4.5:1 (text), 3:1 (UI) | Use Mantine's built-in color palette; test with contrast checker |
| Touch targets | Minimum 44×44px | Use appropriate padding/sizing on interactive elements |
| Focus visibility | Visible focus ring | Use Mantine's default focus styles; never remove outlines |
| Screen reader support | Meaningful labels | Use `aria-label`, visible labels, or `sr-only` text |

### 4.2 Navigation Accessibility

```tsx
// ✅ Good:  Burger has accessible label
<Burger
  opened={opened}
  onClick={toggle}
  aria-label={opened ? 'Close navigation' : 'Open navigation'}
/>

// ✅ Good: Icon buttons have labels
<ActionIcon aria-label="Search">
  <IconSearch size={18} />
</ActionIcon>

// ❌ Bad: No accessible name
<ActionIcon>
  <IconSearch size={18} />
</ActionIcon>
```

### 4.3 Focus Management

- When opening a modal or drawer, focus MUST move to the first focusable element inside.
- When closing, focus MUST return to the trigger element.
- Mobile navigation MUST trap focus when open. 

Mantine's `Modal` and `Drawer` components handle this automatically.  Agents MUST NOT override this behavior.

### 4.4 Motion and Animation

```tsx
// Respect user's reduced motion preference
import { useReducedMotion } from '@mantine/hooks';

function AnimatedComponent() {
  const reduceMotion = useReducedMotion();
  
  return (
    <Box
      style={{
        transition: reduceMotion ? 'none' : 'transform 200ms ease',
      }}
    >
      {/* content */}
    </Box>
  );
}
```

### 4.5 Form Accessibility

- All inputs MUST have associated labels (use `label` prop, not just `placeholder`).
- Error messages MUST be programmatically associated with inputs (Mantine handles this via `error` prop).
- Required fields MUST be indicated (use `required` prop or `withAsterisk`).

```tsx
// ✅ Good:  Proper labeling and error handling
<TextInput
  label="Email address"
  placeholder="you@example.com"
  required
  error={form.errors.email}
  {...form.getInputProps('email')}
/>

// ❌ Bad:  Placeholder-only, no label
<TextInput
  placeholder="Email address"
  {... form.getInputProps('email')}
/>
```

---

## 5. Shared Components & Design Tokens

### 5.1 Centralized Breakpoints & Theme

Agents MUST: 

- Use `theme.breakpoints` instead of ad-hoc hard-coded pixel values.
- Access breakpoints via `useMantineTheme` for dynamic usage.
- Use CSS-in-JS or Mantine's style props for responsive styles. 

```tsx
import { useMantineTheme } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';

function useBreakpoints() {
  const theme = useMantineTheme();
  
  return {
    isMobile: useMediaQuery(`(max-width:  ${theme.breakpoints.sm})`),
    isTablet: useMediaQuery(`(max-width: ${theme.breakpoints.md})`),
    isDesktop: useMediaQuery(`(min-width: ${theme.breakpoints.md})`),
  };
}
```

### 5.2 Reusable Layout Components

Agents SHOULD introduce small, composable layout helpers instead of repeating responsive logic everywhere.

```tsx
import { Box, Flex, Title, Group } from '@mantine/core';
import type { ReactNode } from 'react';

interface PageHeaderProps {
  title: string;
  actions?: ReactNode;
}

export function PageHeader({ title, actions }: PageHeaderProps) {
  return (
    <Flex
      justify="space-between"
      align={{ base: 'stretch', sm: 'center' }}
      direction={{ base: 'column', sm: 'row' }}
      gap="md"
      mb="xl"
    >
      <Title order={1} size="h2">{title}</Title>
      {actions && (
        <Group gap="sm" justify={{ base: 'stretch', sm: 'flex-end' }}>
          {actions}
        </Group>
      )}
    </Flex>
  );
}

interface TwoColumnProps {
  main: ReactNode;
  sidebar: ReactNode;
  sidebarPosition?: 'left' | 'right';
}

export function TwoColumn({ main, sidebar, sidebarPosition = 'right' }: TwoColumnProps) {
  const mainContent = <Box style={{ flex: 2 }}>{main}</Box>;
  const sidebarContent = <Box style={{ flex: 1 }}>{sidebar}</Box>;
  
  return (
    <Flex
      direction={{ base: 'column', md: 'row' }}
      gap="xl"
      align="flex-start"
    >
      {sidebarPosition === 'left' ?  (
        <>
          <Box visibleFrom="md">{sidebarContent}</Box>
          {mainContent}
          <Box hiddenFrom="md">{sidebarContent}</Box>
        </>
      ) : (
        <>
          {mainContent}
          {sidebarContent}
        </>
      )}
    </Flex>
  );
}
```

Agents SHOULD: 

- Reuse such components (`PageHeader`, `TwoColumn`, `CardGrid`, `ResponsiveTable`) across pages.
- Encode breakpoint-specific behavior inside these shared components instead of per-page custom code.
- Document props with TypeScript interfaces. 

---

## 6. Detecting Screen Size

Agents MUST use Mantine's hooks for viewport detection: 

### 6.1 useMediaQuery Hook

```tsx
import { useMantineTheme } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';

// Canonical helper for mobile detection
export function useIsMobile() {
  const theme = useMantineTheme();
  // Use 'em' units for better accessibility (respects user font size preferences)
  return useMediaQuery(`(max-width: ${theme.breakpoints.sm})`);
}

// More granular breakpoint detection
export function useBreakpoint() {
  const theme = useMantineTheme();
  
  const isXs = useMediaQuery(`(max-width: ${theme.breakpoints.xs})`);
  const isSm = useMediaQuery(`(max-width: ${theme.breakpoints.sm})`);
  const isMd = useMediaQuery(`(max-width: ${theme.breakpoints.md})`);
  const isLg = useMediaQuery(`(max-width: ${theme.breakpoints.lg})`);
  
  if (isXs) return 'xs';
  if (isSm) return 'sm';
  if (isMd) return 'md';
  if (isLg) return 'lg';
  return 'xl';
}
```

### 6.2 useMatches Hook (Multiple Breakpoints)

The `useMatches` hook from `@mantine/core` is useful when you need to match multiple breakpoints and return different values. It uses the same logic as `useMediaQuery` under the hood.

```tsx
import { Box, useMatches } from '@mantine/core';

function Demo() {
  // Returns 'blue.9' below sm, 'orange.9' from sm to lg, 'red.9' from lg and up
  const color = useMatches({
    base: 'blue.9',
    sm: 'orange.9',
    lg: 'red.9',
  });
  
  // Returns different padding values per breakpoint
  const padding = useMatches({
    base: 'xs',
    sm: 'md',
    lg: 'xl',
  });

  return (
    <Box bg={color} p={padding}>
      Content adapts to screen size
    </Box>
  );
}
```

**When to use `useMatches` vs responsive props:**

| Scenario | Approach |
|----------|----------|
| Styling props (`bg`, `p`, `w`, etc.) | Prefer responsive prop objects: `p={{ base: 'xs', sm: 'md' }}` |
| Non-style values (strings, numbers, booleans) | Use `useMatches` |
| Complex conditional logic | Use `useMatches` or `useMediaQuery` |
| SSR applications | Avoid `useMatches`/`useMediaQuery` for initial render |

**SSR Warning:** Both `useMatches` and `useMediaQuery` are **not recommended for SSR** applications (Next.js, Remix, etc.) as they may cause hydration mismatch. For client-only apps (like this Vite SPA), they are safe to use.

```tsx
// ✅ Good: Use responsive style props when possible (SSR-safe)
<Button size={{ base: 'sm', md: 'lg' }}>Click me</Button>

// ✅ Good: useMatches for non-style values in client-only apps
const columns = useMatches({ base: 1, sm: 2, lg: 3 });
<MyGrid columns={columns} />

// ⚠️ Careful: useMatches with SSR (may cause hydration issues)
// Only safe for elements not rendered on server (tooltips, modals, etc.)
```

Usage:

```tsx
function MyComponent() {
  const isMobile = useIsMobile();

  // Only use conditional rendering when structure fundamentally differs
  return isMobile ? <MobileTable data={data} /> : <DesktopTable data={data} />;
}
```

Agents MUST: 

- Use `useMediaQuery` from `@mantine/hooks`, not raw `window.matchMedia`.
- Prefer Mantine's responsive props over `useMediaQuery` when possible.
- Only use conditional rendering (mobile vs desktop variants) when structure **fundamentally differs** (e.g., table vs cards), not for minor style tweaks. 
- Handle SSR correctly (initial render may not have window access).

---

## 7. Performance Guidelines

Agents MUST consider performance, especially on low-powered mobile devices.

### 7.1 Code Splitting

Use dynamic imports for route-level and heavy components:

```tsx
import { lazy, Suspense } from 'react';
import { LoadingOverlay } from '@mantine/core';

// Lazy load heavy route components
const Dashboard = lazy(() => import('./pages/Dashboard'));
const Reports = lazy(() => import('./pages/Reports'));
const Analytics = lazy(() => import('./pages/Analytics'));

function App() {
  return (
    <Suspense fallback={<LoadingOverlay visible />}>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/reports" element={<Reports />} />
        <Route path="/analytics" element={<Analytics />} />
      </Routes>
    </Suspense>
  );
}
```

### 7.2 List Virtualization

For lists with >50 items, use virtualization:

```tsx
import { useVirtualizer } from '@tanstack/react-virtual';
import { useRef } from 'react';
import { Card, Text } from '@mantine/core';

function VirtualizedList({ items }: { items: Item[] }) {
  const parentRef = useRef<HTMLDivElement>(null);
  
  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 80, // Estimated row height
  });

  return (
    <div ref={parentRef} style={{ height: '500px', overflow: 'auto' }}>
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {virtualizer.getVirtualItems().map((virtualItem) => (
          <div
            key={virtualItem.key}
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: `${virtualItem.size}px`,
              transform: `translateY(${virtualItem. start}px)`,
            }}
          >
            <Card withBorder>
              <Text>{items[virtualItem. index].name}</Text>
            </Card>
          </div>
        ))}
      </div>
    </div>
  );
}
```

### 7.3 Input Debouncing

Debounce expensive operations triggered by user input: 

```tsx
import { TextInput } from '@mantine/core';
import { useDebouncedValue } from '@mantine/hooks';
import { useState, useEffect } from 'react';

function SearchInput({ onSearch }: { onSearch:  (query: string) => void }) {
  const [value, setValue] = useState('');
  const [debounced] = useDebouncedValue(value, 300);

  useEffect(() => {
    onSearch(debounced);
  }, [debounced, onSearch]);

  return (
    <TextInput
      placeholder="Search..."
      value={value}
      onChange={(e) => setValue(e.currentTarget.value)}
    />
  );
}
```

### 7.4 Image Optimization

```tsx
import { Image } from '@mantine/core';

function OptimizedImage({ src, alt }: { src: string; alt: string }) {
  return (
    <Image
      src={src}
      alt={alt}
      loading="lazy" // Native lazy loading
      fit="cover"
      fallbackSrc="/placeholder. jpg" // Fallback for errors
    />
  );
}
```

### 7.5 Performance Rules

Agents MUST: 

- Avoid rendering >100 DOM nodes without virtualization.
- Use `React.memo` for components that receive stable props but have expensive renders.
- Avoid inline function definitions in render for callbacks passed to memoized children.
- Use `useDebouncedValue` or `useDebouncedCallback` for search/filter inputs. 
- Lazy load images below the fold. 

---

## 8. Loading & Error States

Agents MUST implement consistent loading and error patterns across the application.

### 8.1 Loading States

```tsx
import { Skeleton, Stack, Card, Group } from '@mantine/core';

// Skeleton for card grid
function CardGridSkeleton({ count = 6 }: { count?: number }) {
  return (
    <SimpleGrid cols={{ base:  1, sm:  2, md:  3 }}>
      {Array.from({ length: count }).map((_, i) => (
        <Card key={i} withBorder padding="lg">
          <Skeleton height={20} width="70%" mb="sm" />
          <Skeleton height={14} mb="xs" />
          <Skeleton height={14} width="50%" />
        </Card>
      ))}
    </SimpleGrid>
  );
}

// Skeleton for table rows
function TableSkeleton({ rows = 5, cols = 4 }: { rows?: number; cols?:  number }) {
  return (
    <Stack gap="xs">
      {Array.from({ length: rows }).map((_, i) => (
        <Group key={i} gap="md">
          {Array.from({ length: cols }).map((_, j) => (
            <Skeleton key={j} height={20} style={{ flex: 1 }} />
          ))}
        </Group>
      ))}
    </Stack>
  );
}
```

### 8.2 Error States

```tsx
import { Alert, Button, Stack, Text } from '@mantine/core';
import { IconAlertCircle, IconRefresh } from '@tabler/icons-react';

interface ErrorStateProps {
  title?:  string;
  message:  string;
  onRetry?: () => void;
}

function ErrorState({ 
  title = 'Something went wrong', 
  message, 
  onRetry 
}: ErrorStateProps) {
  return (
    <Alert
      icon={<IconAlertCircle size={16} />}
      title={title}
      color="red"
      variant="light"
    >
      <Stack gap="sm">
        <Text size="sm">{message}</Text>
        {onRetry && (
          <Button
            variant="light"
            color="red"
            size="xs"
            leftSection={<IconRefresh size={14} />}
            onClick={onRetry}
          >
            Try again
          </Button>
        )}
      </Stack>
    </Alert>
  );
}
```

### 8.3 Empty States

```tsx
import { Stack, Text, Button, ThemeIcon } from '@mantine/core';
import { IconInbox } from '@tabler/icons-react';

interface EmptyStateProps {
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
}

function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <Stack align="center" gap="md" py="xl">
      <ThemeIcon size={60} radius="xl" variant="light" color="gray">
        <IconInbox size={30} />
      </ThemeIcon>
      <Text fw={500} size="lg">{title}</Text>
      {description && (
        <Text c="dimmed" size="sm" ta="center" maw={400}>
          {description}
        </Text>
      )}
      {action && (
        <Button onClick={action.onClick}>{action.label}</Button>
      )}
    </Stack>
  );
}
```

---

## 9. Anti-Patterns to Avoid

Agents MUST NOT use these patterns:

### 9.1 Layout Anti-Patterns

```tsx
// ❌ Bad: Hard-coded pixel breakpoints
const isMobile = useMediaQuery('(max-width:  768px)');

// ✅ Good: Use theme breakpoints
const theme = useMantineTheme();
const isMobile = useMediaQuery(`(max-width:  ${theme.breakpoints.sm})`);
```

```tsx
// ❌ Bad: Fixed positioning for nav on iOS
<nav style={{ position: 'fixed', bottom: 0 }}>

// ✅ Good: Use AppShell or proper mobile patterns
<AppShell. Navbar>
```

```tsx
// ❌ Bad:  Relying on hover for critical functionality
<div onMouseEnter={() => showMenu()}>
  Hover to see options
</div>

// ✅ Good: Click/tap interaction
<Menu>
  <Menu. Target>
    <Button>Click for options</Button>
  </Menu.Target>
  <Menu. Dropdown>... </Menu.Dropdown>
</Menu>
```

### 9.2 Accessibility Anti-Patterns

```tsx
// ❌ Bad:  Removing focus outline
<Button style={{ outline: 'none' }}>

// ✅ Good: Keep default focus styles or enhance them
<Button>
```

```tsx
// ❌ Bad: Placeholder-only inputs
<TextInput placeholder="Enter email" />

// ✅ Good:  Proper labels
<TextInput label="Email" placeholder="you@example. com" />
```

```tsx
// ❌ Bad: Icon-only button without label
<ActionIcon>
  <IconSettings />
</ActionIcon>

// ✅ Good: Icon button with accessible label
<ActionIcon aria-label="Open settings">
  <IconSettings />
</ActionIcon>
```

### 9.3 Performance Anti-Patterns

```tsx
// ❌ Bad:  Inline styles creating new objects each render
{items.map(item => (
  <div style={{ padding: 10, margin: 5 }}>{item.name}</div>
))}

// ✅ Good: Use Mantine props or stable style objects
const itemStyle = { padding: 10, margin: 5 };
{items.map(item => (
  <Box p="sm" m="xs">{item.name}</Box>
))}
```

```tsx
// ❌ Bad: Rendering thousands of items directly
{allItems.map(item => <Item key={item.id} {...item} />)}

// ✅ Good:  Paginate or virtualize large lists
<VirtualizedList items={allItems} />
// or
<PaginatedList items={allItems} pageSize={20} />
```

### 9.4 Responsive Anti-Patterns

```tsx
// ❌ Bad:  Separate mobile routes
<Route path="/mobile/dashboard" element={<MobileDashboard />} />
<Route path="/desktop/dashboard" element={<DesktopDashboard />} />

// ✅ Good: Single route, adaptive component
<Route path="/dashboard" element={<Dashboard />} /> // Internally adapts
```

```tsx
// ❌ Bad: Using conditional rendering for minor style changes
{isMobile ? (
  <Button size="sm">Submit</Button>
) : (
  <Button size="md">Submit</Button>
)}

// ✅ Good:  Use responsive props
<Button size={{ base: 'sm', md: 'md' }}>Submit</Button>
```

---

## 10. Testing Requirements

### 10.1 Viewport Testing

All layouts MUST be tested at these viewport widths: 

| Width | Represents |
|-------|------------|
| 320px | Small mobile (iPhone SE) |
| 375px | Standard mobile (iPhone 12/13/14) |
| 768px | Tablet / `sm` breakpoint |
| 1024px | Small desktop / `md` breakpoint |
| 1440px | Standard desktop |

### 10.2 Testing Checklist

Before completing any UI work, verify:

- [ ] Layout does not horizontally overflow at 320px width
- [ ] All touch targets are at least 44×44px on mobile
- [ ] Navigation is accessible and functional at all breakpoints
- [ ] Tables convert to appropriate mobile format (cards or scrollable)
- [ ] Forms are usable with touch input
- [ ] All interactive elements have visible focus states
- [ ] Color contrast meets WCAG AA (test with browser dev tools)
- [ ] Header doesn't break with longest translation strings
- [ ] Loading states are implemented for async content
- [ ] Error states are implemented and recoverable

### 10.3 Translation Testing

For multi-language support: 

1.  Identify the 3 longest languages in your supported set (often German, Finnish, Greek).
2. Test all UI with those languages active.
3. Ensure no text truncation hides critical information.
4. Verify breakpoint choices work with longest labels.

---

## 11. Summary of Required Behaviors

When generating or modifying UI code for this project: 

1. **Always design mobile-first**, then scale up to desktop using Mantine breakpoints. 

2. Use **Mantine v8 layout components** (`AppShell`, `Grid`, `SimpleGrid`, `Stack`, `Group`, `Flex`) with **responsive props**. 

3. Keep **routes unified**; adapt only **layout and navigation presentation** per viewport.

4. For the **top header with centered logo and left/right items and 20+ languages**:
   - Make the desktop header flexible (hide less important labels earlier, use icons).
   - Choose the mobile switch breakpoint (`sm` or `md`) based on **where the header breaks** with the **longest translations**. 
   - Below that breakpoint, use a simplified header (logo + burger) and move nav + language selection into a mobile-friendly menu.

5. Convert: 
   - **Tables → cards or stacked layout on mobile** (preferred) OR horizontally scrollable tables where necessary.
   - **Sidebars → collapsible navbars or drawers on mobile**. 

6. **Ensure accessibility**:  visible focus, proper labels, sufficient contrast, adequate touch targets.

7. Build **reusable layout primitives** and use them instead of one-off responsive code.

8. Use **`useMediaQuery`**, **`useMatches`**, or responsive style props for viewport detection, not raw `window` APIs.

9. Use **container queries** for reusable components that must adapt to their container size, not just viewport size.

10. Optimize for **performance**:  code-splitting, virtualization, debouncing, and lazy loading.

11. Implement **loading, error, and empty states** consistently.

12. **Test at multiple viewport widths** with longest translation strings.

These rules are mandatory defaults.  Deviations must be explicitly justified by specific product requirements. 