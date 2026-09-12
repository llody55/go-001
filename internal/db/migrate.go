package db

import (
	"bufio"
	"os"
	"strings"

	"gorm.io/gorm"
)

// ApplySQLFile 执行 schema SQL 文件。SQLite 驱动的 Exec 不支持多语句，
// 先逐行剔除 -- 注释，再按分号切分为独立语句逐条执行（脚本内不含存储过程）。
func ApplySQLFile(gdb *gorm.DB, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var clean strings.Builder
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		clean.WriteString(line)
		clean.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	for _, stmt := range strings.Split(clean.String(), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := gdb.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

// HasTable 判断表是否已建。
func HasTable(gdb *gorm.DB, name string) bool {
	var n int
	gdb.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name = ?", name).Scan(&n)
	return n > 0
}
