import { api } from '@/lib/api'

// Member mirrors memberResponse in core/admin_handler.go.
export interface Member {
  id: number
  login: string
  password_temporary: boolean
  first_name: string | null
  last_name: string | null
  email: string | null
  phone: string | null
  last_login_at: string | null
}

// Organization mirrors organizationResponse in core/admin_handler.go.
export interface Organization {
  id: number
  name: string
  deactivated_at: string | null
  created_at: string
  device_count: number
  last_uplink_at: string | null
  users: Member[]
}

// CreatedOrganization mirrors createOrganizationResponse in
// core/admin_handler.go.
export interface CreatedOrganization {
  organization: Organization
  temporary_password: string
}

// useOrganizations loads the admin list of organizations when the component
// mounts, and holds the admin actions. Each action reloads the list.
export function useOrganizations() {
  const organizations = shallowRef<Organization[]>([])
  const loading = shallowRef(false)
  const error = shallowRef<unknown>()

  async function reload() {
    loading.value = true
    error.value = undefined
    try {
      const { data } = await api.get<{ organizations: Organization[] }>('/admin/organizations')
      organizations.value = data.organizations
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  async function create(name: string, login: string): Promise<CreatedOrganization> {
    const { data } = await api.post<CreatedOrganization>('/admin/organizations', { name, login })
    await reload()
    return data
  }

  async function resetPassword(userId: number): Promise<string> {
    const { data } = await api.post<{ temporary_password: string }>(
      `/admin/users/${userId}/reset-password`,
    )
    await reload()
    return data.temporary_password
  }

  async function setActive(organizationId: number, active: boolean) {
    const action = active ? 'reactivate' : 'deactivate'
    await api.post(`/admin/organizations/${organizationId}/${action}`)
    await reload()
  }

  onMounted(reload)

  return { organizations, loading, error, reload, create, resetPassword, setActive }
}
