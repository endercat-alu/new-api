---
name: i18n-translate
description: Complete and maintain en/zh frontend translations in web/default. Use for new UI copy, missing keys, incorrect translations, or locale synchronization.
---

# Frontend i18n

Locales: `web/default/src/i18n/locales/en.json`, `web/default/src/i18n/locales/zh.json`. Both are flat JSON. Keys are English source strings.

Add every key to both files. The `en` value MUST equal the key unless an intentional override exists. The `zh` value MUST be the Chinese translation. Preserve placeholders, interpolation syntax, whitespace, and required punctuation exactly. Brand names, model names, URLs, API paths, JSON keys, code fragments, and identifiers MUST remain untranslated. Use `useTranslation()` and `t('English source text')` in React components. MUST NOT add another locale UNLESS project language policy changes.

Workflow: find the UI call site and nearby keys; add or update both locale files; run `bun run --cwd web/default i18n:sync` then `bun run --cwd web/default typecheck`. IF sync reports a problem, THEN read `web/default/src/i18n/locales/_reports/_sync-report.json`.
