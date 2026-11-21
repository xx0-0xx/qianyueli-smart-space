var api=require('../../utils/api')
function fmtTime(ms){
  if(!ms)return ''
  var d=new Date(ms)
  var p=function(n){return n<10?'0'+n:''+n}
  return (d.getMonth()+1)+'月'+d.getDate()+'日 '+p(d.getHours())+':'+p(d.getMinutes())
}
Page({
  data:{tab:'0',list:[]},
  goBack(){wx.navigateBack()},
  onShow:function(){this.load()},
  switchTab:function(e){this.setData({tab:String(e.currentTarget.dataset.tab)});this.load()},
  load:function(){
    var self=this
    api.getMerchantRecharges(this.data.tab).then(function(r){
      var list=r||[]
      if(!Array.isArray(list))list=list.list||[]
      list.forEach(function(item){
        item.amountYuan=(item.amount/100).toFixed(2)
        item.timeText=fmtTime(item.created_at)
      })
      self.setData({list:list})
    }).catch(function(){})
  },
  confirm:function(e){
    var self=this
    var id=e.currentTarget.dataset.id
    var item=this.data.list.find(function(x){return String(x.id)===String(id)})
    wx.showModal({
      title:'确认到账',
      content:'确认用户充值 ¥'+(item?item.amountYuan:'')+' 已到账？确认后将加余额。',
      success:function(r){
        if(!r.confirm)return
        api.confirmRecharge(id).then(function(){
          wx.showToast({title:'已到账'})
          self.load()
        }).catch(function(){wx.showToast({title:'操作失败',icon:'none'})})
      }
    })
  },
  cancel:function(e){
    var self=this
    var id=e.currentTarget.dataset.id
    wx.showModal({
      title:'取消充值',
      content:'确认取消该充值单？',
      success:function(r){
        if(!r.confirm)return
        api.cancelRecharge(id).then(function(){
          wx.showToast({title:'已取消'})
          self.load()
        }).catch(function(){wx.showToast({title:'操作失败',icon:'none'})})
      }
    })
  }
})
