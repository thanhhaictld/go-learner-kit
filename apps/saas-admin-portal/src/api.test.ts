import { afterEach, describe, expect, it, vi } from 'vitest'
import { getUsers } from './api'

describe('portal API', () => {
  afterEach(() => vi.restoreAllMocks())

  it('uses the BFF user route with same-origin credentials', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify([
      { id: 'user-1', name: 'Ada Lovelace', email: 'ada@example.test', createdAt: '2026-01-01T00:00:00Z' },
    ]), { status: 200 }))

    await expect(getUsers()).resolves.toHaveLength(1)
    expect(fetchMock).toHaveBeenCalledWith('/api/users', { credentials: 'same-origin' })
  })
})
