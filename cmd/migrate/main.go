// initdb：create/部署阶段执行的建表与种子初始化（go run ./cmd/migrate）。
// process 阶段不执行本命令——模型开跑时表已存在。
package main

import (
	"flag"
	"log"

	"metrobase/internal/config"
	"metrobase/internal/db"
)

func main() {
	schema := flag.String("schema", "doc/schema/cali.sql", "建表 SQL 路径")
	flag.Parse()

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("读取配置失败：%v", err)
	}
	gdb, err := db.Open(cfg.Database.DSN, false)
	if err != nil {
		log.Fatalf("打开数据库失败：%v", err)
	}
	if err := db.ApplySQLFile(gdb, *schema); err != nil {
		log.Fatalf("执行建表脚本失败：%v", err)
	}
	log.Printf("initdb 完成：%s（%s）", cfg.Database.DSN, *schema)
}
