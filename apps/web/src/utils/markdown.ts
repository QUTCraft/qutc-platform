import DOMPurify from 'dompurify'
import { Renderer, marked, type Tokens } from 'marked'
import { resolveApiUrl } from '@/api/client'
import { portalBase } from '@/api/portal'

const absoluteAdminAssetPattern = /https?:\/\/[^)\s]+?\/api\/v1\/admin\/assets\/([a-z0-9-]+)\/download\b/gi
const relativeAdminAssetPattern = /\/api\/v1\/admin\/assets\/([a-z0-9-]+)\/download\b/gi
const absolutePortalAssetPattern = /https?:\/\/[^)\s]+?\/api\/v1\/portal\/organizations\/([a-z0-9-]+)\/assets\/([a-z0-9-]+)\/download\b/gi
const relativePortalAssetPattern = /\/api\/v1\/portal\/organizations\/([a-z0-9-]+)\/assets\/([a-z0-9-]+)\/download\b/gi
const automaticFoldLineThreshold = 12

type CodeFoldMode = 'auto' | 'collapsed' | 'expanded' | 'disabled'

interface MarkdownDocument {
  source: string
  codeFold: CodeFoldMode
}

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

function unquote(value: string): string {
  const trimmed = value.trim()
  if (trimmed.length >= 2 && ((trimmed.startsWith('"') && trimmed.endsWith('"')) || (trimmed.startsWith("'") && trimmed.endsWith("'")))) {
    return trimmed.slice(1, -1).trim()
  }
  return trimmed
}

function parseCodeFoldMode(value: string | undefined): CodeFoldMode | undefined {
  if (!value) return undefined
  switch (unquote(value).toLowerCase()) {
    case 'true':
    case 'fold':
    case 'hide':
    case 'collapsed':
      return 'collapsed'
    case 'show':
    case 'open':
    case 'expanded':
      return 'expanded'
    case 'false':
    case 'off':
    case 'none':
    case 'disabled':
      return 'disabled'
    case 'auto':
      return 'auto'
    default:
      return undefined
  }
}

function codeFoldMetadata(mode: CodeFoldMode): string {
  if (mode === 'collapsed') return 'true'
  if (mode === 'expanded') return 'show'
  if (mode === 'disabled') return 'false'
  return 'auto'
}

/**
 * Convert the author-friendly wrapper syntax into fenced-code metadata before
 * Marked tokenizes the document. Invalid or incomplete wrappers are retained
 * as ordinary Markdown instead of swallowing article content.
 */
