// Package scheduler 定时任务
package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"smart-space-management/backend/config"
	devicePkg "smart-space-management/backend/internal/device"
	cfgModel "smart-space-management/backend/internal/model/config"
	"smart-space-management/backend/internal/model/device"
	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/internal/model/store"
	"smart-space-management/backend/pkg/alert"

	"encoding/json"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	db       *gorm.DB
	redis    *redis.Client
	cfg      config.SchedulerConfig
	voice    config.VoiceConfig
	notifier *alert.Notifier
	ctx      context.Context
	stop     context.CancelFunc
}

func New(db *gorm.DB, rdb *redis.Client, cfg config.SchedulerConfig, notifier *alert.Notifier, voice config.VoiceConfig) *Scheduler {
	ctx, stop := context.WithCancel(context.Background())
	return &Scheduler{db: db, redis: rdb, cfg: cfg, voice: voice, notifier: notifier, ctx: ctx, stop: stop}
}

// Start 启动所有定时任务
func (s *Scheduler) Start() {
	go s.loop("order_timeout", s.cfg.OrderTimeoutCheckInterval, s.checkOrderTimeout)
	go s.loop("device_health", s.cfg.DeviceHealthCheckInterval, s.checkDeviceHealth)
	go s.loop("auto_settle", s.cfg.AutoSettleInterval, s.autoSettle)
	go s.loop("auto_close", time.Minute, s.checkAutoClose)
	if s.cfg.DailyReportEnabled {
		go s.loop("daily_report", time.Minute, s.checkDailyReport)
	}
	// 语音配置由管理后台按设备 SN 动态保存，任务始终运行并按当前数据库配置决定是否播报。
	go s.loop("voice_remind", time.Minute, s.checkVoiceReminders)
	log.Println("[SCHEDULER] started")
}

// Stop 停止所有任务
func (s *Scheduler) Stop() {
	s.stop()
	log.Println("[SCHEDULER] stopped")
}

func (s *Scheduler) loop(name string, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fn() // 启动时立即执行一次

	for {
		select {
		case <-ticker.C:
			fn()
		case <-s.ctx.Done():
			log.Printf("[SCHEDULER] %s stopped", name)
			return
		}
	}
}

// checkOrderTimeout 检查超时未支付订单，释放资源
func (s *Scheduler) checkOrderTimeout() {
	expireTime := time.Now().Add(-s.cfg.OrderPendingPayTTL).UnixMilli()

	var expiredOrders []order.Order
	s.db.Where("status = ? AND created_at < ?",
		order.StatusPendingPay, time.UnixMilli(expireTime)).
		Find(&expiredOrders)

	for _, o := range expiredOrders {
		o.Status = order.StatusCanceled
		o.CancelReason = "超时未支付，系统自动取消"
		s.db.Save(&o)

		// 释放资源
		s.db.Model(&resource.Resource{}).Where("id = ?", o.ResourceID).
			Update("status", resource.StatusFree)

		log.Printf("[SCHEDULER] order %s auto-canceled (timeout)", o.OrderNo)
	}
}

// checkDeviceHealth 检查设备心跳超时 → 标记离线 + 主动告警
func (s *Scheduler) checkDeviceHealth() {
	// 成品控制器空闲时不会主动上报，先发 qa 查询一次状态，避免被误判离线
	s.pingASCIIDevices()

	// 设备超过 2 倍心跳间隔未上报则视为离线
	threshold := time.Now().UnixMilli() - 2*int64(s.cfg.DeviceHealthCheckInterval.Milliseconds())

	// 查出「当前在线但心跳超时」的设备（即将离线）
	var offlineDevices []device.Device
	s.db.Where("status = ? AND last_online_at < ?", 1, threshold).Find(&offlineDevices)
	if len(offlineDevices) == 0 {
		return
	}

	for _, d := range offlineDevices {
		s.db.Model(&device.Device{}).Where("id = ?", d.ID).Update("status", 0)

		// Redis 去重：同一设备 30 分钟内只告警一次
		key := fmt.Sprintf("ssm:alert:offline:%d", d.ID)
		ok, _ := s.redis.SetNX(s.ctx, key, 1, 30*time.Minute).Result()
		if !ok {
			continue
		}

		// 门店名 + 该门店的群机器人
		var st store.Store
		s.db.First(&st, d.StoreID)
		storeName := st.Name
		if storeName == "" {
			storeName = fmt.Sprintf("门店%d", d.StoreID)
		}

		msg := fmt.Sprintf("🔴【设备离线】%s 的设备「%s」(SN:%s) 已离线，请检查网络/电源。",
			storeName, d.Name, d.DeviceSN)
		log.Printf("[ALERT] %s", msg)
		s.sendStoreAlert(st.NotifyWebhook, msg)
	}
}

