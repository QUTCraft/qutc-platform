import { readonly, shallowRef } from 'vue'
import { portalApi } from '@/api/portal'
import type { Organization } from '@/api/types'

const organization = shallowRef<Organization | null>(null)
let pendingRequest: Promise<Organization> | null = null

async function loadPortalOrganization() {
  if (organization.value) return organization.value
  if (!pendingRequest) {
    pendingRequest = portalApi.getOrganization()
      .then((result) => {
        organization.value = result
        return result
      })
      .finally(() => {
        pendingRequest = null
      })
  }
  return pendingRequest
}

export function usePortalIdentity() {
  return {
    organization: readonly(organization),
    loadPortalOrganization,
  }
}
