-- M04: 基础数据管理表

CREATE TABLE IF NOT EXISTS employees (
    id            VARCHAR(64) PRIMARY KEY,
    org_node_id   VARCHAR(64) NOT NULL,
    name          VARCHAR(64) NOT NULL,
    employee_no   VARCHAR(32) DEFAULT NULL,
    phone         VARCHAR(20) DEFAULT NULL,
    email         VARCHAR(128) DEFAULT NULL,
    position      VARCHAR(64) DEFAULT NULL,
    category      VARCHAR(32) DEFAULT NULL,
    status        VARCHAR(16) NOT NULL DEFAULT 'active',
    hire_date     VARCHAR(10) DEFAULT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_org_node (org_node_id),
    INDEX idx_status (org_node_id, status),
    UNIQUE KEY uk_org_no (org_node_id, employee_no)
);

CREATE TABLE IF NOT EXISTS shifts (
    id            VARCHAR(64) PRIMARY KEY,
    org_node_id   VARCHAR(64) NOT NULL,
    name          VARCHAR(64) NOT NULL,
    code          VARCHAR(16) NOT NULL,
    start_time    VARCHAR(8) NOT NULL,
    end_time      VARCHAR(8) NOT NULL,
    duration      INT NOT NULL,
    is_cross_day  BOOLEAN NOT NULL DEFAULT FALSE,
    color         VARCHAR(16) DEFAULT '#409EFF',
    priority      INT NOT NULL DEFAULT 0,
    status        VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_org_node (org_node_id),
    UNIQUE KEY uk_org_code (org_node_id, code)
);

CREATE TABLE IF NOT EXISTS shift_dependencies (
    id              VARCHAR(64) PRIMARY KEY,
    org_node_id     VARCHAR(64) NOT NULL,
    shift_id        VARCHAR(64) NOT NULL,
    depends_on_id   VARCHAR(64) NOT NULL,
    dependency_type VARCHAR(16) NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (shift_id) REFERENCES shifts(id) ON DELETE CASCADE,
    FOREIGN KEY (depends_on_id) REFERENCES shifts(id) ON DELETE CASCADE,
    INDEX idx_org_node (org_node_id),
    UNIQUE KEY uk_dep (shift_id, depends_on_id, dependency_type)
);

CREATE TABLE IF NOT EXISTS employee_groups (
    id            VARCHAR(64) PRIMARY KEY,
    org_node_id   VARCHAR(64) NOT NULL,
    name          VARCHAR(64) NOT NULL,
    description   VARCHAR(256) DEFAULT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_org_node (org_node_id)
);

CREATE TABLE IF NOT EXISTS group_members (
    id            VARCHAR(64) PRIMARY KEY,
    group_id      VARCHAR(64) NOT NULL,
    employee_id   VARCHAR(64) NOT NULL,
    org_node_id   VARCHAR(64) NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (group_id) REFERENCES employee_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE,
    INDEX idx_org_node (org_node_id),
    UNIQUE KEY uk_group_emp (group_id, employee_id)
);

CREATE TABLE IF NOT EXISTS leaves (
    id            VARCHAR(64) PRIMARY KEY,
    org_node_id   VARCHAR(64) NOT NULL,
    employee_id   VARCHAR(64) NOT NULL,
    leave_type    VARCHAR(32) NOT NULL,
    start_date    VARCHAR(10) NOT NULL,
    end_date      VARCHAR(10) NOT NULL,
    reason        VARCHAR(256) DEFAULT NULL,
    status        VARCHAR(16) NOT NULL DEFAULT 'pending',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE,
    INDEX idx_org_node (org_node_id),
    INDEX idx_emp_date (employee_id, start_date, end_date)
);
