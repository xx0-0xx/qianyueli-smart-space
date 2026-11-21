package migrate

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Runner 数据库迁移执行器
type Runner struct {
	DB          *gorm.DB
	MigrationsDir string
}

// New 创建迁移执行器
func New(db *gorm.DB, dir string) *Runner {
	return &Runner{DB: db, MigrationsDir: dir}
}

// Run 执行所有未应用的迁移
func (r *Runner) Run() error {
	// 确保 schema_migrations 表存在
	if err := r.DB.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// 读取已应用的迁移
	var applied []string
	r.DB.Model(&struct{ Version string }{}).
		Table("schema_migrations").
		Pluck("version", &applied)
	appliedSet := make(map[string]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	// 扫描迁移文件
	entries, err := os.ReadDir(r.MigrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// 逐个执行
	for _, f := range files {
		if appliedSet[f] {
			continue
		}

		path := filepath.Join(r.MigrationsDir, f)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}

		log.Printf("[MIGRATE] applying %s ...", f)

		// 逐条执行：go-sql-driver 默认不支持单次多语句，且逐条可对已存在的对象容错
		statements := splitStatements(string(sql))
		if err := r.DB.Transaction(func(tx *gorm.DB) error {
			for _, stmt := range statements {
				if err := tx.Exec(stmt).Error; err != nil {
					if isIdempotentErr(err) {
						log.Printf("[MIGRATE] %s: 语句已生效，跳过 (%v)", f, err)
						continue
					}
					return fmt.Errorf("exec %s: %w", f, err)
				}
			}
			if err := tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
				f, time.Now()).Error; err != nil {
				return fmt.Errorf("record migration %s: %w", f, err)
			}
			return nil
		}); err != nil {
			return err
		}

		log.Printf("[MIGRATE] %s applied", f)
	}

	return nil
}

// splitStatements 将 SQL 脚本按 ';' 拆分为独立语句，
// 正确跳过字符串字面量、反引号标识符与注释中的分号。
func splitStatements(sql string) []string {
	var (
		out        []string
		cur        strings.Builder
		inSingle   bool
		inDouble   bool
		inBacktick bool
		inLineCmnt bool
		inBlockCmt bool
	)
	runes := []rune(sql)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if !inSingle && !inDouble && !inBacktick {
			if inLineCmnt {
				if ch == '\n' {
					inLineCmnt = false
					cur.WriteRune(ch)
				}
				continue
			}
			if inBlockCmt {
				if ch == '*' && i+1 < len(runes) && runes[i+1] == '/' {
					inBlockCmt = false
					i++
				}
				continue
			}
			if ch == '-' && i+1 < len(runes) && runes[i+1] == '-' {
				inLineCmnt = true
				i++
				continue
			}
			if ch == '/' && i+1 < len(runes) && runes[i+1] == '*' {
				inBlockCmt = true
				i++
				continue
			}
		}
		switch ch {
		case '\'':
			if !inDouble && !inBacktick {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && !inBacktick {
				inDouble = !inDouble
			}
		case '`':
			if !inSingle && !inDouble {
				inBacktick = !inBacktick
			}
		case ';':
			if !inSingle && !inDouble && !inBacktick {
				if s := strings.TrimSpace(cur.String()); s != "" {
					out = append(out, s)
				}
				cur.Reset()
				continue
			}
		}
		cur.WriteRune(ch)
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		out = append(out, s)
	}
	return out
}

// isIdempotentErr 判断错误是否属于「对象已存在/已生效」，
// 这类错误在重复执行幂等 DDL 时可以安全忽略。
func isIdempotentErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, s := range []string{
		"Duplicate column name",
		"Duplicate key name",
		"already exists",
		"Duplicate entry",
		"check that column/key exists",
		"Can't DROP",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
