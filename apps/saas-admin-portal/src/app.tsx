import { useQuery } from '@tanstack/react-query'
import { Activity, Building2, LayoutDashboard, ShieldCheck, Users } from 'lucide-react'
import { getSession, getUsers } from './api'

const navigation = [
  { label: 'Overview', icon: LayoutDashboard, active: true },
  { label: 'Organizations', icon: Building2 },
  { label: 'Access control', icon: ShieldCheck },
  { label: 'Users', icon: Users },
  { label: 'Activity', icon: Activity },
]

export function App() {
  const session = useQuery({ queryKey: ['session'], queryFn: getSession })
  const users = useQuery({ queryKey: ['users'], queryFn: getUsers })

  if (session.isPending) return <main className='status-screen'>Restoring your session...</main>
  if (session.isError) return <main className='status-screen'>Redirecting to sign in...</main>

  return (
    <div className='app-shell'>
      <aside className='sidebar'>
        <div className='brand'>
          <span className='brand-mark'>S</span>
          <span>SaaS Control Plane</span>
        </div>
        <nav>
          {navigation.map(({ label, icon: Icon, active }) => (
            <a className={active ? 'nav-item active' : 'nav-item'} href='/' key={label}>
              <Icon size={18} />
              {label}
            </a>
          ))}
        </nav>
        <form action='/bff/logout' method='post' className='sign-out'>
          <button type='submit'>Sign out</button>
        </form>
      </aside>
      <main className='content'>
        <header className='topbar'>
          <div>
            <p className='eyebrow'>Control plane</p>
            <h1>Organization overview</h1>
          </div>
          <div className='profile'>
            <strong>{session.data.name ?? session.data.email}</strong>
            <span>{session.data.organizationId}</span>
          </div>
        </header>
        <section className='metrics' aria-label='Organization metrics'>
          <article className='metric-card'>
            <p>Active organization</p>
            <strong>Connected</strong>
            <span>Tenant context is verified by the BFF.</span>
          </article>
          <article className='metric-card'>
            <p>Application users</p>
            <strong>{users.data?.length ?? '-'}</strong>
            <span>{users.isError ? 'Permission unavailable' : 'From user-service'}</span>
          </article>
          <article className='metric-card'>
            <p>Authorization</p>
            <strong>Protected</strong>
            <span>Trusted headers never leave the BFF.</span>
          </article>
        </section>
        <section className='panel'>
          <div className='panel-heading'>
            <div>
              <p className='eyebrow'>Tenant directory</p>
              <h2>Recent application users</h2>
            </div>
            <span className='badge'>BFF proxy</span>
          </div>
          {users.isPending ? <p className='muted'>Loading users...</p> : null}
          {users.isError ? <p className='muted'>You do not have permission to list application users.</p> : null}
          {users.data ? (
            <table>
              <thead><tr><th>Name</th><th>Email</th><th>Created</th></tr></thead>
              <tbody>{users.data.map((user) => <tr key={user.id}><td>{user.name}</td><td>{user.email}</td><td>{new Date(user.createdAt).toLocaleDateString()}</td></tr>)}</tbody>
            </table>
          ) : null}
        </section>
      </main>
    </div>
  )
}
