const recoveryKey = 'qutc:stale-chunk-recovery'
const now = Date.now()
const lastRecovery = Number(window.sessionStorage.getItem(recoveryKey) || 0)

if (now - lastRecovery >= 30000) {
  window.sessionStorage.setItem(recoveryKey, String(now))
  window.location.reload()
}

export default {}
