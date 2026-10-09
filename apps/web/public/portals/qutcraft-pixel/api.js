/* ============================================================================
 * api.js — 数据访问层（生产：同源 Portal API）
 * ----------------------------------------------------------------------------
 * 方法名与平台 portalApi（apps/web/src/api/portal.ts）一一对应，
 * 返回值结构对齐 apps/web/src/api/types.ts。
 *
 *   const base = `/api/v1/portal/organizations/${ORG_SLUG}`   // 同源，勿写绝对地址
 *   getOrganization       GET    base
 *   getPosts              GET    base + '/posts?page=&page_size=&category=&q='
 *   getContentDetail      GET    base + '/content/{id}'
 *   getProjects           GET    base + '/projects?page=&page_size=&status='
 *   getResources          GET    base + '/resources?page=&page_size=&kind=&q='
 *   getKnowledgeArticles  GET    base + '/knowledge/articles?...'
 *   getKnowledgeDirectories GET  base + '/knowledge/directories'
 *   submitApplication     POST   base + '/apply'      body: ApplicationPayload
 *   submitFeedback        POST   base + '/feedback'   body: SiteFeedbackPayload
 *
 * 响应为统一封装 { data, meta }；分页数据为 Page<T>（items/page/page_size/total）。
 * slug 解析顺序与 portal.ts:17-38 一致：URL ?organization= → sessionStorage → 默认 qutcraft。
 * 禁止调用 /api/v1/admin/*；下载地址直接使用接口返回的 download_url。
 * ========================================================================== */
(function () {
  'use strict'

  var DEFAULT_SLUG = 'qutcraft'
  var STORAGE_KEY = 'qutc.portal_organization_slug'
  var SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/

  function normalizeOrganizationSlug(value) {
    var slug = String(value == null ? '' : value).trim().toLowerCase()
    return SLUG_PATTERN.test(slug) ? slug : ''
  }

  function resolveOrganizationSlug() {
    var fallback = DEFAULT_SLUG
    try {
      var requestedValue = ''
      new URLSearchParams(window.location.search).forEach(function (value, key) {
        if (key === 'organization') requestedValue = value
      })
      var requested = normalizeOrganizationSlug(requestedValue)
      if (requested) {
        window.sessionStorage.setItem(STORAGE_KEY, requested)
        return requested
      }
      return normalizeOrganizationSlug(window.sessionStorage.getItem(STORAGE_KEY)) || fallback
    } catch (e) {
      return fallback
    }
  }

  var organizationSlug = resolveOrganizationSlug()
  var portalBase = '/api/v1/portal/organizations/' + organizationSlug

  function withQuery(path, params) {
    params = params || {}
    var query = new URLSearchParams()
    Object.keys(params).forEach(function (key) {
      var value = params[key]
      if (value !== undefined && value !== '') query.set(key, String(value))
    })
    var suffix = query.toString()
    return path + (suffix ? '?' + suffix : '')
  }

  function request(method, path, body) {
    var opts = {
      method: method,
      headers: { Accept: 'application/json' },
      credentials: 'same-origin',
    }
    if (body !== undefined) {
      opts.headers['Content-Type'] = 'application/json'
      opts.body = JSON.stringify(body)
    }
    return fetch(path, opts).then(function (response) {
      return response.json().then(function (payload) {
        return { response: response, payload: payload }
      }, function () {
        return { response: response, payload: null }
      })
    }).then(function (result) {
      var response = result.response
      var payload = result.payload
      if (!response.ok || !payload || !Object.prototype.hasOwnProperty.call(payload, 'data')) {
        var err = (payload && payload.error) || {}
        var error = new Error(err.message || '请求失败，请稍后重试。')
        error.status = response.status
        error.code = err.code || 'network.request_failed'
        throw error
      }
      return payload
    })
  }

  function getData(path) {
    return request('GET', path).then(function (payload) { return payload.data })
  }

  function getPage(path) {
    return request('GET', path).then(function (payload) {
      var items = Array.isArray(payload.data) ? payload.data : []
      var meta = payload.meta || {}
      return {
        items: items,
        page: meta.page != null ? meta.page : 1,
        page_size: meta.page_size != null ? meta.page_size : items.length,
        total: meta.total != null ? meta.total : items.length,
      }
    })
  }

  function postData(path, body) {
    return request('POST', path, body).then(function (payload) { return payload.data })
  }

  window.PXApi = {
    organizationSlug: organizationSlug,
    portalBase: portalBase,
    getOrganization: function () { return getData(portalBase) },
    getPosts: function (query) { return getPage(withQuery(portalBase + '/posts', query)) },
    getContentDetail: function (id) {
      return getData(portalBase + '/content/' + encodeURIComponent(id))
    },
    getProjects: function (query) { return getPage(withQuery(portalBase + '/projects', query)) },
    getResources: function (query) { return getPage(withQuery(portalBase + '/resources', query)) },
    getKnowledgeArticles: function (query) {
      return getPage(withQuery(portalBase + '/knowledge/articles', query))
    },
    getKnowledgeDirectories: function (query) {
      return getPage(withQuery(portalBase + '/knowledge/directories', query || {}))
    },
    submitApplication: function (payload) { return postData(portalBase + '/apply', payload) },
    submitFeedback: function (payload) { return postData(portalBase + '/feedback', payload) },
  }
})()
