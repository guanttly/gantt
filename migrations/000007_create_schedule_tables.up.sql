-- M06: 排班管道 — 排班计划、分配、变更记录表

-- 排班计划表
CREATE TABLE IF NOT EXISTS schedules (
    id            VARCHAR(64)  NOT NULL PRIMARY KEY,
    org_node_id   VARCHAR(64)  NOT NULL,
    name          VARCHAR(128) NOT NULL,
    start_date    VARCHAR(10)  NOT NULL,
    end_date      VARCHAR(10)  NOT NULL,
    status        VARCHAR(16)  NOT NULL DEFAULT 'draft',
    pipeline_type VARCHAR(32)  NOT NULL DEFAULT 'deterministic',
    config        JSON         DEFAULT NULL,
    created_by    VARCHAR(64)  NOT NULL,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_schedules_org_node (org_node_id),
    INDEX idx_schedules_status   (org_node_id, status),
    INDEX idx_schedules_date     (org_node_id, start_date, end_date)
);

-- 排班分配表
CREATE TABLE IF NOT EXISTS schedule_assignments (
    id            VARCHAR(64)  NOT NULL PRIMARY KEY,
    org_node_id   VARCHAR(64)  NOT NULL,
    schedule_id   VARCHAR(64)  NOT NULL,
    employee_id   VARCHAR(64)  NOT NULL,
    shift_id      VARCHAR(64)  NOT NULL,
    date          VARCHAR(10)  NOT NULL,
    source        VARCHAR(16)  NOT NULL,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_sa_org_node  (org_node_id),
    INDEX idx_sa_schedule  (schedule_id),
    INDEX idx_sa_emp_date  (employee_id, date),
    UNIQUE KEY uk_sa_assignment (schedule_id, employee_id, shift_id, date)
);

-- 排班变更记录表
CREATE TABLE IF NOT EXISTS schedule_changes (
    id              VARCHAR(64)  NOT NULL PRIMARY KEY,
    org_node_id     VARCHAR(64)  NOT NULL,
    schedule_id     VARCHAR(64)  NOT NULL,
    assignment_id   VARCHAR(64)  DEFAULT NULL,
    change_type     VARCHAR(16)  NOT NULL,
    before_data     JSON         DEFAULT NULL,
    after_data      JSON         DEFAULT NULL,
    reason          VARCHAR(256) DEFAULT NULL,
    changed_by      VARCHAR(64)  NOT NULL,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_sc_org_node  (org_node_id),
    INDEX idx_sc_schedule  (schedule_id)
);
