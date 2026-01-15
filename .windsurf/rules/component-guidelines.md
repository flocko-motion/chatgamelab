---
trigger: model_decision
description: Apply when creating, using or modifying react components
---

# Component Design and Usage Guidelines

This document outlines how to properly use, create, and organize components in the ChatGameLab frontend.

## Component Architecture

### Layered Component Structure

```
src/
├── common/components/     # Shared/reusable components
│   ├── Button.tsx        # Common wrapper for Mantine Button
│   ├── Input.tsx         # Common wrapper for Mantine TextInput
│   ├── Card.tsx          # Common wrapper for Mantine Card
│   └── index.ts          # Export all common components
├── features/             # Feature-specific components
│   ├── auth/
│   │   └── components/
│   │       ├── LoginForm.tsx
│   │       └── DevModeIndicator.tsx
│   ├── games/
│   │   └── components/
│   │       ├── GameCard.tsx
│   │       └── GameEditor.tsx
│   └── profile/
│       └── components/
│           └── ProfileForm.tsx
└── routes/              # Page-level components
    ├── auth/
    │   └── login.tsx
    └── index.tsx
```

## Component Usage Rules

### 1. NEVER Use Mantine Components Directly

❌**Incorrect:**

```tsx
import { Button, TextInput } from "@mantine/core";

function MyComponent() {
  return (
    <div>
      <Button onClick={handleClick}>Click me</Button>
      <TextInput placeholder="Enter text" />
    </div>
  );
}
```

✅ **Correct:**

```tsx
import { Button, Input } from "@components";

function MyComponent() {
  return (
    <div>
      <Button onClick={handleClick}>Click me</Button>
      <Input placeholder="Enter text" />
    </div>
  );
}
```

### 2. Use Common Components First

Before creating new components:

1. **Check existing common components** in `src/common/components/`
2. **Review documentation** in `docs/components-overview.md`
3. **Use feature-local components** only when common components don't fit

### 3. Component Placement Guidelines

#### Common Components (`src/common/components/`)

- **Purpose**: Reusable across multiple features
- **Examples**: Button, Input, Modal, Card, Dropdown
- **Requirements**:
  - Must be generic and feature-agnostic
  - Must have proper TypeScript interfaces
  - Must include comprehensive props documentation
  - Must be exported from `index.ts`

#### Feature Components (`src/features/<feature>/components/`)

- **Purpose**: Specific to a particular feature domain
- **Examples**: LoginForm, GameCard, ProfileForm
- **Requirements**:
  - Should be cohesive within the feature domain
  - Can use common components as building blocks
  - Should be feature-specific but potentially reusable within that feature

#### Route Components (`src/routes/`)

- **Purpose**: Page-level components that orchestrate features
- **Examples**: HomePage, LoginPage, GamesPage
- **Requirements**:
  - Should be thin orchestration layers
  - Should compose feature components
  - Should handle routing and page-level state
  - Should NOT contain complex UI logic

## Component Design Principles

### 1. Common Component Wrapper Pattern

Every common component should:

1. **Wrap a Mantine component** with consistent styling
2. **Extend the props interface** while maintaining Mantine compatibility
3. **Apply consistent theming** and design tokens
4. **Provide sensible defaults** for common use cases

**Example Common Button:**

```tsx
// src/common/components/Button.tsx
import {
  Button as MantineButton,
  type ButtonProps as MantineButtonProps,
} from "@mantine/core";

export interface ButtonProps extends Omit<
  MantineButtonProps,
  "color" | "variant"
> {
  variant?: "primary" | "secondary" | "danger";
  size?: "xs" | "sm" | "md" | "lg" | "xl";
  fullWidth?: boolean;
  loading?: boolean;
}

export function Button({
  variant = "primary",
  size = "md",
  fullWidth = false,
  loading = false,
  ...props
}: ButtonProps) {
  const getVariantStyles = () => {
    switch (variant) {
      case "primary":
        return { color: "violet" as const };
      case "secondary":
        return { color: "gray" as const, variant: "outline" as const };
      case "danger":
        return { color: "red" as const };
      default:
        return { color: "violet" as const };
    }
  };

  return (
    <MantineButton
      {...getVariantStyles()}
      {...props}
      loading={loading}
      style={{ width: fullWidth ? "100%" : undefined, ...props.style }}
    />
  );
}
```

### 2. Design System Consistency

All components must follow the established design system:

- **Primary color**: Violet
- **Font**: Inter, system-ui fallback
- **Border radius**: md (medium)
- **Spacing**: Use Mantine spacing scale consistently
- **Typography**: Use Mantine Title/Text components with consistent sizes

### 3. TypeScript Requirements

- **Strict typing** for all props
- **Generic types** where appropriate
- **Proper interface extension** from Mantine components
- **JSDoc comments** for complex prop behaviors

### 4. Component Composition

- **Prefer composition over inheritance**
- **Use compound component patterns** when appropriate
- **Keep components focused** on single responsibilities
- **Avoid deep nesting** of component logic

## Creating New Components

### 1. Before Creating

1. **Search existing components** in `src/common/components/`
2. **Check if a Mantine component** can be wrapped instead
3. **Review similar components** for patterns

### 2. Common Component Creation Process

1. **Create the component file** in `src/common/components/`
2. **Wrap the appropriate Mantine component**
3. **Define the props interface** extending Mantine props
4. **Apply consistent styling** and theming
5. **Add comprehensive TypeScript types**
6. **Export from index.ts**
7. **Add to documentation** in `docs/components-overview.md`

### 3. Feature Component Creation Process

1. **Create feature directory** if it doesn't exist: `src/features/<feature>/components/`
2. **Create component file** with clear naming
3. **Use common components** as building blocks
4. **Keep feature-specific logic** contained
5. **Maintain consistent patterns** with other feature components

## Documentation Requirements

### Component Documentation

Every common component must have:

1. **One-line description** in `docs/components-overview.md`
2. **JSDoc comments** for complex props
3. **Usage examples** for complex components
4. **Props interface documentation**

### docs/components-overview.md Format

```markdown
# Components Overview

## Common Components

- **Button**: Styled button with primary/secondary/danger variants
- **Input**: Text input with consistent styling and validation
- **Card**: Container component with consistent shadows and padding
- **Modal**: Modal wrapper with consistent sizing and animations

## Feature Components

### Auth

- **LoginForm**: Login form with Auth0 integration
- **DevModeIndicator**: Development mode status display

### Games

- **GameCard**: Card component for displaying game information
- **GameEditor**: Game creation and editing interface
```

## Migration Strategy

When existing code uses direct Mantine components:

1. **Identify commonly used patterns**
2. **Create common component wrappers**
3. **Gradually migrate** existing usage
4. **Update imports** throughout the codebase
5. **Remove direct Mantine imports** from route/feature components

## Quality Assurance

### Component Review Checklist

- [ ] Does not use Mantine components directly
- [ ] Uses common components when available
- [ ] Has proper TypeScript interfaces
- [ ] Is placed in correct directory
- [ ] Follows design system guidelines
- [ ] Is exported from index.ts (for common components)
- [ ] Maintains consistent patterns with similar components

### Code Review Focus

1. **Direct Mantine usage** - should be flagged and replaced
2. **Component placement** - ensure correct directory structure
3. **Design consistency** - verify adherence to design system
4. **TypeScript quality** - check interfaces and type safety
5. **Documentation completeness** - ensure proper docs exist
