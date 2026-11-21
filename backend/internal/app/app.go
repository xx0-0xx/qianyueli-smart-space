// Package app 依赖注入与路由装配
package app

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"smart-space-management/backend/config"
	"smart-space-management/backend/internal/device"
	"smart-space-management/backend/internal/handler"
	"smart-space-management/backend/internal/handler/admin"
	wsh "smart-space-management/backend/internal/handler/ws"
	"smart-space-management/backend/internal/middleware"
	auditModel "smart-space-management/backend/internal/model/audit"
	balanceModel "smart-space-management/backend/internal/model/balance"
	cfgModel "smart-space-management/backend/internal/model/config"
	devModel "smart-space-management/backend/internal/model/device"
	elysModel "smart-space-management/backend/internal/model/elys"
	leadModel "smart-space-management/backend/internal/model/lead"
	"smart-space-management/backend/internal/model/order"
	payModel "smart-space-management/backend/internal/model/payment"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/internal/model/store"
	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/internal/router"
	"smart-space-management/backend/internal/scheduler"
	"smart-space-management/backend/internal/service"
	"smart-space-management/backend/internal/ws"
	"smart-space-management/backend/pkg/alert"
	"smart-space-management/backend/pkg/migrate"
	"smart-space-management/backend/pkg/secretbox"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// App 应用程序容器
type App struct {
	cfg            *config.Config
	DB             *gorm.DB
	Redis          *redis.Client
	JWT            *utils.JWTManager
	Hub            *ws.Hub
	Engine         *gin.Engine
	Server         *http.Server
	ElysVoucherSvc *service.ElysVoucherService
}

