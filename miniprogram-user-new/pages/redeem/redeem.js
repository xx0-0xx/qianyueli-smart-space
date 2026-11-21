var api=require('../../utils/api')
Page({
  data:{code:'',verified:false,info:{},used:null},
  onCodeInput:function(e){this.setData({code:e.detail.value})},
  verify:function(){
    var self=this
    if(!this.data.code||this.data.code.length<6){wx.showToast({title:'券码格式不对',icon:'none'});return}
    wx.showLoading({title:'验证中'})
    api.getCouponInfo(this.data.code).then(function(r){
      var d=r.data||r
      wx.hideLoading()
      self.setData({verified:true,info:{name:d.name||'券码',amount:d.amount||0,code:d.code}})
    }).catch(function(){
      wx.hideLoading()
      wx.showToast({title:'券码无效或已使用',icon:'none'})
    })
  },
  useCoupon:function(){
    var self=this
    wx.showLoading({title:'核销中'})
    api.redeemCoupon(this.data.code).then(function(r){
      var d=r.data||r
      wx.hideLoading()
      self.setData({used:{amount:d.amount,balance:d.balance}})
      // 刷新余额缓存
      setTimeout(function(){wx.switchTab({url:'/pages/mine/mine'})},1500)
    }).catch(function(e){
      wx.hideLoading()
      wx.showToast({title:e.message||'核销失败',icon:'none'})
    })
  }
})
