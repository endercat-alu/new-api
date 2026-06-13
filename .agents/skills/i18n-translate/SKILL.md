---
name: i18n-translate
description: >-
  Complete and maintain frontend i18n translations for this project. Covers
  finding missing translation keys and adding translations for supported locales
  (en, zh). Use when the user asks to add translations, fix i18n, complete
  missing translations, or when new UI text needs to be internationalized.
---

# Frontend i18n Workflow

## Supported locales

- `en`
- `zh`

## Files

- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`

## Rules

1. Keys are English source strings.
2. `en` value should normally equal the key.
3. `zh` must contain the Chinese translation.
4. Preserve placeholders such as `{{name}}` in every translation.
5. Keep brand names, model names, URLs, API paths, JSON keys, code-like strings, and identifiers untranslated.
6. UI labels, button text, error messages, descriptions, and action words should be translated in `zh`.
7. Run all commands from `web/default/`.

## Workflow

### Step 1: Add or update keys

For new UI copy:

```tsx
const { t } = useTranslation()

t('English source text')
```

Add the same key to both locale files:

```json
{
  "translation": {
    "English source text": "English source text"
  }
}
```

```json
{
  "translation": {
    "English source text": "中文翻译"
  }
}
```

### Step 2: Sync locale files

```bash
cd web/default
bun run i18n:sync
```

Read `web/default/src/i18n/locales/_reports/_sync-report.json` if sync reports missing or extra keys.

### Step 3: Verify

```bash
cd web/default
bun run typecheck
```

## Notes

- Do not add new locale files unless project language policy changes.
- Do not create temporary multi-language translation scripts for normal i18n work.
- `zh-CN`, `zh-TW`, `zh-Hans`, and `zh-Hant` are runtime-compatible aliases of `zh`.