// pingASCIIDevices 主动查询成品控制器状态，设备回复会更新 last_online_at。
func (s *Scheduler) pingASCIIDevices() {
	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		return
	}
	var rows []device.Device
	if err := s.db.Select("device_sn").
		Where("protocol = ?", device.ProtocolASCII).
		Find(&rows).Error; err != nil {
		log.Printf("[DEVICE] 查询 ASCII 设备失败: %v", err)
		return
	}
	if len(rows) == 0 {
		return
	}

	seen := make(map[string]bool, len(rows))
	sent := 0
	for _, row := range rows {
		if row.DeviceSN == "" || seen[row.DeviceSN] {
			continue
		}
		seen[row.DeviceSN] = true
		if err := cli.PublishRaw(row.DeviceSN, "qa"); err != nil {
			log.Printf("[DEVICE] 查询设备状态失败 SN=%s: %v", row.DeviceSN, err)
			continue
		}
		sent++
	}
	if sent > 0 {
		// 给设备留出上报时间，随后再判断心跳是否超时
		time.Sleep(2 * time.Second)
	}
}

// sendStoreAlert 优先发到门店自己的群，未配置则回落到全局 webhook
func (s *Scheduler) sendStoreAlert(storeWebhook, msg string) {
	if s.notifier == nil {
		return
	}
	var err error
	if storeWebhook != "" {
		err = s.notifier.SendTo(storeWebhook, msg)
	} else {
		err = s.notifier.Send(msg)
	}
	if err != nil {
		log.Printf("[ALERT] send failed: %v", err)
	}
}

// autoSettle 自动结算已完成订单
func (s *Scheduler) autoSettle() {
	now := time.Now().UnixMilli()

	var doneOrders []order.Order
	s.db.Where("status = ? AND end_time < ?", order.StatusPaid, now).
		Find(&doneOrders)

	for _, o := range doneOrders {
		tx := s.db.Begin()

		o.Status = order.StatusCompleted
		tx.Save(&o)

		// 释放资源
		tx.Model(&resource.Resource{}).Where("id = ?", o.ResourceID).
			Update("status", resource.StatusFree)

		tx.Commit()
		log.Printf("[SCHEDULER] order %s auto-completed", o.OrderNo)
	}
}

// checkAutoClose 到点按每台设备的延迟自动关闭。
// 设备需在 config_json 配置 auto_close=true；auto_close_delay_min 为到点后延迟分钟（默认 0）。
// 用 Redis 去重，保证同订单同设备只关一次。
func (s *Scheduler) checkAutoClose() {
	now := time.Now().UnixMilli()
	var orders []order.Order
	if err := s.db.Where("end_time < ? AND end_time > ? AND status IN ?",
		now, now-24*3600*1000,
		[]order.OrderStatus{order.StatusPaid, order.StatusInProgress, order.StatusCompleted}).
		Find(&orders).Error; err != nil {
		return
	}
	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		return
	}
	for _, o := range orders {
		var res resource.Resource
		if s.db.First(&res, o.ResourceID).Error != nil || res.DeviceIDs == "" {
			continue
		}
		var ids []uint64
		var err error
		if ids, err = resource.ParseDeviceIDs(res.DeviceIDs); err != nil || len(ids) == 0 {
			if err != nil {
				log.Printf("[AUTOCLOSE] parse device_ids resource=%d: %v", res.ID, err)
			}
			continue
		}
		var devices []device.Device
		s.db.Where("id IN ?", ids).Find(&devices)
		for _, dev := range devices {
			var cfg struct {
				CmdOff    string `json:"cmd_off"`
				AutoClose bool   `json:"auto_close"`
				DelayMin  int    `json:"auto_close_delay_min"`
			}
			if dev.ConfigJSON != "" {
				json.Unmarshal([]byte(dev.ConfigJSON), &cfg)
			}
			if !autoCloseDue(now, o.EndTime, cfg.DelayMin, cfg.AutoClose) {
				continue
			}
			cmd := cfg.CmdOff
			if cmd == "" {
				cmd = defaultOffCommand(dev.Type)
			}
			if cmd == "" {
				continue
			}
			key := fmt.Sprintf("ssm:autoclose:%d:%d", o.ID, dev.ID)
			ok, err := s.redis.SetNX(s.ctx, key, 1, 7*24*time.Hour).Result()
			if err != nil || !ok {
				continue
			}
			var sendErr error
			if dev.Protocol == device.ProtocolASCII {
				sendErr = cli.PublishRaw(dev.DeviceSN, cmd)
			} else {
				sendErr = cli.PublishCommand(dev.DeviceSN, cmd, nil)
			}
			if sendErr != nil {
				s.redis.Del(s.ctx, key)
				log.Printf("[AUTOCLOSE] send failed order=%s device=%d sn=%s cmd=%s: %v", o.OrderNo, dev.ID, dev.DeviceSN, cmd, sendErr)
				continue
			}
			log.Printf("[AUTOCLOSE] order=%s device=%d sn=%s cmd=%s delay=%dm", o.OrderNo, dev.ID, dev.DeviceSN, cmd, cfg.DelayMin)
		}
	}
}

