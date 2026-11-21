var api=require('../../utils/api')
Page({
  data:{typePrices:[],types:[]},
  onShow:function(){this.load()},
  load:function(){
    var self=this
    Promise.all([api.getResourceTypes(),api.getPriceRules()]).then(function(arr){
      var types=arr[0]||[]
      var rules=arr[1]||{}
      rules=Array.isArray(rules)?rules:(rules.list||[])
      var tp=types.map(function(t){
        var tr=rules.find(function(r){return r.resource_type_id==t.id})
        return {type_id:t.id,name:t.name,price:tr?Math.round(tr.base_price/100):50}
      })
      self.setData({typePrices:tp,types:types})
    })
  },
  onTypeInput:function(e){
    var tid=e.currentTarget.dataset.id
    var v=e.detail.value
    var val=v===''?0:parseInt(v)||0
    var tp=this.data.typePrices.map(function(t){
      if(t.type_id==tid)t.price=val;return t
    })
    this.setData({typePrices:tp})
  },
  save:function(){
    wx.showLoading({title:'保存中'})
    var rules=this.data.typePrices.map(function(t){
      return {resource_type_id:parseInt(t.type_id)||0,is_default:false,base_price:t.price*100,time_unit:60,name:t.name,status:1}
    }).filter(function(r){return r.base_price>0})
    if(rules.length==0){wx.hideLoading();wx.showToast({title:'请先设置价格',icon:'none'});return}
    var self=this
    api.replacePriceRules(rules).then(function(){
      self.load()
      wx.hideLoading()
      wx.showToast({title:'已保存'})
    }).catch(function(){
      wx.hideLoading()
      wx.showToast({title:'保存失败',icon:'none'})
    })
  },
  goBack:function(){wx.navigateBack()}
})
