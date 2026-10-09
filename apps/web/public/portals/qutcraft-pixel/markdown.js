/* ============================================================================
 * markdown.js — 与平台 ContentDetailView 同款的安全 Markdown 渲染
 * marked.parse(gfm+breaks) → DOMPurify.sanitize；公开正文改写 admin 资源 URL。
 * ========================================================================== */
(function () {
  'use strict'

  var AUTOMATIC_FOLD_LINE_THRESHOLD = 12
  var absoluteAdminAssetPattern = /https?:\/\/[^)\s]+?\/api\/v1\/admin\/assets\/([a-z0-9-]+)\/download\b/gi
  var relativeAdminAssetPattern = /\/api\/v1\/admin\/assets\/([a-z0-9-]+)\/download\b/gi
  var absolutePortalAssetPattern = /https?:\/\/[^)\s]+?\/api\/v1\/portal\/organizations\/([a-z0-9-]+)\/assets\/([a-z0-9-]+)\/download\b/gi
  var relativePortalAssetPattern = /\/api\/v1\/portal\/organizations\/([a-z0-9-]+)\/assets\/([a-z0-9-]+)\/download\b/gi

  function escapeHtml(value) {
    return String(value == null ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;')
  }

  function unquote(value) {
    var trimmed = String(value || '').trim()
    if (trimmed.length >= 2 && (
      (trimmed.charAt(0) === '"' && trimmed.charAt(trimmed.length - 1) === '"') ||
      (trimmed.charAt(0) === "'" && trimmed.charAt(trimmed.length - 1) === "'")
    )) {
      return trimmed.slice(1, -1).trim()
    }
    return trimmed
  }

  function parseCodeFoldMode(value) {
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

  function codeFoldMetadata(mode) {
    if (mode === 'collapsed') return 'true'
    if (mode === 'expanded') return 'show'
    if (mode === 'disabled') return 'false'
    return 'auto'
  }

  function applyCodeFoldWrappers(markdown) {
    var lines = markdown.split(/\r?\n/)
    var output = []
    for (var index = 0; index < lines.length; index += 1) {
      var wrapper = lines[index].trim().match(/^<code-fold\s*:\s*([^>]+)>$/i)
      var wrapperMode = parseCodeFoldMode(wrapper && wrapper[1])
      if (!wrapper || !wrapperMode) {
        output.push(lines[index])
        continue
      }
      var fenceIndex = index + 1
      while (fenceIndex < lines.length && !lines[fenceIndex].trim()) fenceIndex += 1
      var openingFence = lines[fenceIndex] && lines[fenceIndex].match(/^(\s*)(`{3,}|~{3,})(.*)$/)
      if (!openingFence) {
        output.push(lines[index])
        continue
      }
      var fenceCharacter = openingFence[2].charAt(0)
      var closingFencePattern = new RegExp('^\\s*' + fenceCharacter + '{' + openingFence[2].length + ',}\\s*$')
      var closingFenceIndex = fenceIndex + 1
      while (closingFenceIndex < lines.length && !closingFencePattern.test(lines[closingFenceIndex])) closingFenceIndex += 1
      var closingWrapperIndex = closingFenceIndex + 1
      while (closingWrapperIndex < lines.length && !lines[closingWrapperIndex].trim()) closingWrapperIndex += 1
      var closingWrapper = lines[closingWrapperIndex] ? lines[closingWrapperIndex].trim() : ''
      if (closingFenceIndex >= lines.length || !/^<\/code-fold\s*>$/i.test(closingWrapper)) {
        output.push(lines[index])
        continue
      }
      var skipped
      for (skipped = index + 1; skipped < fenceIndex; skipped += 1) output.push(lines[skipped])
      var fenceInfo = openingFence[3].trim()
      output.push(openingFence[1] + openingFence[2] + (fenceInfo ? fenceInfo + ' ' : '') + '{code-fold=' + codeFoldMetadata(wrapperMode) + '}')
      var bodyIndex
      for (bodyIndex = fenceIndex + 1; bodyIndex <= closingFenceIndex; bodyIndex += 1) output.push(lines[bodyIndex])
      for (skipped = closingFenceIndex + 1; skipped < closingWrapperIndex; skipped += 1) output.push(lines[skipped])
      index = closingWrapperIndex
    }
    return output.join('\n')
  }

  function parseMarkdownDocument(markdown) {
    var source = String(markdown || '').replace(/^\uFEFF/, '')
    var frontMatter = source.match(/^---[ \t]*\r?\n([\s\S]*?)\r?\n---[ \t]*(?:\r?\n|$)/)
    var codeFold = 'auto'
    var body = source
    if (frontMatter) {
      var fmLines = frontMatter[1].split(/\r?\n/)
      for (var i = 0; i < fmLines.length; i += 1) {
        var setting = fmLines[i].match(/^\s*code-fold\s*:\s*(.*?)\s*$/i)
        var parsed = parseCodeFoldMode(setting && setting[1])
        if (parsed) codeFold = parsed
      }
      body = source.slice(frontMatter[0].length)
    }
    return { source: applyCodeFoldWrappers(body), codeFold: codeFold }
  }

  function parseFenceInfo(info) {
    var metadataPattern = /(?:\{\s*)?code-fold\s*(?:=|:)\s*(true|false|auto|show|open|fold|hide|collapsed|expanded|off|none|disabled)(?:\s*\})?/i
    var metadata = String(info || '').match(metadataPattern)
    var cleaned = String(info || '').replace(metadataPattern, '').replace(/\{\s*\}/g, '').trim()
    var languageMatch = cleaned.match(/^[a-z0-9_+#.-]+/i)
    return {
      language: languageMatch ? languageMatch[0] : '',
      codeFold: parseCodeFoldMode(metadata && metadata[1]),
    }
  }

  function codeBlockHtml(token, documentMode) {
    var code = String(token && token.text != null ? token.text : '').replace(/\r\n/g, '\n').replace(/\n$/, '')
    var lineCount = code ? code.split('\n').length : 1
    var fence = parseFenceInfo(token && token.lang)
    var requestedMode = fence.codeFold || documentMode
    var mode = requestedMode === 'auto'
      ? (lineCount >= AUTOMATIC_FOLD_LINE_THRESHOLD ? 'collapsed' : 'disabled')
      : requestedMode
    var languageClass = fence.language ? ' class="language-' + escapeHtml(fence.language) + '"' : ''
    var languageLabel = fence.language || 'CODE'
    var escapedCode = escapeHtml(code)
    var toolbar = '<div class="markdown-code-toolbar"><span class="markdown-code-language">' +
      escapeHtml(languageLabel) + '</span><span class="markdown-code-lines">' + lineCount +
      ' 行</span><button type="button" class="markdown-copy-button" data-markdown-copy aria-label="复制代码"><span class="markdown-copy-label">复制</span></button></div>'
    var pre = '<pre><code' + languageClass + '>' + escapedCode + '</code></pre>'
    if (mode === 'disabled') {
      return '<div class="markdown-code-block" data-code-fold="disabled">' + toolbar + pre + '</div>'
    }
    var open = mode === 'expanded' ? ' open' : ''
    var summary = '<summary><span class="markdown-fold-closed">展开代码</span><span class="markdown-fold-open">收起代码</span><span class="markdown-fold-lines">' + lineCount + ' 行</span></summary>'
    return '<div class="markdown-code-block" data-code-fold="' + mode + '">' + toolbar + '<details' + open + '>' + summary + pre + '</details></div>'
  }

  function organizationSlug() {
    return (window.PXApi && window.PXApi.organizationSlug) || 'qutcraft'
  }

  function rewritePublicAssetUrls(markdown) {
    var slug = organizationSlug()
    function publicAssetUrl(assetID) {
      return '/api/v1/portal/organizations/' + slug + '/assets/' + assetID + '/download'
    }
    function portalAssetUrl(orgSlug, assetID) {
      return '/api/v1/portal/organizations/' + orgSlug + '/assets/' + assetID + '/download'
    }
    return String(markdown || '')
      .replace(absoluteAdminAssetPattern, function (_m, assetID) { return publicAssetUrl(assetID) })
      .replace(relativeAdminAssetPattern, function (_m, assetID) { return publicAssetUrl(assetID) })
      .replace(absolutePortalAssetPattern, function (_m, orgSlug, assetID) { return portalAssetUrl(orgSlug, assetID) })
      .replace(relativePortalAssetPattern, function (_m, orgSlug, assetID) { return portalAssetUrl(orgSlug, assetID) })
  }

  function renderMarkdown(markdown, publicAssets) {
    var rewritten = publicAssets ? rewritePublicAssetUrls(markdown) : String(markdown || '')
    var document = parseMarkdownDocument(rewritten)
    var markedApi = window.marked
    var purify = window.DOMPurify
    if (!markedApi || typeof markedApi.parse !== 'function' || !purify) {
      return '<p>' + escapeHtml(rewritten) + '</p>'
    }
    var renderer = new markedApi.Renderer()
    renderer.code = function (token) { return codeBlockHtml(token, document.codeFold) }
    var html = markedApi.parse(document.source, { gfm: true, breaks: true, async: false, renderer: renderer })
    return purify.sanitize(html, {
      ADD_ATTR: ['target', 'rel', 'open', 'aria-label', 'data-markdown-copy', 'data-code-fold'],
    })
  }

  function writeClipboard(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text).catch(function () {
        return fallbackCopy(text)
      })
    }
    return fallbackCopy(text)
  }

  function fallbackCopy(text) {
    return new Promise(function (resolve, reject) {
      var textarea = document.createElement('textarea')
      textarea.value = text
      textarea.readOnly = true
      textarea.style.position = 'fixed'
      textarea.style.left = '-9999px'
      document.body.appendChild(textarea)
      textarea.select()
      var copied = false
      try { copied = document.execCommand('copy') } catch (e) { copied = false }
      textarea.remove()
      if (copied) resolve()
      else reject(new Error('copy_failed'))
    })
  }

  function showCopyFeedback(button, message, copied) {
    var label = button.querySelector('.markdown-copy-label')
    if (!label) return
    label.textContent = message
    button.setAttribute('aria-label', message === '已复制' ? '代码已复制' : message)
    button.classList.toggle('is-copied', copied)
    window.setTimeout(function () {
      if (!button.isConnected) return
      label.textContent = '复制'
      button.setAttribute('aria-label', '复制代码')
      button.classList.remove('is-copied')
    }, 1600)
  }

  document.addEventListener('click', function (event) {
    var target = event.target
    if (!target || !target.closest) return
    var button = target.closest('[data-markdown-copy]')
    if (!button) return
    var block = button.closest('.markdown-code-block')
    var codeEl = block && block.querySelector('code')
    if (!codeEl) return
    writeClipboard(codeEl.textContent || '').then(function () {
      showCopyFeedback(button, '已复制', true)
    }).catch(function () {
      showCopyFeedback(button, '复制失败', false)
    })
  })

  window.PXMd = {
    render: function (src) { return renderMarkdown(src, true) },
    renderMarkdown: renderMarkdown,
    rewritePublicAssetUrls: rewritePublicAssetUrls,
  }
})()
