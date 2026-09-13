-- go-001 cali 计量器具周期校准 · 建表与种子数据（SQLite 方言）
-- 服务首次启动时对空库执行一次；表结构以本文件为准，业务代码不自动建表或改表。
-- 列名一经发布保持稳定，外部集成按列名对接，不依赖 Go 结构体命名。

CREATE TABLE IF NOT EXISTS t_sys_user (
    id          INTEGER PRIMARY KEY,
    username    TEXT NOT NULL,
    password    TEXT NOT NULL,
    real_name   TEXT,
    status      INTEGER NOT NULL DEFAULT 0,
    create_by   TEXT,
    create_time DATETIME,
    update_by   TEXT,
    update_time DATETIME,
    del_flag    INTEGER
);

CREATE TABLE IF NOT EXISTS t_sys_job (
    id          INTEGER PRIMARY KEY,
    job_name    TEXT NOT NULL,
    func_key    TEXT NOT NULL,
    cron_expr   TEXT NOT NULL,
    status      INTEGER NOT NULL DEFAULT 0,
    create_by   TEXT,
    create_time DATETIME,
    update_by   TEXT,
    update_time DATETIME,
    del_flag    INTEGER
);

CREATE TABLE IF NOT EXISTS t_cali_device (
    id              INTEGER PRIMARY KEY,
    device_no       TEXT NOT NULL,
    device_name     TEXT NOT NULL,
    model_spec      TEXT,
    full_scale      REAL NOT NULL,          -- 量程上限（MPa），引用误差的分母
    accuracy_class  REAL NOT NULL,          -- 准确度等级（允许引用误差，百分数）
    use_dept        TEXT,
    keeper          TEXT,
    status          INTEGER NOT NULL DEFAULT 0,  -- 0=正常 1=停用
    last_calib_date TEXT,                   -- 上次校准日期 yyyy-MM-dd
    cycle_months    INTEGER NOT NULL DEFAULT 12, -- 校准周期（月）
    next_calib_date TEXT,                   -- 下次应校准日期 yyyy-MM-dd
    create_by       TEXT,
    create_time     DATETIME,
    update_by       TEXT,
    update_time     DATETIME,
    del_flag        INTEGER
);

CREATE TABLE IF NOT EXISTS t_cali_record (
    id              INTEGER PRIMARY KEY,
    record_no       TEXT NOT NULL,
    device_id       INTEGER NOT NULL,
    device_no       TEXT,                   -- 冗余档案字段，以器具档案为准
    device_name     TEXT,
    calib_date      TEXT NOT NULL,          -- 校准日期 yyyy-MM-dd
    standard_value  REAL NOT NULL,          -- 标准值
    indicated_value REAL NOT NULL,          -- 示值
    reference_error REAL,                   -- 最大引用误差（百分数，两位小数）
    result          INTEGER,                -- 0=合格 1=不合格
    cert_no         TEXT,                   -- 证书编号，发证后回填
    status          INTEGER NOT NULL DEFAULT 0, -- 0=已登记 1=已发证 2=作废
    remark          TEXT,
    create_by       TEXT,
    create_time     DATETIME,
    update_by       TEXT,
    update_time     DATETIME,
    del_flag        INTEGER
);

CREATE TABLE IF NOT EXISTS t_cali_plan (
    id           INTEGER PRIMARY KEY,
    plan_no      TEXT NOT NULL,             -- 计划编号，如 RP-2026-001
    year         INTEGER NOT NULL,          -- 计划年度
    device_id    INTEGER NOT NULL,
    device_no    TEXT,                      -- 冗余档案字段，以器具档案为准
    device_name  TEXT,
    plan_date    TEXT NOT NULL,             -- 计划应校准日期 yyyy-MM-dd
    status       INTEGER NOT NULL DEFAULT 0, -- 0=待安排 1=已完成
    record_id    INTEGER,                   -- 完工关联的校准记录 id
    notice_time  DATETIME,                  -- 最近一次到期提醒时间
    create_by    TEXT,
    create_time  DATETIME,
    update_by    TEXT,
    update_time  DATETIME,
    del_flag     INTEGER
);

-- 种子：登录账号 cali01 / 123456
INSERT INTO t_sys_user (id, username, password, real_name, status, create_time, update_time)
SELECT 1, 'cali01', 'e10adc3949ba59abbe56e057f20f883e', '计量校准员', 0,
       '2026-09-12 09:00:00', '2026-09-12 09:00:00'
WHERE NOT EXISTS (SELECT 1 FROM t_sys_user WHERE id = 1);

-- 种子：4 台器具档案，id=4 为停用；last_calib_date + cycle_months 是年度计划排期依据
INSERT INTO t_cali_device
(id, device_no, device_name, model_spec, full_scale, accuracy_class, use_dept, keeper, status,
 last_calib_date, cycle_months, next_calib_date, create_time, update_time)
SELECT 1, 'JL-Y-001', '普通压力表', 'Y-100', 10.0, 1.6, '动力车间', '王保国', 0,
       '2025-09-20', 12, '2026-09-20', '2026-09-12 09:00:00', '2026-09-12 09:00:00'
WHERE NOT EXISTS (SELECT 1 FROM t_cali_device WHERE id = 1);
INSERT INTO t_cali_device
(id, device_no, device_name, model_spec, full_scale, accuracy_class, use_dept, keeper, status,
 last_calib_date, cycle_months, next_calib_date, create_time, update_time)
SELECT 2, 'JL-Y-002', '精密压力表', 'YB-150', 25.0, 0.4, '质检中心', '李检', 0,
       '2025-09-10', 6, '2026-03-10', '2026-09-12 09:00:00', '2026-09-12 09:00:00'
WHERE NOT EXISTS (SELECT 1 FROM t_cali_device WHERE id = 2);
INSERT INTO t_cali_device
(id, device_no, device_name, model_spec, full_scale, accuracy_class, use_dept, keeper, status,
 last_calib_date, cycle_months, next_calib_date, create_time, update_time)
SELECT 3, 'JL-Y-003', '压力变送器', '3051', 1.6, 0.25, '合成车间', '赵仪', 0,
       '2025-01-01', 12, '2026-01-01', '2026-09-12 09:00:00', '2026-09-12 09:00:00'
WHERE NOT EXISTS (SELECT 1 FROM t_cali_device WHERE id = 3);
INSERT INTO t_cali_device
(id, device_no, device_name, model_spec, full_scale, accuracy_class, use_dept, keeper, status,
 last_calib_date, cycle_months, next_calib_date, create_time, update_time)
SELECT 4, 'JL-Y-004', '旧型压力表', 'Y-60', 1.6, 2.5, '检修班', '孙修', 1,
       '2025-02-01', 12, '2026-02-01', '2026-09-12 09:00:00', '2026-09-12 09:00:00'
WHERE NOT EXISTS (SELECT 1 FROM t_cali_device WHERE id = 4);