// autoCloseDue 判断某设备到点后是否已到关闭时刻（纯函数，可单测）
func autoCloseDue(nowMs, endMs int64, delayMin int, autoClose bool) bool {
	if !autoClose {
		return false
	}
	return nowMs >= endMs+int64(delayMin)*60000
}

func defaultOffCommand(t device.DeviceType) string {
	switch t {
	case "light", "other":
		return "light_off"
	case "ac":
		return "ac_off"
	case "exhaust":
		return "exhaust_off"
	case "door_lock":
		return "door_close"
	default:
		return "light_off"
	}
}

// checkDailyReport 每天 22:00 推送「今日营收」
// 配了 webhook 的门店：各推各的；未配的门店：汇总成一条推全局群
func (s *Scheduler) checkDailyReport() {
	now := time.Now()
	reportHour := s.cfg.DailyReportHour
	if reportHour <= 0 || reportHour > 23 {
		reportHour = 22
	}
	if now.Hour() != reportHour || now.Minute() > 1 {
		return
	}
	// 当天去重（只发一次）
	key := fmt.Sprintf("ssm:report:%s", now.Format("20060102"))
	ok, _ := s.redis.SetNX(s.ctx, key, 1, 25*time.Hour).Result()
	if !ok {
		return
	}

	var stores []store.Store
	s.db.Where("status = ?", store.StatusOpen).Find(&stores)
	if len(stores) == 0 {
		return
	}

	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UnixMilli()
	dayEnd := dayStart + 86400000
	dateStr := now.Format("2006-01-02")

	var summary []string
	totalRevenue, totalCnt := 0, 0

	for _, st := range stores {
		var orders []order.Order
		s.db.Where("store_id = ? AND start_time >= ? AND start_time < ?", st.ID, dayStart, dayEnd).Find(&orders)

		revenue, cnt := 0, 0
		for _, o := range orders {
			if o.Status == order.StatusPaid || o.Status == order.StatusInProgress || o.Status == order.StatusCompleted {
				revenue += o.TotalAmount
				cnt++
			}
		}
		log.Printf("[REPORT] %s: %d单 ¥%.2f", st.Name, cnt, float64(revenue)/100)

		if st.NotifyWebhook != "" {
			// 该门店有独立群 → 单独推送
			msg := fmt.Sprintf("📊【今日营收】%s\n日期：%s\n订单：%d 单\n营业额：¥%.2f",
				st.Name, dateStr, cnt, float64(revenue)/100)
			s.sendStoreAlert(st.NotifyWebhook, msg)
		} else {
			// 未配群 → 汇总
			summary = append(summary, fmt.Sprintf("· %s：%d单 ¥%.2f", st.Name, cnt, float64(revenue)/100))
			totalRevenue += revenue
			totalCnt += cnt
		}
	}

	// 汇总推全局群
	if len(summary) > 0 && s.notifier != nil {
		msg := fmt.Sprintf("📊【今日营收汇总】%s\n%s\n──────────\n总计：%d 单  ¥%.2f",
			dateStr, strings.Join(summary, "\n"), totalCnt, float64(totalRevenue)/100)
		if err := s.notifier.Send(msg); err != nil {
			log.Printf("[REPORT] summary send failed: %v", err)
		}
	}
}

