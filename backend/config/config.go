package config

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用全局配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	MQTT     MQTTConfig     `mapstructure:"mqtt"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Wechat   WechatConfig   `mapstructure:"wechat"`

	Storage   StorageConfig   `mapstructure:"storage"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Lock      LockConfig      `mapstructure:"lock"`
	Alert     AlertConfig     `mapstructure:"alert"`
	Voice     VoiceConfig     `mapstructure:"voice"`

	LoginSecurity LoginSecurityConfig `mapstructure:"login_security"`
	Elys          ElysConfig          `mapstructure:"elys"`
}

// ElysConfig 无限聚核聚合核销配置
type ElysConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	BaseURL        string `mapstructure:"base_url"`
	APIKey         string `mapstructure:"api_key"`
	WebhookAccount string `mapstructure:"webhook_account"`
	EncryptKey     string `mapstructure:"encrypt_key"`
}

// LoginSecurityConfig 登录防爆破配置
type LoginSecurityConfig struct {
	Window          time.Duration `mapstructure:"window"`
	IPMaxFailures   int           `mapstructure:"ip_max_failures"`
	CodeMaxFailures int           `mapstructure:"code_max_failures"`
	LockTTL         time.Duration `mapstructure:"lock_ttl"`
	CaptchaAfter    int           `mapstructure:"captcha_after"`
	CaptchaTTL      time.Duration `mapstructure:"captcha_ttl"`
}

// WithDefaults 返回带默认值的配置，允许配置文件缺省。
func (c LoginSecurityConfig) WithDefaults() LoginSecurityConfig {
	if c.Window <= 0 {
		c.Window = 15 * time.Minute
	}
	if c.IPMaxFailures <= 0 {
		c.IPMaxFailures = 30
	}
	if c.CodeMaxFailures <= 0 {
		c.CodeMaxFailures = 20
	}
	if c.LockTTL <= 0 {
		c.LockTTL = 30 * time.Minute
	}
	if c.CaptchaAfter <= 0 {
		c.CaptchaAfter = 3
	}
	if c.CaptchaTTL <= 0 {
		c.CaptchaTTL = 5 * time.Minute
	}
	return c
}

// VoiceConfig 按设备 SN 配置语音播报。
type VoiceConfig struct {
	Devices []DeviceVoiceConfig `mapstructure:"devices" json:"devices"`
}

// DeviceVoiceConfig 单个 ASCII 控制器的语音规则。
type DeviceVoiceConfig struct {
	DeviceSN  string          `mapstructure:"device_sn" json:"device_sn"`
	Name      string          `mapstructure:"name" json:"name,omitempty"`
	Enabled   bool            `mapstructure:"enabled" json:"enabled"`
	OnStart   string          `mapstructure:"on_start" json:"on_start"`
	OnEnd     string          `mapstructure:"on_end" json:"on_end"`
	Reminders []VoiceReminder `mapstructure:"reminders" json:"reminders"`
}

// VoiceReminder 在订单结束前指定分钟播报。
type VoiceReminder struct {
	MinutesBeforeEnd int    `mapstructure:"minutes_before_end" json:"minutes_before_end"`
	Command          string `mapstructure:"command" json:"command"`
}

// ValidateVoiceConfig 验证和归一化设备语音规则。
func ValidateVoiceConfig(vc *VoiceConfig) error {
	if vc == nil {
		return fmt.Errorf("语音配置为空")
	}
	seenSN := make(map[string]struct{}, len(vc.Devices))
	for i := range vc.Devices {
		d := &vc.Devices[i]
		d.DeviceSN = strings.TrimSpace(d.DeviceSN)
		if d.DeviceSN == "" || len(d.DeviceSN) > 64 {
			return fmt.Errorf("第%d台设备 SN 必填且最多64字符", i+1)
		}
		if _, exists := seenSN[d.DeviceSN]; exists {
			return fmt.Errorf("设备 SN 重复: %s", d.DeviceSN)
		}
		seenSN[d.DeviceSN] = struct{}{}
		if len(d.Reminders) > 10 {
			return fmt.Errorf("设备 %s 最多配置10个提醒点", d.DeviceSN)
		}
		if d.Enabled {
			if strings.TrimSpace(d.OnStart) == "" || strings.TrimSpace(d.OnEnd) == "" {
				return fmt.Errorf("设备 %s 启用时必须配置开始和结束指令", d.DeviceSN)
			}
		}
		if err := validateVoiceCommand(d.OnStart); err != nil {
			return fmt.Errorf("设备 %s 开始指令无效: %w", d.DeviceSN, err)
		}
		if err := validateVoiceCommand(d.OnEnd); err != nil {
			return fmt.Errorf("设备 %s 结束指令无效: %w", d.DeviceSN, err)
		}
		seenMinutes := make(map[int]struct{}, len(d.Reminders))
		for j := range d.Reminders {
			r := &d.Reminders[j]
			r.Command = strings.TrimSpace(r.Command)
			if r.MinutesBeforeEnd <= 0 || r.MinutesBeforeEnd > 10080 {
				return fmt.Errorf("设备 %s 提醒分钟必须在1到10080之间", d.DeviceSN)
			}
			if _, exists := seenMinutes[r.MinutesBeforeEnd]; exists {
				return fmt.Errorf("设备 %s 提醒分钟重复: %d", d.DeviceSN, r.MinutesBeforeEnd)
			}
			seenMinutes[r.MinutesBeforeEnd] = struct{}{}
			if d.Enabled && r.Command == "" {
				return fmt.Errorf("设备 %s 的%d分钟提醒指令必填", d.DeviceSN, r.MinutesBeforeEnd)
			}
			if err := validateVoiceCommand(r.Command); err != nil {
				return fmt.Errorf("设备 %s 的%d分钟提醒指令无效: %w", d.DeviceSN, r.MinutesBeforeEnd, err)
			}
		}
	}
	return nil
}

