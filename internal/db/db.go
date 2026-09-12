package db

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"

	"metrobase/internal/auth"
)

// Open 打开本地 SQLite 数据库并完成：审计列自动填充回调、读写分离注册、建表前的目录准备。
// SQLite 下读写连接指向同一文件；部署到 MySQL 时把 replica DSN 换成从库即可，业务代码不变。
func Open(dsn string, enableReplica bool) (*gorm.DB, error) {
	if dsn == "" {
		dsn = "./data/metro.db"
	}
	if !strings.Contains(dsn, ":memory:") {
		if dir := filepath.Dir(dsn); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
	}
	gdb, err := gorm.Open(sqlite.Open(sqlitePragmas(dsn)), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, err
	}
	if enableReplica {
		// 读连接注册：显式 dbresolver.Read 的查询走 replica，其余走 source（主库）。
		// SQLite 单机部署 replica 与 source 同文件，机制保留、部署期替换。
		err = gdb.Use(dbresolver.Register(dbresolver.Config{
			Sources:  []gorm.Dialector{sqlite.Open(sqlitePragmas(dsn))},
			Replicas: []gorm.Dialector{sqlite.Open(sqlitePragmas(dsn))},
			Policy:   dbresolver.RandomPolicy{},
		}))
		if err != nil {
			return nil, err
		}
	}
	registerAuditCallbacks(gdb)
	if sqlDB, err := gdb.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1) // SQLite 写并发串行化，避免 SQLITE_BUSY
		sqlDB.SetMaxIdleConns(1)
	}
	return gdb, nil
}

// sqlitePragmas 打开 WAL 与忙等待，保证 HTTP 服务连接与独立查询连接
// 并发访问同一个数据库文件。
func sqlitePragmas(dsn string) string {
	if strings.Contains(dsn, "_pragma") {
		return dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

// registerAuditCallbacks 注册审计列自动填充（创建人/创建时间、更新人/更新时间）。
// 取值只认请求 context 里的登录身份；且只在列为空时填充——业务代码若手动写入
// 非空值，该值会保留（不会被回调覆盖）。因此正确写法是业务层不碰这四列。
func registerAuditCallbacks(gdb *gorm.DB) {
	_ = gdb.Callback().Create().Before("gorm:create").Register("solo:audit_create", func(tx *gorm.DB) {
		now := time.Now()
		tx.Statement.SetColumn("CreateTime", &now, false)
		tx.Statement.SetColumn("UpdateTime", &now, false)
		if name := auth.CurrentUserName(tx.Statement.Context); name != "" {
			if auditColumnBlank(tx, "CreateBy") {
				tx.Statement.SetColumn("CreateBy", name, false)
			}
			if auditColumnBlank(tx, "UpdateBy") {
				tx.Statement.SetColumn("UpdateBy", name, false)
			}
		}
	})
	_ = gdb.Callback().Update().Before("gorm:update").Register("solo:audit_update", func(tx *gorm.DB) {
		now := time.Now()
		tx.Statement.SetColumn("UpdateTime", &now, false)
		if name := auth.CurrentUserName(tx.Statement.Context); name != "" {
			tx.Statement.SetColumn("UpdateBy", name, false)
		}
	})
}

// auditColumnBlank 检查待写入对象（结构体或结构体切片）该 Go 字段是否全部为零值。
func auditColumnBlank(tx *gorm.DB, goField string) bool {
	if tx.Statement == nil || tx.Statement.Schema == nil {
		return true
	}
	field := tx.Statement.Schema.LookUpField(goField)
	if field == nil {
		return true
	}
	rv := tx.Statement.ReflectValue
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Struct:
		v, isZero := field.ValueOf(tx.Statement.Context, rv)
		return isZero || reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
	case reflect.Slice:
		for i := 0; i < rv.Len(); i++ {
			el := rv.Index(i)
			for el.Kind() == reflect.Ptr {
				el = el.Elem()
			}
			if el.Kind() != reflect.Struct {
				continue
			}
			if v, isZero := field.ValueOf(tx.Statement.Context, el); !isZero &&
				!reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface()) {
				return false
			}
		}
		return true
	default:
		return true
	}
}
