export interface ApiEnvelope<T> { data: T; request_id: string }
export interface ApiError { code: string; message: string; field_errors: Array<{ field: string; rule: string; message: string }>; request_id: string }

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, { ...init, headers: { 'Content-Type': 'application/json', 'X-Demo-Actor': 'demo-supervisor', ...init.headers } })
  const body = await response.json()
  if (!response.ok) throw body as ApiError
  return (body as ApiEnvelope<T>).data
}