func validateVoiceCommand(command string) error {
	for _, r := range command {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("指令不能包含控制字符")
		}
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}
	if len(command) > 64 {
		return fmt.Errorf("指令最多64字节")
	}
	return nil
}

// AlertConfig 告警配置（设备离线等主动通知）
type AlertConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Webhook string `mapstructure:"webhook"` // 企业微信/钉钉机器人 URL
	Type    string `mapstructure:"type"`    // wecom | dingtalk
}

type ServerConfig struct {
	Port           int           `mapstructure:"port"`
	Mode           string        `mapstructure:"mode"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	PublicURL      string        `mapstructure:"public_url"`      // 服务器对外可访问地址
	AllowedOrigins []string      `mapstructure:"allowed_origins"` // CORS 白名单；为空则放行所有来源
	TrustedProxies []string      `mapstructure:"trusted_proxies"` // 可信反向代理网段
	EnableMock     bool          `mapstructure:"enable_mock"`     // 独立开关：模拟登录/模拟支付/平台验券回调
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	// AutoMigrate 过渡开关：显式控制是否运行 GORM AutoMigrate。
	// 未设置时回退为「release 不跑、非 release 跑」。迁移收敛完成后应设为 false 并最终移除。
	AutoMigrate *bool `mapstructure:"auto_migrate"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.DBName)
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type MQTTConfig struct {
	Broker        string        `mapstructure:"broker"`
	ClientIDPrefx string        `mapstructure:"client_id_prefix"`
	Username      string        `mapstructure:"username"`
	Password      string        `mapstructure:"password"`
	TLS           bool          `mapstructure:"tls"`
	CACert        string        `mapstructure:"ca_cert"`
	Keepalive     time.Duration `mapstructure:"keepalive"`
	PingTimeout   time.Duration `mapstructure:"ping_timeout"`
}

type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

type WechatConfig struct {
	Miniprogram MiniprogramConfig `mapstructure:"miniprogram"`
	Pay         WechatPayConfig   `mapstructure:"pay"`
}

type MiniprogramConfig struct {
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

type WechatPayConfig struct {
	MchID     string `mapstructure:"mch_id"`
	APIKey    string `mapstructure:"api_key"`
	APIV3Key  string `mapstructure:"api_v3_key"`
	CertPath  string `mapstructure:"cert_path"`
	KeyPath   string `mapstructure:"key_path"`
	NotifyURL string `mapstructure:"notify_url"`
}

type StorageConfig struct {
	Type         string `mapstructure:"type"`
	LocalPath    string `mapstructure:"local_path"`
	OSSEndpoint  string `mapstructure:"oss_endpoint"`
	OSSAccessKey string `mapstructure:"oss_access_key"`
	OSSSecretKey string `mapstructure:"oss_secret_key"`
	OSSBucket    string `mapstructure:"oss_bucket"`
}

type SchedulerConfig struct {
	OrderTimeoutCheckInterval time.Duration `mapstructure:"order_timeout_check_interval"`
	DeviceHealthCheckInterval time.Duration `mapstructure:"device_health_check_interval"`
	AutoSettleInterval        time.Duration `mapstructure:"auto_settle_interval"`
	OrderPendingPayTTL        time.Duration `mapstructure:"order_pending_pay_ttl"`
	DailyReportHour           int           `mapstructure:"daily_report_hour"`    // 每日营收播报时间（0-23，默认 22）
	DailyReportEnabled        bool          `mapstructure:"daily_report_enabled"` // 是否开启每日营收播报（默认关）
}

type LockConfig struct {
	BookingTTL           time.Duration `mapstructure:"booking_ttl"`
	PaymentIdempotentTTL time.Duration `mapstructure:"payment_idempotent_ttl"`
}

// Load 从指定路径加载配置，支持 YAML 格式
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// 允许环境变量覆盖 — 如 SSM_SERVER_PORT=9090
	// 需要把配置 key 的 "." 映射为 "_"，否则 viper 会查找 "SSM_SERVER.MODE" 而非 "SSM_SERVER_MODE"
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetEnvPrefix("SSM")

	for _, key := range []string{
		"elys.enabled",
		"elys.base_url",
		"elys.api_key",
		"elys.webhook_account",
		"elys.encrypt_key",
	} {
		_ = v.BindEnv(key)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// 安全校验：生产模式下敏感值不能使用默认占位符
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return &cfg, nil
}

// Validate 校验配置安全性 — 非 debug 模式下敏感值不得为空或使用已知默认值
func (c *Config) Validate() error {
	if c.Server.Mode == "debug" {
		return nil // debug 模式允许宽松配置
	}

	if c.Database.Password == "" || c.Database.Password == "root123" {
		return fmt.Errorf("database.password 不能为空或使用默认值，请通过环境变量 SSM_DATABASE_PASSWORD 设置")
	}
	if c.JWT.Secret == "" || c.JWT.Secret == "change-me-in-production" {
		return fmt.Errorf("jwt.secret 不能为空或使用默认值，请通过环境变量 SSM_JWT_SECRET 设置")
	}

	if c.Elys.Enabled {
		if c.Elys.APIKey == "" {
			return fmt.Errorf("elys.api_key 不能为空")
		}
		key, err := base64.StdEncoding.DecodeString(c.Elys.EncryptKey)
		if err != nil || len(key) != 32 {
			return fmt.Errorf("elys.encrypt_key 必须是 base64 编码的 32 字节密钥")
		}
	}

	return nil
}
