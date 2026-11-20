---
id: responsive
applies_to:
  - "frontend/src/**/*.tsx"
  - "frontend/src/**/*.css"
trigger: "lays out a page or component, adds a table, list, image, input or icon button, or loads data or code"
---

# Works on every screen, in every language, fast

## Every screen

- **A phone works:** the main content needs no horizontal scrolling. One route
  serves every screen size; there are no separate mobile pages.
- **Breakpoints come from the theme** (`theme.breakpoints`, or responsive
  props such as `{ base: 'sm', md: 'lg' }`), never as pixel values.
- **Small differences use responsive props**, not two branches.
- **Tables become cards or stacks on a phone.**

```tsx
// Bad
const isMobile = useMediaQuery("(max-width: 768px)");
{isMobile ? <Button size="sm">Save</Button> : <Button size="md">Save</Button>}

// Good
<ActionButton size={{ base: "sm", md: "md" }}>Save</ActionButton>
```

## Every language

Layouts with text, the header and navigation above all, hold up with the
longest translations. Measured over all UI texts, the longest are Somali
(`so`, about 1.7× English), Greek (`el`, 1.3×, a non-Latin script), French and
German (about 1.2×). Somali shows a layout's limits first; German is the
everyday check.

## Everyone

- Icon-only buttons have an `aria-label`.
- Inputs have a visible label, not just a placeholder.
- Nothing works only on hover; menus open on click or tap.
- Focus outlines stay visible.

```tsx
// Bad
<ActionIcon><IconSettings /></ActionIcon>
<TextInput placeholder="Email" />

// Good
<ActionIcon aria-label={t('settings.open')}><IconSettings /></ActionIcon>
<TextInput label={t('auth.email')} placeholder="you@example.com" />
```

## Fast

- **Routes load lazily**: a page's code loads when the page is opened.
- **Long lists page** instead of rendering every row.
- **Search inputs are debounced**: a request per pause, not per keystroke.
- **Images come in the size they are shown at**, and below the fold load
  lazily.
