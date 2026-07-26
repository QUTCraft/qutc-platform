import { get, post } from '@/api/client'
import type { Invitation, InvitationAcceptance } from '@/api/types'

export const invitationApi = {
  get: (token: string) => get<Invitation>(`/api/v1/invitations/${encodeURIComponent(token)}`),
  accept: (token: string) => post<InvitationAcceptance>(`/api/v1/invitations/${encodeURIComponent(token)}/accept`),
}
