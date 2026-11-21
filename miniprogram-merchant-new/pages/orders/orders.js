var api=require('../../utils/api')
function fmtTime(ms){
  if(!ms)return ''
  var d=new Date(ms)
  var p=function(n){return n<10?'0'+n:''+n}
  return (d.getMonth()+1)+'月'+d.getDate()+'日 '+p(d.getHours())+':'+p(d.getMinutes())
}
Page({
  data:{tab:'orders',orders:[],refunds:[],refundCount:0,showDetail:false,detailRows:[],page:0,totalPages:1,loading:false},
  onShow:function(){var app=getApp();if(app.globalData.token){this._loadSeq=(this._loadSeq||0)+1;this.setData({loading:false,page:0,orders:[],refunds:[]});if(app.globalData.ordersTab)this.setData({tab:app.globalData.ordersTab});app.globalData.ordersTab='';this.load(true)}},
  onLoad:function(){var app=getApp();if(app.globalData.ordersTab)this.setData({tab:app.globalData.ordersTab});},
  switchTab:function(e){this.setData({tab:e.currentTarget.dataset.tab});if(e.currentTarget.dataset.tab==='orders')this.load(true)},
  load:function(reset){
    var self=this;var sid=String(getApp().globalData.storeId||wx.getStorageSync('merchantSid')||'');if(this.data.loading)return
    var nextPage=reset?1:this.data.page+1;if(!reset&&nextPage>this.data.totalPages)return
    var seq=this._loadSeq=(this._loadSeq||0)+1
    this.setData({loading:true})
    api.getMerchantOrders({page:nextPage,size:20},sid).then(function(r){
      if(seq!==self._loadSeq||sid!==String(getApp().globalData.storeId||wx.getStorageSync('merchantSid')||''))return
      var d=r||{};var list=Array.isArray(d)?d:(d.list||[])
      list.forEach(function(o){o.price=(Number(o.total_amount||0)/100).toFixed(2);o.timeText=fmtTime(o.start_time)+' - '+fmtTime(o.end_time)})
      self.setData({orders:reset?list:self.data.orders.concat(list),page:Number(d.page||nextPage),totalPages:Number(d.total_pages||1),loading:false})
    }).catch(function(){if(seq===self._loadSeq){self.setData({loading:false});wx.showToast({title:'订单加载失败',icon:'none'})}})
    api.getRefundRequests(sid).then(function(r){
      if(seq!==self._loadSeq||sid!==String(getApp().globalData.storeId||wx.getStorageSync('merchantSid')||''))return
      var d=r||{};var list=Array.isArray(d)?d:(d.list||[]);list.forEach(function(o){o.amountYuan=(Number(o.amount||0)/100).toFixed(2)})
      self.setData({refunds:list,refundCount:list.filter(function(o){return Number(o.status)===0}).length})
    }).catch(function(){})
  },
  loadMore:function(){this.load(false)},

  showDetail:function(e){
    var o=this.data.orders[e.currentTarget.dataset.index]
    if(!o)return
    var stMap=['待支付','已支付','进行中','已完成','已取消','退款中','已退款']
    this.setData({showDetail:true,detailRows:[
      {label:'订单号',value:o.order_no},
      {label:'房间',value:o.resource_name||'-'},
      {label:'时间',value:o.timeText},
      {label:'时长',value:o.duration+'分钟'},
      {label:'金额',value:'¥'+o.price},
      {label:'状态',value:stMap[o.status]||'-'},
      {label:'下单时间',value:fmtTime(o.created_at)}
    ]})
  },
  hideDetail:function(){this.setData({showDetail:false})},
  cancelOrder:function(e){
    var self=this,id=e.currentTarget.dataset.id
    if(this._actionBusy)return
    wx.showModal({title:'取消订单',content:'确认取消该待支付订单？',success:function(r){
      if(!r.confirm)return
      self._actionBusy=true
      api.merchantCancelOrder(id).then(function(){wx.showToast({title:'已取消'});self.load(true)}).catch(function(err){wx.showToast({title:(err&&err.message)||'取消失败',icon:'none'})}).finally(function(){self._actionBusy=false})
    }})
  },
  refundOrder:function(e){
    var self=this,id=e.currentTarget.dataset.id
    if(this._actionBusy)return
    wx.showModal({title:'确认退款',content:'仅余额支付订单支持自动退回余额。确认后订单关闭且不可恢复。',confirmText:'确认退款',success:function(r){
      if(!r.confirm)return
      self._actionBusy=true
      api.merchantDirectRefund(id).then(function(){wx.showToast({title:'退款已退回余额'});self.load(true)}).catch(function(err){wx.showToast({title:(err&&err.message)||'退款处理失败',icon:'none'})}).finally(function(){self._actionBusy=false})
    }})
  },
  approve:function(e){
    var self=this,id=e.currentTarget.dataset.id
    if(this._actionBusy)return
    wx.showModal({title:'确认同意退款',content:'仅余额支付订单会自动退回余额，其他支付方式暂不支持自动退款。',success:function(r){if(!r.confirm)return
      self._actionBusy=true
      api.approveRefund(id).then(function(){wx.showToast({title:'退款已退回余额'});self.load(true)}).catch(function(err){wx.showToast({title:(err&&err.message)||'退款处理失败',icon:'none'})}).finally(function(){self._actionBusy=false})
    }})
  },
  reject:function(e){
    var self=this,id=e.currentTarget.dataset.id
    if(this._actionBusy)return
    wx.showModal({title:'确认拒绝退款',success:function(r){if(!r.confirm)return
      self._actionBusy=true
      api.rejectRefund(id).then(function(){wx.showToast({title:'已拒绝'});self.load(true)}).catch(function(err){wx.showToast({title:(err&&err.message)||'处理失败',icon:'none'})}).finally(function(){self._actionBusy=false})
    }})
  }
})
