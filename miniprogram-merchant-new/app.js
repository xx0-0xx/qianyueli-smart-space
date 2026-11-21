App({
  globalData: {
    token: wx.getStorageSync('token') || '',
    storeId: wx.getStorageSync('merchantSid') || ''
  }
})
