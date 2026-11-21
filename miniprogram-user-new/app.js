App({
  globalData: { token: wx.getStorageSync('token') || '' },
  saveToken(t) { this.globalData.token = t; wx.setStorageSync('token', t) },
  // 会话内是否已定位过（冷启动为 false，定位一次后置 true，直到退出重进）
  onLaunch() { this.globalData.sessionLocated = false }
})
