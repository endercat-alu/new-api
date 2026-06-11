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
export type ApiResponse<T = unknown> = {
  success: boolean
  message?: string
  data?: T
}

export type RelayCaptureState = {
  enabled: boolean
  started_at?: number
  stopped_at?: number
  record_count: number
  max_records: number
  max_body_bytes: number
  max_total_bytes: number
  total_bytes: number
}

export type RelayCaptureRequest = {
  method: string
  url: string
  path: string
  query?: string
  proto?: string
  host?: string
  remote_addr?: string
  headers: Record<string, string[]>
  body?: string
  body_base64?: boolean
  body_truncated?: boolean
  body_bytes: number
  captured_body_bytes: number
}

export type RelayCaptureResponse = {
  status_code: number
  headers: Record<string, string[]>
  body?: string
  body_base64?: boolean
  body_truncated?: boolean
  body_bytes: number
  captured_body_bytes: number
}

export type RelayCaptureRecord = {
  id: string
  started_at: number
  ended_at?: number
  duration_ms?: number
  route?: string
  request: RelayCaptureRequest
  response?: RelayCaptureResponse
  error?: string
  truncated: boolean
  stored_bytes: number
}

export type RelayCaptureRecordsPage = {
  page: number
  page_size: number
  total: number
  items: RelayCaptureRecord[]
}
