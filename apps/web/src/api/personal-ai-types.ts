export interface PersonalAIConfiguration {
  base_url: string
  model: string
  api_key_configured: boolean
  organization_available: boolean
}
export interface ChatMessage { role: 'user' | 'assistant'; content: string }
export interface EditorChatInput { source: 'personal' | 'organization'; messages: ChatMessage[]; article?: string }
export interface EditorChatResult { markdown: string; source: string; model: string; request_id: string }