// New 创建并装配应用
func New(cfg *config.Config) (*App, error) {
	// ── 初始化雪花 ID ──
	if err := utils.InitSnowflake(1); err != nil {
		return nil, fmt.Errorf("init snowflake: %w", err)
	}

	// ── 数据库 ──
	db, err := initDB(cfg)
	if err != nil {
		return nil, err
	}

	// ── Redis ──
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	log.Println("[APP] redis connected")

	// ── JWT ──
	jwtMgr := utils.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)

	// ── WebSocket Hub ──
	hub := ws.NewHub(rdb)

	// ── Gin 引擎 ──
	gin.SetMode(cfg.Server.Mode)
	engine := gin.New()

	// 可信反向代理网段：确保 c.ClientIP() 只信任 nginx/docker 网络转发的 X-Forwarded-For，
	// 防止外部伪造来源 IP 绕过限流或污染审计日志。
	trusted := cfg.Server.TrustedProxies
	if len(trusted) == 0 {
		trusted = []string{"127.0.0.1", "172.16.0.0/12", "10.0.0.0/8"}
	}
	if err := engine.SetTrustedProxies(trusted); err != nil {
		log.Printf("[APP] SetTrustedProxies 失败: %v", err)
	}

	// ── 全局中间件 ──
	engine.Use(
		middleware.RequestID(),
		middleware.CORS(cfg.Server.AllowedOrigins),
		middleware.BodyLimit(52<<20), // 52 MB（与 nginx client_max_body_size 50m 对齐）
		middleware.Logger(),
		middleware.Recovery(),
		middleware.Audit(db),
		middleware.RateLimit(middleware.NewPerClientStore()),
	)

	// ── 路由 ──
	balanceSvc := service.NewBalanceService(db)
	elysClient := service.NewElysClient(cfg.Elys.BaseURL, cfg.Elys.APIKey)
	elysBox, err := secretbox.New(cfg.Elys.EncryptKey)
	if err != nil && cfg.Elys.Enabled {
		return nil, fmt.Errorf("elys encrypt key: %w", err)
	}
	elysVoucherSvc := service.NewElysVoucherService(db, rdb, balanceSvc, elysClient, elysBox, cfg.Elys.Enabled)

	handlerCtx := &handler.Context{
		DB:    db,
		Redis: rdb,
		JWT:   jwtMgr,
		Hub:   hub,
		Cfg:   cfg,
		MQTT:  nil, // MQTT 客户端在 Run() 中注入

		OrderSvc:       service.NewOrderService(db, rdb),
		DeviceSvc:      service.NewDeviceService(db, cfg.Voice),
		StoreSvc:       service.NewStoreService(db),
		BalanceSvc:     balanceSvc,
		ElysBindingSvc: service.NewElysBindingService(db, elysClient),
		ElysVoucherSvc: elysVoucherSvc,
	}

	apiRouter := router.NewAPIRouter(handlerCtx)
	adminRouter := router.NewAdminRouter(handlerCtx)

	api := engine.Group("/api/v1")
	apiRouter.Register(api)

	// 管理员公开接口（登录）
	adminPublic := engine.Group("/api/v1/admin")
	adminRouter.RegisterPublic(adminPublic)

	// 管理员鉴权接口
	adm := engine.Group("/api/v1/admin")
	adm.Use(middleware.Auth(jwtMgr), middleware.AdminOnly())
	adminRouter.Register(adm)

	// ── WebSocket ──
	wsHandler := wsh.NewWSHandler(hub, jwtMgr, rdb)
	engine.GET("/ws", func(c *gin.Context) {
		wsHandler.Serve(c.Writer, c.Request)
	})

	// ── 音频文件服务 ──
	engine.Static("/audio", "./audio")

	// ── 健康检查（含关键依赖探活）──
	engine.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		checks := gin.H{}
		healthy := true

		if sqlDB, err := db.DB(); err != nil {
			healthy = false
			checks["database"] = "down"
		} else if err := sqlDB.PingContext(ctx); err != nil {
			healthy = false
			checks["database"] = "down"
		} else {
			checks["database"] = "up"
		}

		if err := rdb.Ping(ctx).Err(); err != nil {
			healthy = false
			checks["redis"] = "down"
		} else {
			checks["redis"] = "up"
		}

		if healthy {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "checks": checks, "time": time.Now().Unix()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "checks": checks, "time": time.Now().Unix()})
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{
		cfg:            cfg,
		DB:             db,
		Redis:          rdb,
		JWT:            jwtMgr,
		Hub:            hub,
		Engine:         engine,
		Server:         server,
		ElysVoucherSvc: elysVoucherSvc,
	}, nil
}

