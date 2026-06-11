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
import { useMemo, useState, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  Bug,
  Loader2,
  Play,
  RefreshCw,
  Square,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { SectionPageLayout } from '@/components/layout'
import {
  clearRelayCaptureRecords,
  getRelayCaptureRecords,
  getRelayCaptureStatus,
  relayCaptureQueryKeys,
  startRelayCapture,
  stopRelayCapture,
} from './api'
import type { RelayCaptureRecord } from './types'

const PAGE_SIZE = 50

export function RelayCapture() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [clearOpen, setClearOpen] = useState(false)

  const statusQuery = useQuery({
    queryKey: relayCaptureQueryKeys.status(),
    queryFn: getRelayCaptureStatus,
    refetchInterval: 3000,
  })
  const status = statusQuery.data?.data

  const recordsQuery = useQuery({
    queryKey: relayCaptureQueryKeys.records(1, PAGE_SIZE),
    queryFn: () => getRelayCaptureRecords({ page: 1, pageSize: PAGE_SIZE }),
    refetchInterval: status?.enabled ? 3000 : false,
  })
  const records = useMemo(
    () => recordsQuery.data?.data?.items ?? [],
    [recordsQuery.data?.data?.items]
  )
  const selectedRecord = useMemo(
    () => records.find((record) => record.id === selectedId) ?? records[0],
    [records, selectedId]
  )

  const invalidateCapture = async () => {
    await queryClient.invalidateQueries({ queryKey: relayCaptureQueryKeys.all })
  }

  const startMutation = useMutation({
    mutationFn: startRelayCapture,
    onSuccess: async (res) => {
      if (res.success) toast.success(t('Capture started'))
      await invalidateCapture()
    },
  })

  const stopMutation = useMutation({
    mutationFn: stopRelayCapture,
    onSuccess: async (res) => {
      if (res.success) toast.success(t('Capture stopped'))
      await invalidateCapture()
    },
  })

  const clearMutation = useMutation({
    mutationFn: clearRelayCaptureRecords,
    onSuccess: async (res) => {
      if (res.success) {
        toast.success(t('Captured records cleared'))
        setSelectedId(null)
        setClearOpen(false)
      }
      await invalidateCapture()
    },
  })

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('AI Traffic Capture')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          onClick={() => {
            void invalidateCapture()
          }}
          disabled={statusQuery.isFetching || recordsQuery.isFetching}
        >
          <RefreshCw
            data-icon='inline-start'
            className={
              statusQuery.isFetching || recordsQuery.isFetching
                ? 'animate-spin'
                : undefined
            }
          />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <Alert variant='destructive'>
            <AlertTriangle />
            <AlertTitle>{t('Sensitive capture warning')}</AlertTitle>
            <AlertDescription>
              {t(
                'This feature may capture sensitive data such as Authorization headers, API keys, user prompts, and model outputs. Use it only for short troubleshooting sessions.'
              )}
            </AlertDescription>
          </Alert>

          <div className='grid gap-4 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]'>
            <Card>
              <CardHeader>
                <CardTitle>{t('Capture Control')}</CardTitle>
                <CardDescription>
                  {t(
                    'Records are stored in server memory only and will be lost after server restart.'
                  )}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className='flex flex-col gap-4'>
                  <div className='grid gap-3 sm:grid-cols-2'>
                    <StatusItem
                      label={t('Status')}
                      value={
                        status?.enabled
                          ? t('Capture is running')
                          : t('Capture is stopped')
                      }
                      badge={
                        <Badge
                          variant={status?.enabled ? 'default' : 'secondary'}
                        >
                          {status?.enabled ? t('Running') : t('Stopped')}
                        </Badge>
                      }
                    />
                    <StatusItem
                      label={t('Records')}
                      value={formatNumber(status?.record_count ?? 0)}
                    />
                    <StatusItem
                      label={t('Started At')}
                      value={formatTimestamp(status?.started_at, t)}
                    />
                    <StatusItem
                      label={t('Stopped At')}
                      value={formatTimestamp(status?.stopped_at, t)}
                    />
                    <StatusItem
                      label={t('Max Records')}
                      value={formatNumber(status?.max_records ?? 0)}
                    />
                    <StatusItem
                      label={t('Memory Used')}
                      value={formatBytes(status?.total_bytes ?? 0)}
                    />
                    <StatusItem
                      label={t('Max Body Size')}
                      value={formatBytes(status?.max_body_bytes ?? 0)}
                    />
                    <StatusItem
                      label={t('Max Memory')}
                      value={formatBytes(status?.max_total_bytes ?? 0)}
                    />
                  </div>

                  <div className='flex flex-wrap gap-2'>
                    <Button
                      onClick={() => startMutation.mutate()}
                      disabled={
                        Boolean(status?.enabled) || startMutation.isPending
                      }
                    >
                      {startMutation.isPending ? (
                        <Loader2
                          data-icon='inline-start'
                          className='animate-spin'
                        />
                      ) : (
                        <Play data-icon='inline-start' />
                      )}
                      {t('Start Capture')}
                    </Button>
                    <Button
                      variant='outline'
                      onClick={() => stopMutation.mutate()}
                      disabled={!status?.enabled || stopMutation.isPending}
                    >
                      {stopMutation.isPending ? (
                        <Loader2
                          data-icon='inline-start'
                          className='animate-spin'
                        />
                      ) : (
                        <Square data-icon='inline-start' />
                      )}
                      {t('Stop Capture')}
                    </Button>
                    <AlertDialog open={clearOpen} onOpenChange={setClearOpen}>
                      <AlertDialogTrigger
                        render={
                          <Button
                            variant='destructive'
                            disabled={
                              clearMutation.isPending || !status?.record_count
                            }
                          />
                        }
                      >
                        {clearMutation.isPending ? (
                          <Loader2
                            data-icon='inline-start'
                            className='animate-spin'
                          />
                        ) : (
                          <Trash2 data-icon='inline-start' />
                        )}
                        {t('Clear Records')}
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>
                            {t('Clear all captured records?')}
                          </AlertDialogTitle>
                          <AlertDialogDescription>
                            {t(
                              'This permanently removes the temporary in-memory capture records to free space.'
                            )}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel disabled={clearMutation.isPending}>
                            {t('Cancel')}
                          </AlertDialogCancel>
                          <AlertDialogAction
                            variant='destructive'
                            disabled={clearMutation.isPending}
                            onClick={(event) => {
                              event.preventDefault()
                              clearMutation.mutate()
                            }}
                          >
                            {t('Clear Records')}
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{t('Recorded Requests')}</CardTitle>
                <CardDescription>
                  {t('Latest captured AI relay traffic is shown first.')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {records.length === 0 ? (
                  <Empty>
                    <EmptyHeader>
                      <EmptyMedia variant='icon'>
                        <Bug />
                      </EmptyMedia>
                      <EmptyTitle>{t('No records captured')}</EmptyTitle>
                      <EmptyDescription>
                        {t(
                          'Start capture and send AI requests to collect records.'
                        )}
                      </EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Time')}</TableHead>
                        <TableHead>{t('Method')}</TableHead>
                        <TableHead>{t('Path')}</TableHead>
                        <TableHead>{t('Status')}</TableHead>
                        <TableHead>{t('Duration')}</TableHead>
                        <TableHead>{t('Size')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {records.map((record) => (
                        <TableRow
                          key={record.id}
                          data-state={
                            selectedRecord?.id === record.id
                              ? 'selected'
                              : undefined
                          }
                          className='cursor-pointer'
                          onClick={() => setSelectedId(record.id)}
                        >
                          <TableCell>
                            {formatTimeOnly(record.started_at)}
                          </TableCell>
                          <TableCell>
                            <Badge variant='outline'>
                              {record.request.method}
                            </Badge>
                          </TableCell>
                          <TableCell className='max-w-56 truncate'>
                            {record.request.path}
                          </TableCell>
                          <TableCell>
                            <StatusBadge
                              statusCode={record.response?.status_code}
                            />
                          </TableCell>
                          <TableCell>
                            {formatDuration(record.duration_ms)}
                          </TableCell>
                          <TableCell>
                            {formatBytes(record.stored_bytes)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </div>

          {selectedRecord && <RecordDetail record={selectedRecord} />}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function StatusItem(props: {
  label: string
  value: string
  badge?: ReactNode
}) {
  return (
    <div className='bg-muted/40 flex min-w-0 flex-col gap-1 rounded-lg p-3'>
      <div className='text-muted-foreground text-xs font-medium'>
        {props.label}
      </div>
      <div className='flex min-w-0 items-center justify-between gap-2'>
        <span className='truncate text-sm font-medium'>{props.value}</span>
        {props.badge}
      </div>
    </div>
  )
}

function StatusBadge(props: { statusCode?: number }) {
  const { t } = useTranslation()
  if (!props.statusCode)
    return <Badge variant='secondary'>{t('Pending')}</Badge>
  if (props.statusCode >= 400) {
    return <Badge variant='destructive'>{props.statusCode}</Badge>
  }
  return <Badge variant='secondary'>{props.statusCode}</Badge>
}

function RecordDetail({ record }: { record: RelayCaptureRecord }) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Capture Detail')}</CardTitle>
        <CardDescription>
          {record.request.method} {record.request.url}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className='grid gap-4 xl:grid-cols-2'>
          <DetailSection
            title={t('Request Headers')}
            value={formatJson(record.request.headers)}
          />
          <DetailSection
            title={t('Response Headers')}
            value={formatJson(record.response?.headers ?? {})}
          />
          <DetailSection
            title={t('Request Body')}
            value={record.request.body || t('No body captured')}
            meta={buildBodyMeta(
              record.request.body_base64,
              record.request.body_truncated,
              t
            )}
          />
          <DetailSection
            title={t('Response Body')}
            value={record.response?.body || t('No body captured')}
            meta={buildBodyMeta(
              record.response?.body_base64,
              record.response?.body_truncated,
              t
            )}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function DetailSection(props: { title: string; value: string; meta?: string }) {
  return (
    <div className='flex min-w-0 flex-col gap-2'>
      <div className='flex items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{props.title}</h3>
        {props.meta && <Badge variant='outline'>{props.meta}</Badge>}
      </div>
      <pre className='bg-muted/60 max-h-80 overflow-auto rounded-lg p-3 text-xs break-words whitespace-pre-wrap'>
        {props.value}
      </pre>
    </div>
  )
}

function buildBodyMeta(
  base64: boolean | undefined,
  truncated: boolean | undefined,
  t: (key: string) => string
) {
  const tags = []
  if (base64) tags.push(t('Base64 encoded'))
  if (truncated) tags.push(t('Truncated'))
  return tags.join(' · ')
}

function formatJson(value: unknown) {
  return JSON.stringify(value, null, 2)
}

function formatNumber(value: number) {
  return new Intl.NumberFormat().format(value)
}

function formatTimestamp(
  value: number | undefined,
  t: (key: string) => string
) {
  if (!value) return t('Never')
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(new Date(value))
}

function formatTimeOnly(value: number) {
  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(value))
}

function formatDuration(value: number | undefined) {
  if (!value) return '-'
  return `${value} ms`
}

function formatBytes(value: number) {
  if (!value) return '0 B'
  const units = ['B', 'KiB', 'MiB', 'GiB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit += 1
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
}
