<script setup lang="ts">
import { reactive } from 'vue'
import { ElMessage } from 'element-plus'

const form = reactive({ name: 'QUTCraft Commons', slug: 'qutcraft', publicPortal: true, customPortal: false })
</script>

<template>
  <section class="admin-page-heading"><div><p class="eyebrow">ORGANIZATION & PORTAL</p><h2>组织设置</h2><p>默认门户使用 MD3；自定义门户只通过公开 Portal API 读取已发布数据。</p></div></section><section class="settings-layout"><article class="admin-panel"><h2>公开门户</h2><el-form :model="form" label-position="top"><el-form-item label="组织名称"><el-input v-model="form.name" /></el-form-item><el-form-item label="组织标识"><el-input v-model="form.slug" disabled /></el-form-item><el-form-item><el-switch v-model="form.publicPortal" active-text="启用公开门户" /></el-form-item><el-form-item><el-switch v-model="form.customPortal" active-text="允许自定义门户接入" /></el-form-item><el-button type="primary" @click="ElMessage.success('设置已保存到模拟环境。')">保存设置</el-button></el-form></article><aside class="admin-panel settings-note"><p class="eyebrow">API BOUNDARY</p><h2>安全边界</h2><p>公开读取：<code>/api/v1/portal/*</code></p><p>受控管理：<code>/api/v1/admin/*</code></p><small>前端不能自行拼接对象存储链接，也不能在公开页面调用成员、审核或 RCON 接口。</small></aside></section>
</template>