// Run 启动 HTTP 服务和 WS Hub，等待退出信号
func (a *App) Run() error {
	// 启动调度器（订单超时清理、设备心跳检查、自动结算）
	notifier := alert.New(a.cfg.Alert)
	sched := scheduler.New(a.DB, a.Redis, a.cfg.Scheduler, notifier, a.cfg.Voice)
	sched.Start()
	defer sched.Stop()

	// 启动 WS Hub
	go a.Hub.Run()

	// 启动 MQTT 客户端（连接 EMQX，接收设备状态）
	mqttClient, err := device.NewClient(a.cfg.MQTT)
	if err != nil {
		log.Printf("[APP] MQTT 连接失败（非致命）: %v", err)
	} else {
		// 注入 MQTT 客户端到全局，让 handler 能直接使用
		device.SetGlobalClient(mqttClient)

		mqttClient.Subscribe(
			[]string{"ssm/device/+/status", "ssm/device/+/report", "ssm/device/+/event"},
			func(deviceSN string, payload map[string]interface{}) {
				// 设备状态更新 → 写 Redis + 推送 WS
				raw, _ := json.Marshal(payload)
				a.Redis.Set(context.Background(), "ssm:device:status:"+deviceSN, string(raw), 2*time.Minute)
				ws.PublishDeviceState(a.Redis, 0, 0, payload)

				// 更新所有同SN设备的在线状态
				now := time.Now().UnixMilli()
				result := a.DB.Model(&devModel.Device{}).Where("device_sn = ?", deviceSN).Updates(map[string]interface{}{"status": 1, "last_online_at": now})
				if result.RowsAffected == 0 {
					// 新设备自动注册
					dev := devModel.Device{
						DeviceSN: deviceSN, Name: deviceSN, Type: devModel.TypeOther,
						StoreID: 0, Status: 1, LastOnlineAt: &now,
					}
					dev.ID = utils.ShortID()
					a.DB.Create(&dev)
					log.Printf("[MQTT] 新设备自动注册: %s", deviceSN)
				}
			},
			func(deviceSN, eventType string, payload map[string]interface{}) {
				log.Printf("[MQTT] device %s event %s: %v", deviceSN, eventType, payload)
			},
			func(deviceSN, text string) {
				// 成品控制器 ASCII 上报（n1/f1/gs/rest_ok/状态串）→ 标记设备在线
				now := time.Now().UnixMilli()
				a.DB.Model(&devModel.Device{}).Where("device_sn = ?", deviceSN).
					Updates(map[string]interface{}{"status": 1, "last_online_at": now})
				a.Redis.Set(context.Background(), "ssm:device:status:"+deviceSN, text, 2*time.Minute)
				log.Printf("[MQTT] ASCII 上报 %s: %s", deviceSN, text)
			},
			func(clientID string, online bool) {
				deviceSN := device.DeviceSNFromMQTTClientID(clientID)
				if deviceSN == "" {
					return
				}
				status := devModel.StatusOffline
				updates := map[string]interface{}{"status": status}
				if online {
					status = devModel.StatusOnline
					now := time.Now().UnixMilli()
					updates["last_online_at"] = now
				}
				updates["status"] = status
				result := a.DB.Model(&devModel.Device{}).Where("device_sn = ?", deviceSN).Updates(updates)
				if result.Error != nil {
					log.Printf("[MQTT] 更新客户端状态失败 SN=%s: %v", deviceSN, result.Error)
					return
				}
				if result.RowsAffected == 0 && online {
					var existing int64
					if err := a.DB.Model(&devModel.Device{}).Where("device_sn = ?", deviceSN).Count(&existing).Error; err == nil && existing == 0 {
						now := time.Now().UnixMilli()
						dev := devModel.Device{DeviceSN: deviceSN, Name: deviceSN, Type: devModel.TypeOther, StoreID: 0, Status: devModel.StatusOnline, LastOnlineAt: &now}
						dev.ID = utils.ShortID()
						if err := a.DB.Create(&dev).Error; err != nil {
							log.Printf("[MQTT] 注册连接设备失败 SN=%s: %v", deviceSN, err)
						}
					}
				}
				log.Printf("[MQTT] 设备连接状态 SN=%s online=%v", deviceSN, online)
			},
		)
		defer mqttClient.Close()
	}

	// 启动 HTTP
	go func() {
		log.Printf("[APP] server listening on %s", a.Server.Addr)
		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[APP] server error: %v", err)
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	retryCtx, retryCancel := context.WithCancel(context.Background())
	if a.ElysVoucherSvc != nil && a.cfg.Elys.Enabled {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					a.ElysVoucherSvc.RetryPendingCredits(retryCtx)
				case <-retryCtx.Done():
					return
				}
			}
		}()
	}

	<-quit
	retryCancel()
	log.Println("[APP] shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a.Hub.Stop()
	if err := a.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	log.Println("[APP] server exited")
	return nil
}

