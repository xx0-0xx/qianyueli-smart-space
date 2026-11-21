var api=require('../../utils/api')
Page({
  data: { months: [], loading:false },
  goBack() { wx.navigateBack() },
  onLoad() { this.load() },
  async load() {
    this.setData({loading:true})
    try {
      var sid=String(getApp().globalData.storeId||wx.getStorageSync('merchantSid')||'')
      var orders=await api.getAllMerchantOrders(sid)
      var monthMap={}
      orders.forEach(function(o) {
        var dt=new Date(o.start_time)
        var key=dt.getFullYear()+'-'+('0'+(dt.getMonth()+1)).slice(-2)
        if(!monthMap[key])monthMap[key]={month:key,total:0,paid:0,refund:0,revenue:0}
        var m=monthMap[key];m.total++
        if([1,2,3].indexOf(Number(o.status))>=0){m.paid++;m.revenue+=Number(o.total_amount||0)}
        if(Number(o.status)===6)m.refund++
      })
      var months=Object.keys(monthMap).sort().reverse().map(function(k){
        var m=monthMap[k];m.revenue=(m.revenue/100).toFixed(2);m.avgPrice=m.paid?((Number(m.revenue)/m.paid).toFixed(2)): '0.00';return m
      })
      this.setData({months:months,loading:false})
    } catch(e) { this.setData({loading:false});wx.showToast({title:'报表加载失败',icon:'none'}) }
  },
  showDetail(e) {
    var m=this.data.months.find(function(x){return x.month===e.currentTarget.dataset.month})
    if(!m)return
    wx.showModal({title:m.month,content:'总订单:'+m.total+' 已支付:'+m.paid+' 已退款:'+m.refund+' 营收:'+m.revenue+'元 客单价:'+m.avgPrice+'元',showCancel:false})
  }
})
