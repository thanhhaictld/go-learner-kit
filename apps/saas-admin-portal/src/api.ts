export type Session = {
  subject: string
  name: string | null
  email: string | null
  organizationId: string
}

export type User = {
  id: string
  name: string
  email: string
  createdAt: string
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, { ...init, credentials: 'same-origin' })
  if (response.status === 401) {
    window.location.assign(`/bff/login?returnUrl=${encodeURIComponent(window.location.pathname)}`)
    throw new Error('Authentication required')
  }
  if (!response.ok) throw new Error(`Request failed with status ${response.status}`)
  return response.json() as Promise<T>
}

export const getSession = () => request<Session>('/bff/user')
export const getUsers = () => request<User[]>('/api/users')
