import axios, { isAxiosError } from 'axios'

// api is the HTTP client for the Go server. Every path is relative to /api/v1.
export const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30_000,
})

// ApiErrorBody mirrors errorBody in pkg/httpkit/httpkit.go: the body of every
// error answer.
export interface ApiErrorBody {
  code: string
  message: string
}

// apiErrorCode returns the code of an error answer, or '' when err carries
// none (a transport failure, for example).
export function apiErrorCode(err: unknown): string {
  if (isAxiosError<Partial<ApiErrorBody>>(err)) {
    return err.response?.data?.code ?? ''
  }
  return ''
}
