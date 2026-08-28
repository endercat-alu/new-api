# AGENTS.md

## Project layout

- Treat `new-api` as a Go/Gin/GORM AI API gateway with the backend flow `router/` -> `controller/` -> `service/` -> `model/`.
- Use `web/default/` as the React 19, TypeScript, Rsbuild, Base UI, and Tailwind frontend.
- Place HTTP routes in `router/`, handlers in `controller/`, business logic in `service/`, GORM access in `model/`, and upstream adapters in `relay/`.
- Place authentication, rate-limit, and logging middleware in `middleware/`; system, model, ratio, and performance configuration in `setting/`; shared utilities in `common/`; DTOs in `dto/`; shared types in `types/`; constants in `constant/`; and backend English/Chinese localization in `i18n/`.

## JSON

- Route business-code JSON operations through `common/json.go`: use `common.Marshal`, `common.Unmarshal`, `common.UnmarshalJsonStr`, `common.DecodeJson`, and `common.GetJsonType`.
- Use `encoding/json` types when needed, while keeping JSON marshal and unmarshal operations on the `common` wrappers.

## Database

- Keep every database change compatible with SQLite, MySQL `>= 5.7.8`, and PostgreSQL `>= 9.6`.
- Use GORM APIs for database access and apply `lockForUpdate(tx)` for standard row locks in `model/`.
- Quote reserved-word columns in raw SQL with `commonGroupCol` and `commonKeyCol`, and use `commonTrueVal` and `commonFalseVal` for boolean literals.
- Select dialect branches through `common.UsingSQLite`, `common.UsingMySQL`, and `common.UsingPostgreSQL`, then provide valid behavior for all three databases.
- Implement dialect-specific functions, operators, types, and DDL with a valid cross-database fallback.
- Keep migrations runnable on all three databases and use a SQLite-compatible migration path for SQLite-incompatible column changes.

## Billing safety

- Bound every user-controlled or upstream-controlled billing multiplier before calculation, reusing bounds such as `dto.MaxImageN`, `relaycommon.MaxTaskDurationSeconds`, and `maxTokensLimit`.
- Apply the same bounds to passthrough fields, metadata, multipart data, media metadata, and upstream settlement values.
- Route quota and token conversion through `common.QuotaFromFloat`, `common.QuotaRound`, or `common.QuotaFromDecimal` and use the corresponding `*Checked` helper in billing paths.
- Store conversion clamps on `relayInfo.QuotaClamp` or the task settlement context, and call `attachQuotaSaturation` before writing consume or task logs.
- Write price ratios through `types.PriceData.AddOtherRatio`.
- Keep pre-consume and settlement values non-negative and preserve saturation and audit invariants.
- Treat an oversized pre-consume that reaches saturation as insufficient quota and fail it.
- Bound unsigned inputs before they enter billing calculations.
- Place regression tests beside the validation, conversion, or settlement boundary they protect.
- Read `pkg/billingexpr/expr.md` in full before changing dynamic billing expressions.

## Upstream request DTOs

- Model optional scalar fields parsed from client JSON and re-marshaled upstream as pointer types with `omitempty`.
- If an optional field is absent, set it to `nil` so marshal omits it; if the client explicitly sends `0`, `0.0`, or `false`, keep a non-nil pointer and send that value upstream.

## Frontend and i18n

- Prefer Bun for frontend package and script work: install from the repository root with `bun install --cwd web`, and run scripts with `bun run --cwd web/default <script>`.
- Keep frontend locales as flat JSON files at `web/default/src/i18n/locales/{en,zh}.json`, using English source strings as keys.
- Add new UI copy through `useTranslation()` and `t('English source text')`, then update both `en` and `zh` locales.
- Preserve placeholders such as `{{name}}`, and leave code, URLs, API paths, and identifiers untranslated.
- When a new channel supports `StreamOptions`, add its channel type to `streamSupportedChannels`.

## Verification

- Scope verification to the change: run relevant tests for backend behavior changes, and run `go test ./...` plus `go build ./...` when the change warrants full coverage.
- Run the matching frontend checks; use `bun run --cwd web/default build:check` for the full frontend check.
- Run `bun run --cwd web/default i18n:sync` for i18n changes and leave no unexpected diffs.
- Report only `Compilation/Build successful` when compilation or build is the sole successful verification, and reserve runtime-behavior claims for evidence beyond compilation or build.
