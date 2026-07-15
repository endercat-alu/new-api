---
name: shadcn-ui
description: Apply repository-aware component composition rules when changing the web/default UI or its shared shadcn-style components.
---

# web/default UI components

Ground truth: app root `web/default/`; config `web/default/components.json`; shared components `web/default/src/components/`. Read `components.json` and the current TypeScript config before choosing imports.

Search existing shared and feature-local components before creating one. Reuse existing primitives, variants, spacing, colors, typography, icons, and form patterns. Compose primitives in feature code. Change shared primitives ONLY IF the behavior is reusable across features. Preserve labels, keyboard operation, focus, disabled state, validation feedback, and semantic structure. MUST NOT introduce a second component library for behavior already covered. Product copy MUST use the existing en/zh i18n workflow. Frontend commands MUST use Bun.

Workflow: read `web/default/components.json` and neighboring implementations; confirm whether the primitive already exists under `web/default/src/components/`. IF a registry component is required, THEN run `bun x --cwd web/default shadcn@latest add <component>`, inspect the diff, and adapt it to current aliases and composition patterns without replacing unrelated local customizations. Run relevant tests and `bun run --cwd web/default build:check`.
