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
import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import {
  FlaskConical,
  Loader2,
  Power,
  PowerOff,
  RefreshCw,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { StatusBadge } from '@/components/status-badge'
import {
  getMultiKeyStatus,
  enableMultiKey,
  disableMultiKey,
  deleteMultiKey,
  enableAllMultiKeys,
  disableAllMultiKeys,
  deleteDisabledMultiKeys,
  testMultiKeys,
} from '../../api'
import { MULTI_KEY_FILTER_OPTIONS } from '../../constants'
import {
  channelsQueryKeys,
  formatTimestamp,
  getMultiKeyStatusConfig,
  getMultiKeyConfirmMessage,
  isDestructiveAction,
} from '../../lib'
import type {
  KeyStatus,
  MultiKeyConfirmAction,
  MultiKeyTestResult,
} from '../../types'
import { useChannels } from '../channels-provider'
import { StatisticsCard } from './multi-key-statistics-card'
import { MultiKeyTableRowActions } from './multi-key-table-row-actions'

type MultiKeyManageDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function MultiKeyManageDialog({
  open,
  onOpenChange,
}: MultiKeyManageDialogProps) {
  const { t } = useTranslation()
  const { currentRow } = useChannels()
  const queryClient = useQueryClient()

  // Data state
  const [isLoading, setIsLoading] = useState(false)
  const [keys, setKeys] = useState<KeyStatus[]>([])
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [totalPages, setTotalPages] = useState(0)
  const [enabledCount, setEnabledCount] = useState(0)
  const [manualDisabledCount, setManualDisabledCount] = useState(0)
  const [autoDisabledCount, setAutoDisabledCount] = useState(0)

  // UI state
  const [statusFilter, setStatusFilter] = useState<number | null>(null)
  const [confirmAction, setConfirmAction] =
    useState<MultiKeyConfirmAction | null>(null)
  const [isPerformingAction, setIsPerformingAction] = useState(false)
  const [testIntervalMs, setTestIntervalMs] = useState(1000)
  const [isTestingKeys, setIsTestingKeys] = useState(false)
  const [testingKeyIndexes, setTestingKeyIndexes] = useState<Set<number>>(
    () => new Set()
  )
  const [keyTestResults, setKeyTestResults] = useState<
    Record<number, MultiKeyTestResult>
  >({})

  // Reset and load data when dialog opens
  useEffect(() => {
    if (open && currentRow) {
      setCurrentPage(1)
      setStatusFilter(null)
      loadKeyStatus(1, pageSize, null)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, currentRow?.id])

  async function loadKeyStatus(
    page: number = currentPage,
    size: number = pageSize,
    status: number | null = statusFilter
  ) {
    if (!currentRow) return

    setIsLoading(true)
    try {
      const response = await getMultiKeyStatus(
        currentRow.id,
        page,
        size,
        status === null ? undefined : status
      )

      if (response.success && response.data) {
        setKeys(response.data.keys || [])
        setTotal(response.data.total || 0)
        setCurrentPage(response.data.page || 1)
        setPageSize(response.data.page_size || 10)
        setTotalPages(response.data.total_pages || 0)
        setEnabledCount(response.data.enabled_count || 0)
        setManualDisabledCount(response.data.manual_disabled_count || 0)
        setAutoDisabledCount(response.data.auto_disabled_count || 0)
      } else {
        toast.error(response.message || t('Failed to load key status'))
      }
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : t('Failed to load key status')
      )
    } finally {
      setIsLoading(false)
    }
  }

  const handleStatusFilterChange = (value: string) => {
    const newFilter = value === 'all' ? null : parseInt(value)
    setStatusFilter(newFilter)
    setCurrentPage(1)
    loadKeyStatus(1, pageSize, newFilter)
  }

  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage)
    loadKeyStatus(newPage, pageSize)
  }

  const handleTestKeys = async () => {
    if (!currentRow) return

    setIsTestingKeys(true)
    setKeyTestResults({})
    try {
      const normalizedInterval = Math.max(1, Math.floor(testIntervalMs || 1000))
      const response = await testMultiKeys(currentRow.id, normalizedInterval)

      if (response.success) {
        const results = response.data || []
        setKeyTestResults(
          Object.fromEntries(results.map((result) => [result.index, result]))
        )
        toast.success(response.message || t('Key test completed'))
        queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() })
        loadKeyStatus(currentPage, pageSize)
      } else {
        toast.error(response.message || t('Key test failed'))
      }
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Key test failed'))
    } finally {
      setIsTestingKeys(false)
    }
  }

  const handleTestKey = async (keyIndex: number) => {
    if (!currentRow) return

    setTestingKeyIndexes((prev) => new Set(prev).add(keyIndex))
    try {
      const normalizedInterval = Math.max(1, Math.floor(testIntervalMs || 1000))
      const response = await testMultiKeys(
        currentRow.id,
        normalizedInterval,
        keyIndex
      )

      if (response.success) {
        const results = response.data || []
        setKeyTestResults((prev) => ({
          ...prev,
          ...Object.fromEntries(
            results.map((result) => [result.index, result])
          ),
        }))
        toast.success(response.message || t('Key test completed'))
        queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() })
        loadKeyStatus(currentPage, pageSize)
      } else {
        toast.error(response.message || t('Key test failed'))
      }
    } catch (error: unknown) {
      toast.error(error instanceof Error ? error.message : t('Key test failed'))
    } finally {
      setTestingKeyIndexes((prev) => {
        const next = new Set(prev)
        next.delete(keyIndex)
        return next
      })
    }
  }

  const performAction = async () => {
    if (!confirmAction || !currentRow) return

    setIsPerformingAction(true)
    try {
      const { type, keyIndex } = confirmAction
      let response

      // Execute the appropriate action
      if (type === 'enable' && keyIndex !== undefined) {
        response = await enableMultiKey(currentRow.id, keyIndex)
      } else if (type === 'disable' && keyIndex !== undefined) {
        response = await disableMultiKey(currentRow.id, keyIndex)
      } else if (type === 'delete' && keyIndex !== undefined) {
        response = await deleteMultiKey(currentRow.id, keyIndex)
      } else if (type === 'enable-all') {
        response = await enableAllMultiKeys(currentRow.id)
      } else if (type === 'disable-all') {
        response = await disableAllMultiKeys(currentRow.id)
      } else if (type === 'delete-disabled') {
        response = await deleteDisabledMultiKeys(currentRow.id)
      }

      if (response?.success) {
        toast.success(response.message || t('Operation successful'))
        queryClient.invalidateQueries({ queryKey: channelsQueryKeys.lists() })

        // Reload data - reset to page 1 for bulk actions
        const isBulkAction = type.includes('all') || type === 'delete-disabled'
        if (isBulkAction) {
          setCurrentPage(1)
          loadKeyStatus(1, pageSize)
        } else {
          loadKeyStatus(currentPage, pageSize)
        }
      } else {
        toast.error(response?.message || t('Operation failed'))
      }
    } catch (error: unknown) {
      toast.error(
        error instanceof Error ? error.message : t('Operation failed')
      )
    } finally {
      setIsPerformingAction(false)
      setConfirmAction(null)
    }
  }

  const renderStatusBadge = (status: number) => {
    const config = getMultiKeyStatusConfig(status)
    return (
      <StatusBadge
        label={t(config.label)}
        variant={config.variant}
        showDot
        copyable={false}
      />
    )
  }

  const formatKeyTimestamp = (timestamp?: number) => {
    if (!timestamp) return '-'
    return formatTimestamp(timestamp)
  }

  const renderTestResult = (result?: MultiKeyTestResult) => {
    if (!result) return <span className='text-muted-foreground'>-</span>

    if (result.success) {
      return (
        <StatusBadge
          label={`${t('Success')} (${result.response_time}ms)`}
          variant='success'
          showDot
          copyable={false}
        />
      )
    }

    return (
      <span className='text-destructive block max-w-48 truncate text-sm'>
        {result.message || t('Failed')}
      </span>
    )
  }

  if (!currentRow) return null

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className='flex max-h-[90vh] !w-[min(980px,calc(100vw-2rem))] !max-w-none flex-col sm:!max-w-none'>
          <DialogHeader>
            <DialogTitle className='flex items-center gap-2'>
              {t('Multi-Key Management')}
              <StatusBadge
                label={currentRow.name}
                variant='neutral'
                copyable={false}
              />
              {currentRow.channel_info?.multi_key_mode && (
                <StatusBadge
                  label={
                    currentRow.channel_info.multi_key_mode === 'random'
                      ? t('Random')
                      : t('Polling')
                  }
                  variant='neutral'
                  copyable={false}
                />
              )}
            </DialogTitle>
            <DialogDescription>
              {t('Manage multi-key status and configuration for this channel')}
            </DialogDescription>
          </DialogHeader>

          <div className='flex min-h-0 flex-1 flex-col space-y-4 overflow-hidden'>
            {/* Statistics */}
            <div className='grid shrink-0 grid-cols-3 gap-3'>
              <StatisticsCard
                label={t('Enabled')}
                count={enabledCount}
                total={total}
              />
              <StatisticsCard
                label={t('Manual Disabled')}
                count={manualDisabledCount}
                total={total}
              />
              <StatisticsCard
                label={t('Auto Disabled')}
                count={autoDisabledCount}
                total={total}
              />
            </div>

            <Separator className='shrink-0' />

            {/* Toolbar */}
            <div className='flex shrink-0 flex-col gap-2 xl:flex-row xl:items-center xl:justify-between'>
              <div className='flex flex-wrap items-center gap-2'>
                <Select
                  items={[
                    ...MULTI_KEY_FILTER_OPTIONS.map((option) => ({
                      value: option.value,
                      label: t(option.label),
                    })),
                  ]}
                  value={
                    statusFilter === null ? 'all' : statusFilter.toString()
                  }
                  onValueChange={(v) =>
                    v !== null && handleStatusFilterChange(v)
                  }
                >
                  <SelectTrigger className='w-40'>
                    <SelectValue placeholder={t('All Status')} />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {MULTI_KEY_FILTER_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {t(option.label)}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>

              <div className='flex flex-wrap items-center justify-start gap-2 xl:justify-end'>
                <Input
                  type='number'
                  min={1}
                  className='w-28'
                  value={testIntervalMs}
                  onChange={(event) =>
                    setTestIntervalMs(Number(event.target.value) || 1000)
                  }
                />
                <span className='text-muted-foreground text-sm'>ms</span>
                <Button
                  variant='default'
                  size='sm'
                  onClick={handleTestKeys}
                  disabled={isLoading || isTestingKeys || keys.length === 0}
                >
                  {isTestingKeys ? (
                    <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  ) : (
                    <FlaskConical className='mr-2 h-4 w-4' />
                  )}
                  {t('Test All')}
                </Button>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => loadKeyStatus()}
                  disabled={isLoading}
                >
                  <RefreshCw className='h-4 w-4' />
                </Button>

                {manualDisabledCount + autoDisabledCount > 0 && (
                  <Button
                    variant='default'
                    size='sm'
                    onClick={() => setConfirmAction({ type: 'enable-all' })}
                  >
                    <Power className='mr-2 h-4 w-4' />
                    {t('Enable All')}
                  </Button>
                )}

                {enabledCount > 0 && (
                  <Button
                    variant='destructive'
                    size='sm'
                    onClick={() => setConfirmAction({ type: 'disable-all' })}
                  >
                    <PowerOff className='mr-2 h-4 w-4' />
                    {t('Disable All')}
                  </Button>
                )}

                {autoDisabledCount > 0 && (
                  <Button
                    variant='destructive'
                    size='sm'
                    onClick={() =>
                      setConfirmAction({ type: 'delete-disabled' })
                    }
                  >
                    <Trash2 className='mr-2 h-4 w-4' />
                    {t('Delete Auto-Disabled')}
                  </Button>
                )}
              </div>
            </div>

            {/* Table */}
            <div className='min-h-0 min-w-0 flex-1 overflow-auto rounded-md border'>
              {isLoading ? (
                <div className='flex items-center justify-center py-12'>
                  <Loader2 className='text-muted-foreground h-8 w-8 animate-spin' />
                </div>
              ) : keys.length === 0 ? (
                <div className='text-muted-foreground py-12 text-center'>
                  {t('No keys found')}
                </div>
              ) : (
                <div className='min-w-[980px]'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className='w-12'>{t('Index')}</TableHead>
                        <TableHead className='w-24'>
                          {t('Key Preview')}
                        </TableHead>
                        <TableHead className='w-20'>{t('Status')}</TableHead>
                        <TableHead className='w-48'>
                          {t('Disabled Reason')}
                        </TableHead>
                        <TableHead className='w-36'>
                          {t('Disabled Time')}
                        </TableHead>
                        <TableHead className='w-24'>
                          {t('Test Result')}
                        </TableHead>
                        <TableHead className='w-48 text-right'>
                          {t('Actions')}
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {keys.map((key) => (
                        <TableRow key={key.index}>
                          <TableCell className='font-mono text-sm'>
                            #{key.index + 1}
                          </TableCell>
                          <TableCell className='font-mono text-sm'>
                            {key.key_preview || '-'}
                          </TableCell>
                          <TableCell>{renderStatusBadge(key.status)}</TableCell>
                          <TableCell className='max-w-48 truncate text-sm'>
                            {key.reason || '-'}
                          </TableCell>
                          <TableCell className='text-muted-foreground text-sm whitespace-nowrap'>
                            {formatKeyTimestamp(key.disabled_time)}
                          </TableCell>
                          <TableCell>
                            {renderTestResult(keyTestResults[key.index])}
                          </TableCell>
                          <TableCell className='text-right'>
                            <MultiKeyTableRowActions
                              keyIndex={key.index}
                              status={key.status}
                              isTesting={testingKeyIndexes.has(key.index)}
                              onAction={setConfirmAction}
                              onTest={handleTestKey}
                            />
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className='flex shrink-0 items-center justify-between'>
                <div className='text-muted-foreground text-sm'>
                  {t('Page {{current}} of {{total}}', {
                    current: currentPage,
                    total: totalPages,
                  })}
                </div>
                <div className='flex gap-2'>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => handlePageChange(currentPage - 1)}
                    disabled={currentPage === 1 || isLoading}
                  >
                    {t('Previous')}
                  </Button>
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => handlePageChange(currentPage + 1)}
                    disabled={currentPage >= totalPages || isLoading}
                  >
                    {t('Next')}
                  </Button>
                </div>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Confirmation Dialog */}
      <ConfirmDialog
        open={confirmAction !== null}
        onOpenChange={(open) => !open && setConfirmAction(null)}
        title={t('Confirm Action')}
        desc={t(getMultiKeyConfirmMessage(confirmAction))}
        destructive={isDestructiveAction(confirmAction)}
        isLoading={isPerformingAction}
        handleConfirm={performAction}
      />
    </>
  )
}
