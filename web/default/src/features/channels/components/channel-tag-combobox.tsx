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
import * as React from 'react'
import { Add01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
  useComboboxAnchor,
} from '@/components/ui/combobox'
import type { ChannelTagSummary } from '../types'

type ChannelTagComboboxProps = {
  value?: string | null
  options: ChannelTagSummary[]
  onChange: (value: string, option?: ChannelTagSummary) => void
  onSearchChange?: (value: string) => void
  placeholder?: string
  emptyText?: string
  disabled?: boolean
  loading?: boolean
  className?: string
}

function normalizeTag(value: string | null | undefined) {
  return String(value || '').trim()
}

export function ChannelTagCombobox(props: ChannelTagComboboxProps) {
  const { t } = useTranslation()
  const anchorRef = useComboboxAnchor()
  const normalizedValue = normalizeTag(props.value)
  const selected = React.useMemo(
    () => (normalizedValue ? [normalizedValue] : []),
    [normalizedValue]
  )
  const [inputValue, setInputValue] = React.useState('')
  const [open, setOpen] = React.useState(false)

  const optionMap = React.useMemo(() => {
    const map = new Map<string, ChannelTagSummary>()
    for (const option of props.options) {
      const tag = normalizeTag(option.tag)
      if (tag) map.set(tag, option)
    }
    return map
  }, [props.options])

  const lowerOptionMap = React.useMemo(() => {
    const map = new Map<string, ChannelTagSummary>()
    for (const [tag, option] of optionMap.entries()) {
      map.set(tag.toLowerCase(), option)
    }
    return map
  }, [optionMap])

  const trimmedInput = inputValue.trim()
  const inputMatchesExisting =
    trimmedInput.length > 0 &&
    Array.from(optionMap.keys()).some(
      (tag) => tag.toLowerCase() === trimmedInput.toLowerCase()
    )
  const canCreate = trimmedInput.length > 0 && !inputMatchesExisting

  const items = React.useMemo(() => {
    const set = new Set<string>()
    if (normalizedValue) set.add(normalizedValue)
    for (const option of props.options) {
      const tag = normalizeTag(option.tag)
      if (tag) set.add(tag)
    }
    if (canCreate) set.add(trimmedInput)
    return Array.from(set)
  }, [normalizedValue, props.options, canCreate, trimmedInput])

  const commitValue = React.useCallback(
    (rawValue: string) => {
      const nextValue = normalizeTag(rawValue)
      const matchedOption =
        optionMap.get(nextValue) || lowerOptionMap.get(nextValue.toLowerCase())
      props.onChange(matchedOption?.tag || nextValue, matchedOption)
      setInputValue('')
      props.onSearchChange?.('')
    },
    [lowerOptionMap, optionMap, props]
  )

  const handleValueChange = (next: string[]) => {
    const nextValue = next.at(-1) || ''
    commitValue(nextValue)
  }

  const handleInputValueChange = (value: string) => {
    setInputValue(value)
    props.onSearchChange?.(value)
  }

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== 'Enter' || trimmedInput.length === 0) return

    const popup = document.querySelector<HTMLElement>(
      '[data-slot="combobox-content"][data-open]'
    )
    const hasHighlight = popup?.querySelector('[data-highlighted]') != null
    if (hasHighlight) return

    event.preventDefault()
    commitValue(trimmedInput)
  }

  return (
    <Combobox
      multiple
      items={items}
      value={selected}
      onValueChange={handleValueChange}
      inputValue={inputValue}
      onInputValueChange={handleInputValueChange}
      open={open}
      onOpenChange={setOpen}
      disabled={props.disabled}
    >
      <ComboboxChips ref={anchorRef} className={cn('w-full', props.className)}>
        <ComboboxValue>
          {(values: string[]) =>
            values.map((tag) => (
              <ComboboxChip key={tag}>
                <span className='max-w-[16rem] truncate'>{tag}</span>
              </ComboboxChip>
            ))
          }
        </ComboboxValue>
        <ComboboxChipsInput
          placeholder={
            selected.length === 0
              ? props.placeholder || t('Search tags...')
              : undefined
          }
          onKeyDown={handleKeyDown}
          aria-label={props.placeholder || t('Search tags...')}
        />
      </ComboboxChips>

      <ComboboxContent anchor={anchorRef}>
        <ComboboxList>
          <ComboboxCollection>
            {(item: string) => {
              const option = optionMap.get(item)
              const isCreate = canCreate && item === trimmedInput
              return (
                <ComboboxItem key={item} value={item}>
                  {isCreate ? (
                    <>
                      <HugeiconsIcon
                        icon={Add01Icon}
                        strokeWidth={2}
                        className='text-muted-foreground'
                        aria-hidden='true'
                      />
                      <span className='truncate'>
                        {t('Add "{{value}}"', { value: item })}
                      </span>
                    </>
                  ) : (
                    <div className='flex min-w-0 flex-1 items-center justify-between gap-2'>
                      <span className='truncate'>{item}</span>
                      {option && (
                        <div className='flex shrink-0 items-center gap-1'>
                          <Badge variant='outline' className='h-5 px-1.5'>
                            {option.count}
                          </Badge>
                          {typeof option.priority === 'number' && (
                            <Badge variant='secondary' className='h-5 px-1.5'>
                              P:{option.priority}
                            </Badge>
                          )}
                          {typeof option.weight === 'number' && (
                            <Badge variant='secondary' className='h-5 px-1.5'>
                              W:{option.weight}
                            </Badge>
                          )}
                        </div>
                      )}
                    </div>
                  )}
                </ComboboxItem>
              )
            }}
          </ComboboxCollection>
        </ComboboxList>
        <ComboboxEmpty>
          {props.loading
            ? t('Loading')
            : props.emptyText || t('No matching items')}
        </ComboboxEmpty>
      </ComboboxContent>
    </Combobox>
  )
}
