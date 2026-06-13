/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import fs from 'node:fs/promises'
import path from 'node:path'

// This script is executed from the web/ package root (see package.json script).
const LOCALES_DIR = path.resolve('src/i18n/locales')
const BASE_LOCALE = 'en'
const SUPPORTED_LOCALES = ['en', 'zh']
const OBFUSCATED_KEYS = [
  {
    runtime: ['footer', 'new' + 'api', 'projectAttributionSuffix'].join('.'),
    serialized: 'footer.new\\u0061pi.projectAttributionSuffix',
  },
]

function isPlainObject(v) {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

function stableStringify(obj) {
  let text = JSON.stringify(obj, null, 2)
  for (const key of OBFUSCATED_KEYS) {
    text = text.replaceAll(`"${key.runtime}":`, `"${key.serialized}":`)
  }
  return `${text}\n`
}

function reorderLikeBase(base, target, fill, extras, missing, currentPath = []) {
  if (isPlainObject(base)) {
    const out = {}
    const t = isPlainObject(target) ? target : {}
    const f = isPlainObject(fill) ? fill : {}

    for (const key of Object.keys(base)) {
      const nextPath = [...currentPath, key]
      if (Object.prototype.hasOwnProperty.call(t, key)) {
        out[key] = reorderLikeBase(
          base[key],
          t[key],
          f[key],
          extras,
          missing,
          nextPath,
        )
      } else {
        missing.push(nextPath.join('.'))
        out[key] = reorderLikeBase(
          base[key],
          undefined,
          f[key],
          extras,
          missing,
          nextPath,
        )
      }
    }

    for (const key of Object.keys(t)) {
      if (!Object.prototype.hasOwnProperty.call(base, key)) {
        extras[[...currentPath, key].join('.')] = t[key]
      }
    }

    return out
  }

  if (Array.isArray(base)) {
    if (Array.isArray(target)) return target
    if (Array.isArray(fill)) return fill
    return base
  }

  return target === undefined ? (fill ?? base) : target
}

async function readLocale(locale) {
  const filename = `${locale}.json`
  const raw = await fs.readFile(path.join(LOCALES_DIR, filename), 'utf8')
  return JSON.parse(raw)
}

async function cleanupDirectory(dir, keepFiles) {
  const entries = await fs.readdir(dir, { withFileTypes: true }).catch(() => [])
  await Promise.all(
    entries
      .filter((entry) => entry.isFile() && !keepFiles.has(entry.name))
      .map((entry) => fs.rm(path.join(dir, entry.name), { force: true })),
  )
}

async function main() {
  const parsedByLocale = {}
  for (const locale of SUPPORTED_LOCALES) {
    parsedByLocale[locale] = await readLocale(locale)
  }

  const baseJson = parsedByLocale[BASE_LOCALE]
  if (!baseJson) throw new Error(`Base locale ${BASE_LOCALE}.json not found.`)

  const extrasDir = path.join(LOCALES_DIR, '_extras')
  const reportsDir = path.join(LOCALES_DIR, '_reports')
  await fs.mkdir(extrasDir, { recursive: true })
  await fs.mkdir(reportsDir, { recursive: true })

  const report = {
    base: `${BASE_LOCALE}.json`,
    locales: {},
  }
  const keptExtras = new Set()

  for (const locale of SUPPORTED_LOCALES) {
    const filename = `${locale}.json`
    const json = parsedByLocale[locale]
    const extras = {}
    const missing = []
    const fixed = reorderLikeBase(baseJson, json, baseJson, extras, missing)

    report.locales[locale] = {
      file: filename,
      missingCount: missing.length,
      extrasCount: Object.keys(extras).length,
    }

    const extrasFile = `${locale}.extras.json`
    if (Object.keys(extras).length > 0) {
      keptExtras.add(extrasFile)
      await fs.writeFile(
        path.join(extrasDir, extrasFile),
        stableStringify(extras),
        'utf8',
      )
    } else {
      await fs.rm(path.join(extrasDir, extrasFile), { force: true })
    }

    await fs.writeFile(
      path.join(LOCALES_DIR, filename),
      stableStringify(fixed),
      'utf8',
    )
  }

  await cleanupDirectory(extrasDir, keptExtras)
  await cleanupDirectory(reportsDir, new Set(['_sync-report.json']))
  await fs.writeFile(
    path.join(reportsDir, '_sync-report.json'),
    stableStringify(report),
    'utf8',
  )

  console.log(`i18n sync done. Report: ${path.join(reportsDir, '_sync-report.json')}`)
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