// checkVoiceReminders 按每台 ASCII 控制器配置的订单剩余时间发送语音提醒。
func (s *Scheduler) checkVoiceReminders() {
	var uc cfgModel.UIConfig
	if err := s.db.Where("store_id = 0 AND `key` = ?", "voice_config").First(&uc).Error; err != nil || uc.Value == "" {
		return
	}
	var vc config.VoiceConfig
	if err := json.Unmarshal([]byte(uc.Value), &vc); err != nil || len(vc.Devices) == 0 {
		return
	}
	voiceBySN := make(map[string]config.DeviceVoiceConfig, len(vc.Devices))
	for _, d := range vc.Devices {
		if d.Enabled {
			voiceBySN[d.DeviceSN] = d
		}
	}
	if len(voiceBySN) == 0 {
		return
	}

	now := time.Now().UnixMilli()
	var orders []order.Order
	if err := s.db.Where("status IN (?, ?) AND end_time > ?", order.StatusPaid, order.StatusInProgress, now).Find(&orders).Error; err != nil {
		log.Printf("[VOICE] 查询有效订单失败: %v", err)
		return
	}
	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		log.Printf("[VOICE] MQTT client 未就绪，无法播报")
		return
	}

	for _, o := range orders {
		var res resource.Resource
		if err := s.db.First(&res, o.ResourceID).Error; err != nil {
			continue
		}
		var ids []uint64
		var err error
		if ids, err = resource.ParseDeviceIDs(res.DeviceIDs); err != nil || len(ids) == 0 {
			continue
		}
		var devices []device.Device
		if err := s.db.Where("id IN ? AND protocol = ?", ids, device.ProtocolASCII).Find(&devices).Error; err != nil {
			log.Printf("[VOICE] 查询房间设备失败 resource=%d: %v", res.ID, err)
			continue
		}
		remain := o.EndTime - now
		seenSN := make(map[string]bool)
		for _, dev := range devices {
			if seenSN[dev.DeviceSN] {
				continue
			}
			seenSN[dev.DeviceSN] = true
			deviceConfig, configured := voiceBySN[dev.DeviceSN]
			if !configured {
				continue
			}
			for _, reminder := range matchingVoiceReminders(deviceConfig, remain) {
				key := voiceReminderKey(o.ID, dev.DeviceSN, reminder.MinutesBeforeEnd)
				reserved, err := s.redis.SetNX(s.ctx, key, 1, 24*time.Hour).Result()
				if err != nil || !reserved {
					continue
				}
				if err := cli.PublishRaw(dev.DeviceSN, reminder.Command); err != nil {
					s.redis.Del(s.ctx, key)
					log.Printf("[VOICE] 发布失败 order=%s sn=%s: %v", o.OrderNo, dev.DeviceSN, err)
					continue
				}
				log.Printf("[VOICE] 订单 %s 提醒（剩%d分钟）→ %s : %s", o.OrderNo, reminder.MinutesBeforeEnd, dev.DeviceSN, reminder.Command)
			}
		}
	}
}

func voiceReminderMatches(remainMillis int64, minutes int) bool {
	if minutes <= 0 {
		return false
	}
	target := int64(minutes) * int64(time.Minute/time.Millisecond)
	window := int64(time.Minute / time.Millisecond)
	return remainMillis > target-window && remainMillis <= target+window
}

func matchingVoiceReminders(cfg config.DeviceVoiceConfig, remainMillis int64) []config.VoiceReminder {
	if !cfg.Enabled {
		return nil
	}
	var matched []config.VoiceReminder
	for _, reminder := range cfg.Reminders {
		if reminder.Command != "" && voiceReminderMatches(remainMillis, reminder.MinutesBeforeEnd) {
			matched = append(matched, reminder)
		}
	}
	return matched
}

func voiceReminderKey(orderID uint64, deviceSN string, minutes int) string {
	return fmt.Sprintf("ssm:voice:%d:%s:%d", orderID, deviceSN, minutes)
}
