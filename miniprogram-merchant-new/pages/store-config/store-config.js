var api=require('../../utils/api')
Page({
  data:{wifiName:'',wifiPass:'',phone:'',autoApproveRefund:true},
  onShow:function(){this.load()},
  load:function(){
    var self=this
    api.getStoreConfig().then(function(r){
      var d=r||r.data||{}
      self.setData({phone:d.phone||'', autoApproveRefund: d.auto_approve_refund === undefined ? true : !!d.auto_approve_refund})
      try{
        var cfg=JSON.parse(d.config_json||'{}')
        self.setData({wifiName:cfg.wifi_ssid||'',wifiPass:cfg.wifi_pass||''})
      }catch(e){}
    })
  },
  onWifiNameIn:function(e){this.setData({wifiName:e.detail.value})},
  onWifiPassIn:function(e){this.setData({wifiPass:e.detail.value})},
  onPhoneIn:function(e){this.setData({phone:e.detail.value})},
  onAutoApproveChange:function(e){this.setData({autoApproveRefund:!!e.detail.value})},
  save:function(){
    var cfg=JSON.stringify({wifi_ssid:this.data.wifiName,wifi_pass:this.data.wifiPass})
    api.updateStoreConfig({config_json:cfg,phone:this.data.phone,auto_approve_refund:this.data.autoApproveRefund}).then(function(){
      wx.showToast({title:'已保存'})
    }).catch(function(){wx.showToast({title:'保存失败',icon:'none'})})
  },
  goBack:function(){wx.navigateBack()},
  openAgreement:function(){wx.navigateTo({url:'/pages/agreement/agreement'})},
  openPrivacy:function(){wx.navigateTo({url:'/pages/privacy/privacy'})}
})
