var api=require('../../utils/api')
Page({
  data:{date:'',hours:[],rows:[],loading:false},
  _loadSeq:0,
  onLoad:function(){var d=new Date(),ds=this.fmt(d);this.setData({date:ds});this.load(ds)},
  load:function(ds){
    var self=this,seq=++this._loadSeq
    var sid=String(getApp().globalData.storeId||wx.getStorageSync('merchantSid')||'')
    var base=new Date(ds.replace(/-/g,'/')+' 00:00:00').getTime()
    var hours=[];for(var i=0;i<24;i++)hours.push((i<10?'0':'')+i)
    this.setData({hours:hours,loading:true})
    Promise.all([api.getResources(sid),api.getAllMerchantOrders(sid)]).then(function(arr){
      if(seq!==self._loadSeq||ds!==self.data.date)return
      var rooms=Array.isArray(arr[0])?arr[0]:((arr[0]&&arr[0].list)||[]),orders=arr[1]||[]
      var rows=rooms.map(function(r){
        var cells=[]
        for(var h=0;h<24;h++){
          var st=base+h*3600000,et=st+3600000
          var booked=orders.some(function(o){return String(o.resource_id)===String(r.id)&&[0,1,2,3,5].indexOf(Number(o.status))>=0&&o.start_time<et&&o.end_time>st})
          cells.push(booked?1:0)
        }
        return {id:r.id,name:r.name,cells:cells}
      })
      self.setData({rows:rows,loading:false})
    }).catch(function(){if(seq===self._loadSeq){self.setData({loading:false});wx.showToast({title:'时段加载失败',icon:'none'})}})
  },
  fmt:function(d){var m=d.getMonth()+1,day=d.getDate();return d.getFullYear()+'-'+(m<10?'0':'')+m+'-'+(day<10?'0':'')+day},
  prevDay:function(){var d=new Date(this.data.date.replace(/-/g,'/'));d.setDate(d.getDate()-1);var ns=this.fmt(d);this.setData({date:ns,rows:[]});this.load(ns)},
  nextDay:function(){var d=new Date(this.data.date.replace(/-/g,'/'));d.setDate(d.getDate()+1);var ns=this.fmt(d);this.setData({date:ns,rows:[]});this.load(ns)},
  goBack:function(){wx.navigateBack()}
})
