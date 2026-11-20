---
id: semantic-components
applies_to:
  - "frontend/src/**/*.tsx"
excludes:
  - "frontend/src/common/components/buttons/**"
  - "frontend/src/common/components/typography/**"
trigger: "renders a button, a heading or text, or sets a colour or a style"
---

# Choose components by intent

Pick a component for **what** the thing is, not for how it should look. The
look of an intent is decided once, in its component, so the app stays
consistent and a design change happens in one place.

## Buttons and text

Buttons come from `@components/buttons`, headings and text from
`@components/typography`. Look there for the intent you need (a primary
action, a cancel, a destructive action, a page title, helper text, …). If none
fits, add a semantic component there rather than styling a raw Mantine
`Button`, `Title` or `Text` in place.

```tsx
// Bad: style chosen on the spot
<Button variant="filled" color="accent" size="lg">Get started</Button>
<Title order={2} c="accent.9">Settings</Title>
<Text size="sm" c="dimmed">Enter your email</Text>

// Good: chosen by intent
<ActionButton>Get started</ActionButton>
<SectionTitle accent>Settings</SectionTitle>
<HelperText>Enter your email</HelperText>
```

## Colours

Colours come from the theme (`config/mantineTheme.ts`; `accent` is the primary
colour), never as hex or rgb values.

```tsx
// Bad
style={{ color: '#29D0DE', background: 'rgba(139, 92, 246, 0.2)' }}

// Good
c="accent.9"
style={{ background: 'var(--mantine-color-accent-0)' }}
```

## Styles

Simple layout may be inline (flex, grid, positioning, a dynamic value).
Anything more (several related properties, hover or focus states, a pattern
used twice) goes into a CSS module.

```tsx
// Good: simple layout
style={{ display: 'flex', alignItems: 'center' }}

// Bad: belongs in a CSS module
style={{
  background: 'var(--mantine-color-accent-0)',
  border: '1px solid var(--mantine-color-accent-2)',
  borderRadius: 'var(--mantine-radius-md)',
}}
```

