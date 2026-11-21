var api=require('../../utils/api')
Page({
  data:{coupons:[],couponName:'',discount:'',count:'1',tab:'unused',unusedList:[],usedList:[],unusedCount:0,usedCount:0},
  onShow:function(){this.load()},
  load:function(){
    var self=this
    api.getCoupons().then(function(r){
      var list=r||[];if(!Array.isArray(list))list=list.list||[]
      var unused=[],used=[]
      list.forEach(function(c){
        c._amount=((c.discount>0?c.discount:c.price)/100).toFixed(2)
        if(c.status===0)unused.push(c);else used.push(c)
      })
      self.setData({coupons:list,unusedList:unused,usedList:used,unusedCount:unused.length,usedCount:used.length})
    })
  },
  switchTab:function(e){this.setData({tab:e.currentTarget.dataset.tab})},
  onNameIn:function(e){this.setData({couponName:e.detail.value})},
  onDiscountIn:function(e){this.setData({discount:e.detail.value})},
  onCountIn:function(e){this.setData({count:e.detail.value})},
  create:function(){
    var self=this
    if(!this.data.couponName){wx.showToast({title:'请输券名',icon:'none'});return}
    var discount=parseFloat(this.data.discount)||0
    if(discount<1){wx.showToast({title:'请输券面额（元）',icon:'none'});return}
    var count=parseInt(this.data.count)||1
    if(count<1)count=1
    if(count>100)count=100
    wx.showLoading({title:'生成中...'})
    api.createCoupon({name:this.data.couponName,discount:Math.round(discount*100),type:1,hours:0,price:0,resource_type_id:0,count:count}).then(function(r){
      wx.hideLoading()
      var n=(r&&r.data&&r.data.count)||(r&&r.count)||count
      wx.showModal({title:'生成成功',content:'已生成 '+n+' 张券码\n\n用户在小程序「券码兑换」输入券码即可充入余额',showCancel:false})
      self.setData({couponName:'',discount:'',count:'1',tab:'unused'})
      self.load()
    }).catch(function(){wx.hideLoading();wx.showToast({title:'生成失败',icon:'none'})})
  },
  copyCode:function(e){
    var code=e.currentTarget.dataset.code||''
    wx.setClipboardData({data:code,success:function(){wx.showToast({title:'券码已复制'})}})
  },
  delCoupon:function(e){
    var self=this,id=e.currentTarget.dataset.id
    wx.showModal({title:'删除券码',content:'确定删除该券码？删除后不可恢复',confirmColor:'#e94560',success:function(r){
      if(!r.confirm)return
      api.deleteCoupon(id).then(function(){wx.showToast({title:'已删除'});self.load()}).catch(function(){wx.showToast({title:'删除失败',icon:'none'})})
    }})
  },
  // 导出未使用券码
  exportCodes:function(){
    var unused=this.data.unusedList
    if(unused.length===0){wx.showToast({title:'没有未使用的券码',icon:'none'});return}
    var lines=unused.map(function(c){return c.code+'\t'+c.name+'\t'+c._amount})
    var content='券码\t券名\t面额(元)\n'+lines.join('\n')
    try{
      var fs=wx.getFileSystemManager()
      var path=wx.env.USER_DATA_PATH+'/券码列表.csv'
      fs.writeFileSync(path,content,'utf8')
      wx.shareFileMessage({filePath:path,fileName:'券码列表.csv',
        success:function(){wx.showToast({title:'已导出'})},
        fail:function(){wx.setClipboardData({data:content,success:function(){wx.showToast({title:'已复制全部券码'})}})}
      })
    }catch(e){
      wx.setClipboardData({data:content,success:function(){wx.showToast({title:'已复制全部券码'})}})
    }
  },
  goBack:function(){wx.navigateBack()}
})
