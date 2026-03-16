import { getSession } from '@/lib/session'
import { backendFetch, type User } from '@/lib/api'

export default async function UsersPage({
  searchParams,
}: {
  searchParams: { q?: string }
}) {
  const session = await getSession()

  // Only nitecore can see all users
  if (session.role !== 'nitecore') {
    return (
      <div className="p-8">
        <h1 className="text-2xl font-bold text-white mb-4">Users</h1>
        <p className="text-sm text-gray-400">nitecore role required to view user list.</p>
      </div>
    )
  }

  const q = searchParams.q ?? ''
  let users: User[] = []
  let err = ''

  try {
    const path = q ? `/profiles/users?q=${encodeURIComponent(q)}` : '/profiles/users'
    const data = await backendFetch<{ users: User[] }>(path, session.accessToken)
    users = data.users ?? []
  } catch (e: unknown) {
    err = (e as Error).message
  }

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold text-white mb-6">Users</h1>

      <form method="GET" className="mb-6 flex gap-3">
        <input
          type="search"
          name="q"
          defaultValue={q}
          placeholder="Search by email…"
          className="w-64 rounded-lg bg-gray-800 border border-gray-700 px-4 py-2 text-sm text-white
                     placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand-500"
        />
        <button
          type="submit"
          className="rounded-lg bg-gray-700 hover:bg-gray-600 px-4 py-2 text-sm text-white"
        >
          Search
        </button>
      </form>

      {err && (
        <div className="mb-4 rounded-lg bg-red-950/40 border border-red-800 px-4 py-3 text-sm text-red-300">
          {err}
        </div>
      )}

      <div className="rounded-xl border border-gray-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-900 text-gray-400">
            <tr>
              <th className="text-left px-4 py-3 font-medium">Email</th>
              <th className="text-left px-4 py-3 font-medium">Display name</th>
              <th className="text-left px-4 py-3 font-medium">Role</th>
              <th className="text-left px-4 py-3 font-medium">XP</th>
              <th className="text-left px-4 py-3 font-medium">Joined</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {users.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                  No users found.
                </td>
              </tr>
            )}
            {users.map(u => (
              <tr key={u.user_id} className="bg-gray-950 hover:bg-gray-900">
                <td className="px-4 py-3 text-white">{u.email}</td>
                <td className="px-4 py-3 text-gray-300">{u.display_name ?? '—'}</td>
                <td className="px-4 py-3">
                  <span className={`text-xs px-2 py-1 rounded-full ${
                    u.role === 'nitecore' ? 'bg-brand-500/20 text-brand-100' :
                    u.role === 'venue_admin' ? 'bg-blue-900/40 text-blue-300' :
                    u.role === 'bartender' || u.role === 'door_staff' ? 'bg-yellow-900/40 text-yellow-300' :
                    'bg-gray-800 text-gray-400'
                  }`}>
                    {u.role}
                  </span>
                </td>
                <td className="px-4 py-3 text-gray-400">{u.global_xp}</td>
                <td className="px-4 py-3 text-gray-400 text-xs">
                  {new Date(u.created_at).toLocaleDateString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
