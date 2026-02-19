-- DDL for goctl model generation (MySQL syntax for parser compatibility)
-- Actual SQLite schema is in pkg/db/db.go Migrate()
CREATE TABLE `email_events` (
    `id` varchar(36) NOT NULL DEFAULT '',
    `email_id` varchar(36) NOT NULL DEFAULT '',
    `event_type` varchar(100) NOT NULL DEFAULT '',
    `timestamp` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `details` text,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
