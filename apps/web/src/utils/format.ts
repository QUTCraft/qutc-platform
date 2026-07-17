export const formatDate = (value: string) => new Intl.DateTimeFormat('zh-CN', {
  month: 'long', day: 'numeric', year: 'numeric',
}).format(new Date(value))

export const formatBytes = (value: number) => {
  if (value < 1_024 * 1_024) return `${Math.round(value / 1_024)} KB`
  return `${(value / (1_024 * 1_024)).toFixed(1)} MB`
}
