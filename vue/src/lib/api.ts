import axios from 'axios'

// api is the HTTP client for the Go server. Every path is relative to /api/v1.
export const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30_000,
})
