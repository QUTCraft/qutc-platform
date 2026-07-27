import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { resolveApiUrl } from '@/api/client'
import { portalBase } from '@/api/portal'

const absoluteAdminAssetPattern = /https?:\/\/[^)\s]+?\/api\/v1\/admin\/assets\/([a-z0-9-]+)\/download\b/gi
const relativeAdminAssetPattern = /\/api\/v1\/admin\/assets\/([a-z0-9-]+)\/download\b/gi
const absolutePortalAssetPattern = /https?:\/\/[^)\s]+?\/api\/v1\/portal\/organizations\/([a-z0-9-]+)\/assets\/([a-z0-9-]+)\/download\b/gi
const relativePortalAssetPattern = /\/api\/v1\/portal\/organizations\/([a-z0-9-]+)\/assets\/([a-z0-9-]+)\/download\b/gi

/**
 * Markdown saved by the CMS contains protected admin asset URLs. Public
 * content must reference the corresponding organization-scoped portal URL.
 */
export function rewritePublicAssetUrls(markdown: string): string {
  const publicAssetUrl = (assetID: string) => resolveApiUrl(`${portalBase}/assets/${assetID}/download`)
  const portalAssetUrl = (slug: string, assetID: string) => resolveApiUrl(`/api/v1/portal/organizations/${slug}/assets/${assetID}/download`)
  return markdown
    .replace(absoluteAdminAssetPattern, (_match, assetID: string) => publicAssetUrl(assetID))
    .replace(relativeAdminAssetPattern, (_match, assetID: string) => publicAssetUrl(assetID))
    .replace(absolutePortalAssetPattern, (_match, slug: string, assetID: string) => portalAssetUrl(slug, assetID))
    .replace(relativePortalAssetPattern, (_match, slug: string, assetID: string) => portalAssetUrl(slug, assetID))
}

export function renderMarkdown(markdown: string, publicAssets = false): string {
  const source = publicAssets ? rewritePublicAssetUrls(markdown) : markdown
  const html = marked.parse(source, { gfm: true, breaks: true, async: false })
  return DOMPurify.sanitize(html, { ADD_ATTR: ['target', 'rel'] })
}
