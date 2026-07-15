# AGENTS.md

new-api is a Go/Gin/GORM AI API gateway with a single frontend at `web/default/` (React 19, TypeScript, Rsbuild, Base UI, Tailwind). Backend layers: Router -> Controller -> Service -> Model.

Paths: `router/` HTTP routing; `controller/` handlers; `service/` business logic; `model/` GORM access; `relay/` upstream adapters; `middleware/` auth, rate limits, logging; `setting/` system/model/ratio/performance config; `common/` shared utilities; `dto/`, `types/`, `constant/` DTOs and constants; `i18n/` backend en/zh; `web/default/` sole frontend and frontend en/zh i18n.

## JSON

Business code MUST marshal/unmarshal JSON only through `common/json.go`: `common.Marshal`, `common.Unmarshal`, `common.UnmarshalJsonStr`, `common.DecodeJson`, `common.GetJsonType`. Types from `encoding/json` MAY be referenced. Direct `encoding/json` marshal/unmarshal calls MUST NOT be used.

## Database

All database code MUST work on SQLite, MySQL >= 5.7.8, and PostgreSQL >= 9.6 simultaneously.

Prefer GORM APIs. Standard row locks in `model/` MUST use `lockForUpdate(tx)`. Reserved-word columns in raw SQL MUST use `commonGroupCol` and `commonKeyCol`. Boolean literals MUST use `commonTrueVal` and `commonFalseVal`. Dialect branches MUST use `common.UsingSQLite`, `common.UsingMySQL`, and `common.UsingPostgreSQL`, and MUST provide valid behavior for all three. Dialect-only functions, operators, types, or DDL MUST NOT be used without a cross-database fallback. Migrations MUST run on all three databases. SQLite-incompatible column changes MUST use a compatible migration path.

## Billing safety

User-controlled or upstream-controlled billing multipliers MUST be upper-bounded before calculation. Reuse existing bounds such as `dto.MaxImageN`, `relaycommon.MaxTaskDurationSeconds`, and `maxTokensLimit`. Bypass paths (passthrough fields, metadata, multipart, media metadata, upstream settlement values) MUST enforce the same bounds. Quota/token conversion MUST NOT use unrestricted bare `int` casts; MUST use `common.QuotaFromFloat`, `common.QuotaRound`, or `common.QuotaFromDecimal`. Billing paths MUST use the matching `*Checked` helper, store clamps on `relayInfo.QuotaClamp` or the task settlement context, and call `attachQuotaSaturation` before writing consume/task logs. Ratios MUST be written only through `types.PriceData.AddOtherRatio`. Pre-consume and settlement MUST preserve non-negative, saturation, and audit invariants. Saturated oversized pre-consume MUST fail as insufficient quota. Unsigned inputs MUST also have upper bounds. Regression tests MUST sit next to the validation, conversion, or settlement boundary they protect.

Before changing dynamic billing expressions, MUST fully read `pkg/billingexpr/expr.md`.

## Upstream request DTOs

Optional scalar fields parsed from client JSON and re-marshaled upstream MUST use pointer types with `omitempty`. IF the field is absent, THEN use `nil` and omit on marshal. IF the field is explicit `0`, `0.0`, or `false`, THEN keep a non-nil pointer and send it upstream.

## Frontend and i18n

Frontend package and script work MUST prefer Bun. From the repo root: install with `bun install --cwd web`; run scripts with `bun run --cwd web/default <script>`. Frontend locales are flat JSON at `web/default/src/i18n/locales/{en,zh}.json` with English source strings as keys. New UI copy MUST use `useTranslation()` and `t('English source text')` and MUST update both `en` and `zh`. Preserve placeholders such as `{{name}}` and leave code, URLs, API paths, and identifiers untranslated. IF a new channel supports `StreamOptions`, THEN add it to `streamSupportedChannels`.

## Verification

Scope verification to the change. Backend behavior changes MUST run relevant tests; run `go test ./...` and `go build ./...` when the change warrants full coverage. Frontend changes MUST run the matching checks; full check is `bun run --cwd web/default build:check`. i18n changes MUST run `bun run --cwd web/default i18n:sync` and leave no unexpected diffs. IF only compile/build was verified, THEN report only compile/build success; MUST NOT claim runtime behavior was verified.
