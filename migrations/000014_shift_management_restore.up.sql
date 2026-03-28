ALTER TABLE shifts
    ADD COLUMN type VARCHAR(32) NOT NULL DEFAULT 'regular' COMMENT '班次类型',
    ADD COLUMN description TEXT DEFAULT NULL COMMENT '班次描述',
    ADD COLUMN metadata JSON DEFAULT NULL COMMENT '扩展信息';

CREATE TABLE IF NOT EXISTS shift_groups (
    id            VARCHAR(64) PRIMARY KEY,
    org_node_id   VARCHAR(64) NOT NULL,
    shift_id      VARCHAR(64) NOT NULL,
    group_id      VARCHAR(64) NOT NULL,
    priority      INT NOT NULL DEFAULT 0,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    notes         TEXT DEFAULT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_org_node (org_node_id),
    INDEX idx_shift_groups_shift (shift_id),
    INDEX idx_shift_groups_group (group_id),
    UNIQUE KEY uk_shift_group (shift_id, group_id),
    FOREIGN KEY (shift_id) REFERENCES shifts(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES employee_groups(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS fixed_assignments (
    id             VARCHAR(64) PRIMARY KEY,
    org_node_id    VARCHAR(64) NOT NULL,
    shift_id       VARCHAR(64) NOT NULL,
    employee_id    VARCHAR(64) NOT NULL,
    pattern_type   VARCHAR(16) NOT NULL,
    weekdays       JSON DEFAULT NULL,
    week_pattern   VARCHAR(16) DEFAULT NULL,
    monthdays      JSON DEFAULT NULL,
    specific_dates JSON DEFAULT NULL,
    start_date     VARCHAR(10) DEFAULT NULL,
    end_date       VARCHAR(10) DEFAULT NULL,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_org_node (org_node_id),
    INDEX idx_fixed_assignments_shift (shift_id),
    INDEX idx_fixed_assignments_employee (employee_id),
    FOREIGN KEY (shift_id) REFERENCES shifts(id) ON DELETE CASCADE,
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS shift_weekly_staff (
    id            VARCHAR(64) PRIMARY KEY,
    org_node_id   VARCHAR(64) NOT NULL,
    shift_id      VARCHAR(64) NOT NULL,
    weekday       INT NOT NULL,
    staff_count   INT NOT NULL DEFAULT 0,
    is_custom     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_org_node (org_node_id),
    INDEX idx_shift_weekly_staff_shift (shift_id),
    UNIQUE KEY uk_shift_weekday (shift_id, weekday),
    FOREIGN KEY (shift_id) REFERENCES shifts(id) ON DELETE CASCADE
);