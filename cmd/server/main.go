package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"metrobase/internal/config"
	"metrobase/internal/cronjob"
	"metrobase/internal/db"
	"metrobase/internal/router"
)

func main() {
	schema := flag.String("schema", "doc/schema/cali.sql", "建表 SQL（表不存在时初始化用）")
	flag.Parse()

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("读取配置失败：%v", err)
	}

	secret := cfg.JWT.Secret
	if secret == "" || secret == "CHANGE_ME" {
		// 不硬编码口令：未注入 APP_JWT_SECRET 时每次启动生成一次性密钥
		buf := make([]byte, 16)
		_, _ = rand.Read(buf)
		secret = hex.EncodeToString(buf)
		log.Println("警告：未配置 APP_JWT_SECRET，已生成一次性密钥（重启后旧 token 失效）")
	}

	gdb, err := db.Open(cfg.Database.DSN, cfg.Database.EnableReplica)
	if err != nil {
		log.Fatalf("打开数据库失败：%v", err)
	}
	// 表结构随建表脚本管理：仅在库全新时初始化一次，业务代码不自动建表。
	if !db.HasTable(gdb, "t_sys_user") {
		if err := db.ApplySQLFile(gdb, *schema); err != nil {
			log.Fatalf("初始化数据库失败：%v", err)
		}
		log.Printf("已按 %s 初始化数据库", *schema)
	}

	// 定时任务从 t_sys_job 恢复（进程重启不丢任务配置）
	scheduler := cronjob.NewScheduler(gdb)
	scheduler.RegisterFunc("calibration_due_notice", func() {}) // 后续迭代实现到期提醒
	if err := scheduler.Start(); err != nil {
		log.Fatalf("定时任务启动失败：%v", err)
	}

	engine := router.NewEngine(gdb, secret, cfg.JWT.ExpireHours)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("solo 服务已启动：%s（本地 SQLite %s）", addr, cfg.Database.DSN)
	if err := engine.Run(addr); err != nil {
		log.Fatal(err)
	}
}
