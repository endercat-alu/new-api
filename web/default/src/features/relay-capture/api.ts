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
import { api } from '@/lib/api'
import type {
  ApiResponse,
  RelayCaptureRecordsPage,
  RelayCaptureState,
} from './types'

export const relayCaptureQueryKeys = {
  all: ['relay-capture'] as const,
  status: () => [...relayCaptureQueryKeys.all, 'status'] as const,
  records: (page: number, pageSize: number) =>
    [...relayCaptureQueryKeys.all, 'records', page, pageSize] as const,
}

export async function getRelayCaptureStatus(): Promise<
  ApiResponse<RelayCaptureState>
> {
  const res = await api.get('/api/relay-capture/status')
  return res.data
}

export async function startRelayCapture(): Promise<
  ApiResponse<RelayCaptureState>
> {
  const res = await api.post('/api/relay-capture/start')
  return res.data
}

export async function stopRelayCapture(): Promise<
  ApiResponse<RelayCaptureState>
> {
  const res = await api.post('/api/relay-capture/stop')
  return res.data
}

export async function clearRelayCaptureRecords(): Promise<
  ApiResponse<RelayCaptureState>
> {
  const res = await api.post('/api/relay-capture/clear')
  return res.data
}

export async function getRelayCaptureRecords(params: {
  page?: number
  pageSize?: number
}): Promise<ApiResponse<RelayCaptureRecordsPage>> {
  const res = await api.get('/api/relay-capture/records', {
    params: {
      p: params.page ?? 1,
      page_size: params.pageSize ?? 50,
    },
  })
  return res.data
}