function applyCodeFoldWrappers(markdown: string): string {
  const lines = markdown.split(/\r?\n/)
  const output: string[] = []

  for (let index = 0; index < lines.length; index += 1) {
    const wrapper = lines[index].trim().match(/^<code-fold\s*:\s*([^>]+)>$/i)
    const wrapperMode = parseCodeFoldMode(wrapper?.[1])
    if (!wrapper || !wrapperMode) {
      output.push(lines[index])
      continue
    }

    let fenceIndex = index + 1
    while (fenceIndex < lines.length && !lines[fenceIndex].trim()) fenceIndex += 1
    const openingFence = lines[fenceIndex]?.match(/^(\s*)(`{3,}|~{3,})(.*)$/)
    if (!openingFence) {
      output.push(lines[index])
      continue
    }

    const fenceCharacter = openingFence[2][0]
    const closingFencePattern = new RegExp(`^\\s*${fenceCharacter}{${openingFence[2].length},}\\s*$`)
    let closingFenceIndex = fenceIndex + 1
    while (closingFenceIndex < lines.length && !closingFencePattern.test(lines[closingFenceIndex])) closingFenceIndex += 1

    let closingWrapperIndex = closingFenceIndex + 1
    while (closingWrapperIndex < lines.length && !lines[closingWrapperIndex].trim()) closingWrapperIndex += 1
    if (closingFenceIndex >= lines.length || !/^<\/code-fold\s*>$/i.test(lines[closingWrapperIndex]?.trim() ?? '')) {
      output.push(lines[index])
      continue
    }

    for (let skipped = index + 1; skipped < fenceIndex; skipped += 1) output.push(lines[skipped])
    const fenceInfo = openingFence[3].trim()
    output.push(`${openingFence[1]}${openingFence[2]}${fenceInfo ? `${fenceInfo} ` : ''}{code-fold=${codeFoldMetadata(wrapperMode)}}`)
    for (let bodyIndex = fenceIndex + 1; bodyIndex <= closingFenceIndex; bodyIndex += 1) output.push(lines[bodyIndex])
    for (let skipped = closingFenceIndex + 1; skipped < closingWrapperIndex; skipped += 1) output.push(lines[skipped])
    index = closingWrapperIndex
  }

  return output.join('\n')
}

function parseMarkdownDocument(markdown: string): MarkdownDocument {
  const source = markdown.replace(/^\uFEFF/, '')
  const frontMatter = source.match(/^---[ \t]*\r?\n([\s\S]*?)\r?\n---[ \t]*(?:\r?\n|$)/)
  let codeFold: CodeFoldMode = 'auto'
  let body = source

  if (frontMatter) {
    for (const line of frontMatter[1].split(/\r?\n/)) {
      const setting = line.match(/^\s*code-fold\s*:\s*(.*?)\s*$/i)
      const parsed = parseCodeFoldMode(setting?.[1])
      if (parsed) codeFold = parsed
    }
    body = source.slice(frontMatter[0].length)
  }

  return { source: applyCodeFoldWrappers(body), codeFold }
}

function parseFenceInfo(info: string | undefined): { language: string; codeFold?: CodeFoldMode } {
  const metadataPattern = /(?:\{\s*)?code-fold\s*(?:=|:)\s*(true|false|auto|show|open|fold|hide|collapsed|expanded|off|none|disabled)(?:\s*\})?/i
  const metadata = info?.match(metadataPattern)
  const cleaned = (info ?? '').replace(metadataPattern, '').replace(/\{\s*\}/g, '').trim()
  const language = cleaned.match(/^[a-z0-9_+#.-]+/i)?.[0] ?? ''
  return { language, codeFold: parseCodeFoldMode(metadata?.[1]) }
}

function codeBlockHtml(token: Tokens.Code, documentMode: CodeFoldMode): string {
  const code = token.text.replace(/\r\n/g, '\n').replace(/\n$/, '')
  const lineCount = code ? code.split('\n').length : 1
  const { language, codeFold } = parseFenceInfo(token.lang)
  const requestedMode = codeFold ?? documentMode
  const mode = requestedMode === 'auto'
    ? (lineCount >= automaticFoldLineThreshold ? 'collapsed' : 'disabled')
    : requestedMode
  const languageClass = language ? ` class="language-${escapeHtml(language)}"` : ''
  const languageLabel = language || 'CODE'
  const escapedCode = escapeHtml(code)
  const toolbar = `<div class="markdown-code-toolbar"><span class="markdown-code-language">${escapeHtml(languageLabel)}</span><span class="markdown-code-lines">${lineCount} 行</span><button type="button" class="markdown-copy-button" data-markdown-copy aria-label="复制代码"><span class="markdown-copy-label">复制</span></button></div>`
  const pre = `<pre><code${languageClass}>${escapedCode}</code></pre>`

  if (mode === 'disabled') {
    return `<div class="markdown-code-block" data-code-fold="disabled">${toolbar}${pre}</div>`
  }

  const open = mode === 'expanded' ? ' open' : ''
  const summary = `<summary><span class="markdown-fold-closed">展开代码</span><span class="markdown-fold-open">收起代码</span><span class="markdown-fold-lines">${lineCount} 行</span></summary>`
  return `<div class="markdown-code-block" data-code-fold="${mode}">${toolbar}<details${open}>${summary}${pre}</details></div>`
}

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
  const rewritten = publicAssets ? rewritePublicAssetUrls(markdown) : markdown
  const document = parseMarkdownDocument(rewritten)
  const renderer = new Renderer()
  renderer.code = (token) => codeBlockHtml(token, document.codeFold)
  const html = marked.parse(document.source, { gfm: true, breaks: true, async: false, renderer })
  return DOMPurify.sanitize(html, {
    ADD_ATTR: ['target', 'rel', 'open', 'aria-label', 'data-markdown-copy', 'data-code-fold'],
  })
}
