package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"smart-space-management/backend/config"

	"github.com/go-redis/redis/v8"
)

const (
	loginFailIPKeyPrefix   = "ssm:login:%s:fail:ip:"
	loginFailSubKeyPrefix  = "ssm:login:%s:fail:subject:"
	loginLockIPKeyPrefix   = "ssm:login:%s:lock:ip:"
	loginLockSubKeyPrefix  = "ssm:login:%s:lock:subject:"
	loginCaptchaKeyPrefix  = "ssm:login:captcha:"
	loginCaptchaQuestion   = "%d + %d = ?"
	LoginScopeInvite       = "invite"
	LoginScopeAdmin        = "admin"
	loginCaptchaAnswerBits = 16
)

// LoginGuard 登录防爆破守卫，商家邀请码登录和管理员账号登录共用。
type LoginGuard struct {
	redis *redis.Client
	cfg   config.LoginSecurityConfig
}

func NewLoginGuard(redisClient *redis.Client, cfg config.LoginSecurityConfig) *LoginGuard {
	return &LoginGuard{redis: redisClient, cfg: cfg.WithDefaults()}
}

func (h *UserHandler) loginGuard() *LoginGuard {
	cfg := config.LoginSecurityConfig{}
	if h.ctx != nil && h.ctx.Cfg != nil {
		cfg = h.ctx.Cfg.LoginSecurity
	}
	var redisClient *redis.Client
	if h.ctx != nil {
		redisClient = h.ctx.Redis
	}
	return NewLoginGuard(redisClient, cfg)
}

func normalizeInviteCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func normalizeLoginSubject(subject string) string {
	return strings.ToLower(strings.TrimSpace(subject))
}

func subjectHash(subject string) string {
	sum := sha256.Sum256([]byte(normalizeLoginSubject(subject)))
	return hex.EncodeToString(sum[:])
}

func (g *LoginGuard) failIPKey(scope, ip string) string {
	return fmt.Sprintf(loginFailIPKeyPrefix, scope) + ip
}

func (g *LoginGuard) failSubjectKey(scope, subject string) string {
	return fmt.Sprintf(loginFailSubKeyPrefix, scope) + subjectHash(subject)
}

func (g *LoginGuard) lockIPKey(scope, ip string) string {
	return fmt.Sprintf(loginLockIPKeyPrefix, scope) + ip
}

func (g *LoginGuard) lockSubjectKey(scope, subject string) string {
	return fmt.Sprintf(loginLockSubKeyPrefix, scope) + subjectHash(subject)
}

// IsLocked 检查 IP 或登录主体是否已被锁定。
func (g *LoginGuard) IsLocked(ctx context.Context, scope, ip, subject string) bool {
	if g.redis == nil {
		return false
	}
	keys := []string{g.lockIPKey(scope, ip), g.lockSubjectKey(scope, subject)}
	n, err := g.redis.Exists(ctx, keys...).Result()
	if err != nil {
		log.Printf("[LOGIN] 检查锁定状态失败: %v", err)
		return false
	}
	return n > 0
}

// NeedsCaptcha 判断当前 IP/登录主体是否已触发验证码。
func (g *LoginGuard) NeedsCaptcha(ctx context.Context, scope, ip, subject string) bool {
	if g.redis == nil {
		return false
	}
	values, err := g.redis.MGet(ctx, g.failIPKey(scope, ip), g.failSubjectKey(scope, subject)).Result()
	if err != nil {
		log.Printf("[LOGIN] 读取失败计数失败: %v", err)
		return false
	}
	for _, value := range values {
		if n, ok := value.(string); ok {
			if count, err := strconv.Atoi(n); err == nil && count >= g.cfg.CaptchaAfter {
				return true
			}
		}
	}
	return false
}

// RecordFailure 记录一次失败，达到阈值时锁定 IP/登录主体。
func (g *LoginGuard) RecordFailure(ctx context.Context, scope, ip, subject string) {
	if g.redis == nil {
		return
	}
	ipCount := g.incrWithWindow(ctx, g.failIPKey(scope, ip))
	subjectCount := g.incrWithWindow(ctx, g.failSubjectKey(scope, subject))
	if ipCount >= g.cfg.IPMaxFailures {
		if err := g.redis.Set(ctx, g.lockIPKey(scope, ip), "1", g.cfg.LockTTL).Err(); err != nil {
			log.Printf("[LOGIN] 锁定 IP 失败: %v", err)
		}
	}
	if subjectCount >= g.cfg.CodeMaxFailures {
		if err := g.redis.Set(ctx, g.lockSubjectKey(scope, subject), "1", g.cfg.LockTTL).Err(); err != nil {
			log.Printf("[LOGIN] 锁定登录主体失败: %v", err)
		}
	}
}

func (g *LoginGuard) incrWithWindow(ctx context.Context, key string) int {
	pipe := g.redis.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, g.cfg.Window)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("[LOGIN] 写入失败计数失败: %v", err)
		return 0
	}
	value, err := incr.Result()
	if err != nil {
		return 0
	}
	return int(value)
}

// Clear 登录成功后清理失败计数与锁定。
func (g *LoginGuard) Clear(ctx context.Context, scope, ip, subject string) {
	if g.redis == nil {
		return
	}
	keys := []string{
		g.failIPKey(scope, ip),
		g.failSubjectKey(scope, subject),
		g.lockIPKey(scope, ip),
		g.lockSubjectKey(scope, subject),
	}
	if err := g.redis.Del(ctx, keys...).Err(); err != nil {
		log.Printf("[LOGIN] 清理登录失败计数失败: %v", err)
	}
}

// NewCaptcha 生成一个简单的算术验证码挑战。
func (g *LoginGuard) NewCaptcha(ctx context.Context) map[string]string {
	if g.redis == nil {
		return nil
	}
	a := randomInt(1, 9)
	b := randomInt(1, 9)
	id := randomHex(loginCaptchaAnswerBits)
	answer := strconv.Itoa(a + b)
	if err := g.redis.Set(ctx, loginCaptchaKeyPrefix+id, answer, g.cfg.CaptchaTTL).Err(); err != nil {
		log.Printf("[LOGIN] 写入验证码失败: %v", err)
		return nil
	}
	return map[string]string{
		"captcha_id": id,
		"question":   fmt.Sprintf(loginCaptchaQuestion, a, b),
	}
}

// VerifyCaptcha 校验并消费验证码。
func (g *LoginGuard) VerifyCaptcha(ctx context.Context, id, answer string) bool {
	if g.redis == nil || id == "" || answer == "" {
		return false
	}
	key := loginCaptchaKeyPrefix + id
	expected, err := g.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false
	}
	if err != nil {
		log.Printf("[LOGIN] 读取验证码失败: %v", err)
		return false
	}
	// 验证码一次性使用，无论对错都删除，避免重复尝试。
	_ = g.redis.Del(ctx, key).Err()
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.TrimSpace(answer))) == 1
}

func randomInt(min, max int) int {
	if max <= min {
		return min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return min
	}
	return min + int(n.Int64())
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buf)
}