func initDB(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Info
	if cfg.Server.Mode == "release" {
		logLevel = logger.Warn
	}

	db, err := gorm.Open(mysql.Open(cfg.Database.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	log.Println("[APP] database connected")

	// 版本化迁移（幂等：已应用的自动跳过）
	migrator := migrate.New(db, "migrations")
	if err := migrator.Run(); err != nil {
		log.Printf("[APP] migrations warning: %v", err)
	}

	// 自动建表（过渡开关）：迁移收敛后应关闭；未显式配置时回退为「release 不跑、非 release 跑」
	if autoMigrateEnabled(cfg) {
		if err := db.AutoMigrate(autoMigrateModels()...); err != nil {
			log.Printf("[APP] AutoMigrate warning: %v", err)
		}
	} else {
		log.Println("[APP] AutoMigrate 已关闭（依赖版本化迁移 migrations/）")
	}

	// 回填历史门店的邀请码（InviteCode 字段后加，老数据为空）
	backfillInviteCodes(db)
	// 将历史短邀请码统一轮换为 10 位
	rotateShortInviteCodes(db)

	// 种子数据：确保至少有一个超级管理员
	seedAdminUser(db)

	// 回填已有 admin 的密码哈希（password_hash 字段为后加，老数据为空）
	backfillAdminPasswords(db)

	// 种子数据：内置产品模板
	seedProductTemplates(db)

	// 余额组与账户回填（幂等）
	backfillBalanceGroups(db)
	backfillBalanceAccounts(db)

	return db, nil
}

// autoMigrateEnabled 是否运行 AutoMigrate（过渡开关）
func autoMigrateEnabled(cfg *config.Config) bool {
	if cfg.Database.AutoMigrate != nil {
		return *cfg.Database.AutoMigrate
	}
	return cfg.Server.Mode != "release"
}

// autoMigrateModels 需要 AutoMigrate 的模型（过渡期使用；迁移收敛后应移除本函数）
func autoMigrateModels() []interface{} {
	return []interface{}{
		&user.User{},
		&store.Store{},
		&resource.ResourceType{},
		&resource.Resource{},
		&devModel.Device{},
		&order.Order{},
		&cfgModel.PriceRule{},
		&cfgModel.BookingRule{},
		&cfgModel.UIConfig{},
		&cfgModel.ConfigHistory{},
		&payModel.Payment{},
		&payModel.RefundRequest{},
		&payModel.Coupon{},
		&payModel.Recharge{},
		&elysModel.StoreBinding{},
		&elysModel.VoucherVerification{},
		&elysModel.WebhookEvent{},
		&payModel.BalanceLog{},
		&auditModel.AuditLog{},
		&devModel.ProductTemplate{},
		&user.MerchantStore{},
		&leadModel.Lead{},
		&balanceModel.BalanceGroup{},
		&balanceModel.Account{},
	}
}

// seedProductTemplates 内置产品模板（全局 store_id=0，商家绑定设备时可选）
func seedProductTemplates(db *gorm.DB) {
	var count int64
	db.Model(&devModel.ProductTemplate{}).Count(&count)
	if count > 0 {
		return
	}
	seed := []struct {
		name    string
		devices []devModel.TemplateDevice
	}{
		{"电磁门锁", []devModel.TemplateDevice{{Name: "门锁", GPIO: 4, CmdOn: "door_open", CmdOff: "door_close", Type: "door_lock", IsToggle: true}}},
		{"插座", []devModel.TemplateDevice{{Name: "插座", GPIO: 4, CmdOn: "machine_on", CmdOff: "machine_off", Type: "machine", IsToggle: true}}},
		{"灯开关", []devModel.TemplateDevice{{Name: "灯", GPIO: 5, CmdOn: "light_on", CmdOff: "light_off", Type: "light", IsToggle: true}}},
	}
	for _, s := range seed {
		t := devModel.ProductTemplate{Name: s.name, StoreID: 0, Status: 1}
		t.SetDevices(s.devices)
		t.ID = utils.ShortID()
		db.Create(&t)
	}
	log.Printf("[APP] seeded %d product templates", len(seed))
}

// backfillBalanceGroups 为无余额组的门店各建一个独立余额组
func backfillBalanceGroups(db *gorm.DB) {
	var stores []store.Store
	if err := db.Where("balance_group_id = 0").Find(&stores).Error; err != nil {
		log.Printf("[APP] backfill balance groups query: %v", err)
		return
	}
	for _, s := range stores {
		g := balanceModel.BalanceGroup{Name: s.Name}
		g.ID = utils.ShortID()
		if err := db.Create(&g).Error; err != nil {
			log.Printf("[APP] backfill balance group for store %d: %v", s.ID, err)
			continue
		}
		db.Model(&store.Store{}).Where("id = ?", s.ID).Update("balance_group_id", g.ID)
	}
}

// backfillBalanceAccounts 把 users.balance 迁入账户（幂等；归属"最近充值门店"所属组）
func backfillBalanceAccounts(db *gorm.DB) {
	var users []user.User
	if err := db.Where("balance <> 0").Find(&users).Error; err != nil || len(users) == 0 {
		return
	}
	for _, u := range users {
		var rc payModel.Recharge
		var storeID uint64
		if db.Where("user_id = ?", u.ID).Order("created_at DESC").First(&rc).Error == nil {
			storeID = rc.StoreID
		} else {
			var minStore store.Store
			if db.Order("id ASC").First(&minStore).Error != nil {
				continue
			}
			storeID = minStore.ID
		}
		gid, err := service.ResolveGroupID(db, storeID)
		if err != nil {
			log.Printf("[APP] backfill balance: user %d 无法解析余额组: %v", u.ID, err)
			continue
		}
		var cnt int64
		db.Model(&balanceModel.Account{}).Where("user_id = ? AND group_id = ?", u.ID, gid).Count(&cnt)
		if cnt > 0 {
			continue
		}
		acc := balanceModel.Account{UserID: u.ID, GroupID: gid, Balance: u.Balance}
		acc.ID = utils.ShortID()
		if err := db.Create(&acc).Error; err != nil {
			log.Printf("[APP] backfill balance account user=%d: %v", u.ID, err)
			continue
		}
		db.Model(&user.User{}).Where("id = ?", u.ID).Update("balance", 0)
	}
}

// backfillInviteCodes 为所有邀请码为空的历史门店生成邀请码
func backfillInviteCodes(db *gorm.DB) {
	var stores []store.Store
	// 只查空邀请码门店（尽量少扫描；大表可加 LIMIT 分批）
	if err := db.Where("invite_code = '' OR invite_code IS NULL").Find(&stores).Error; err != nil {
		log.Printf("[APP] backfill invite_code query: %v", err)
		return
	}
	if len(stores) == 0 {
		return
	}
	for i := range stores {
		code := genUniqueInviteCodeDB(db)
		if err := db.Model(&store.Store{}).Where("id = ?", stores[i].ID).Update("invite_code", code).Error; err != nil {
			log.Printf("[APP] backfill invite_code for store %d: %v", stores[i].ID, err)
			continue
		}
		log.Printf("[APP] backfill invite_code for store %d -> %s", stores[i].ID, code)
	}
}

// rotateShortInviteCodes 将历史 6 位邀请码统一轮换为 10 位。
// 该操作会使旧邀请码立即失效，属于有意的安全升级。
func rotateShortInviteCodes(db *gorm.DB) {
	var stores []store.Store
	if err := db.Where("invite_code IS NOT NULL AND invite_code <> '' AND CHAR_LENGTH(invite_code) < 10").
		Find(&stores).Error; err != nil {
		log.Printf("[SECURITY] 查询短邀请码失败: %v", err)
		return
	}
	if len(stores) == 0 {
		return
	}

	rotated := 0
	for i := range stores {
		code := genUniqueInviteCodeDB(db)
		if err := db.Model(&store.Store{}).Where("id = ?", stores[i].ID).
			Update("invite_code", code).Error; err != nil {
			log.Printf("[SECURITY] 轮换门店 %d 邀请码失败: %v", stores[i].ID, err)
			continue
		}
		rotated++
	}
	log.Printf("[SECURITY] 已将 %d 个短邀请码轮换为 10 位，请通知商家重新获取邀请码", rotated)
}

// genUniqueInviteCodeDB 在数据库范围内生成唯一邀请码
func genUniqueInviteCodeDB(db *gorm.DB) string {
	for i := 0; i < 10; i++ {
		code := utils.GenInviteCode()
		var count int64
		db.Model(&store.Store{}).Where("invite_code = ?", code).Count(&count)
		if count == 0 {
			return code
		}
	}
	return utils.GenInviteCode() + "-" + strconv.FormatUint(utils.ShortID(), 36)
}

// seedAdminUser 确保至少有一个超级管理员 — 首次启动时自动创建
func seedAdminUser(db *gorm.DB) {
	var count int64
	db.Model(&user.User{}).Where("role IN ?", []string{"admin", "super_admin"}).Count(&count)
	if count > 0 {
		return
	}

	password := os.Getenv("SSM_ADMIN_INITIAL_PASSWORD")
	fromEnv := password != ""
	if !fromEnv {
		password = randomPassword()
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[APP] seed admin: bcrypt hash failed: %v", err)
		return
	}

	admin := user.User{
		Nickname:     "admin",
		OpenID:       "seed_admin_" + strconv.FormatUint(utils.ShortID(), 36),
		Role:         user.RoleSuperAdmin,
		Status:       user.StatusNormal,
		PasswordHash: string(hash),
	}
	admin.ID = utils.ShortID()
	admin.CreatedAt = time.Now()
	admin.UpdatedAt = time.Now()

	if err := db.Create(&admin).Error; err != nil {
		log.Printf("[APP] seed admin: create failed: %v", err)
		return
	}
	if fromEnv {
		log.Println("[APP] seed admin: 已用 SSM_ADMIN_INITIAL_PASSWORD 创建超级管理员 admin")
	} else {
		recordInitialCredential("admin", password, admin.ID)
		log.Printf("[APP] seed admin: 已创建超级管理员 admin，初始密码写入 %s（请登录后立即修改并删除该文件）", initialCredFile)
	}
}

// initialCredFile 初始凭据落盘文件名（相对于工作目录），0600 权限，避免明文口令进入容器日志
const initialCredFile = "initial-credentials.txt"

// recordInitialCredential 将初始凭据追加写入受限权限文件
func recordInitialCredential(nickname, password string, id uint64) {
	f, err := os.OpenFile(initialCredFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		log.Printf("[APP] 无法写入初始凭据文件 %s: %v", initialCredFile, err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\t%s\t%d\n", nickname, password, id)
}

// randomPassword 生成随机初始密码（避免使用固定弱口令）
func randomPassword() string {
	b := make([]byte, 12)
	if _, err := crand.Read(b); err != nil {
		return "Ssm-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// backfillAdminPasswords 为所有 password_hash 为空的 admin/merchant 用户生成默认密码哈希
func backfillAdminPasswords(db *gorm.DB) {
	var users []user.User
	if err := db.Where("role IN ? AND (password_hash = '' OR password_hash IS NULL)",
		[]string{"admin", "super_admin", "merchant"}).Find(&users).Error; err != nil {
		log.Printf("[APP] backfill admin passwords query: %v", err)
		return
	}
	if len(users) == 0 {
		return
	}

	for _, u := range users {
		pw := randomPassword()
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[APP] backfill admin password for user %d: bcrypt failed: %v", u.ID, err)
			continue
		}
		if err := db.Model(&user.User{}).Where("id = ?", u.ID).
			Update("password_hash", string(hash)).Error; err != nil {
			log.Printf("[APP] backfill admin password for user %d: %v", u.ID, err)
			continue
		}
		recordInitialCredential(u.Nickname, pw, u.ID)
	}
	if len(users) > 0 {
		log.Printf("[APP] backfill admin passwords: 已为 %d 个账号重置随机密码，凭据写入 %s（请尽快修改并删除该文件）", len(users), initialCredFile)
	}
}

// 确保编译期检查
var _ = admin.AdminHandler{}
