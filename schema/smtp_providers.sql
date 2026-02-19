-- DDL for goctl model generation (MySQL syntax for parser compatibility)
-- Actual SQLite schema is in pkg/db/db.go Migrate()
CREATE TABLE `smtp_providers` (
    `id` varchar(36) NOT NULL DEFAULT '',
    `name` varchar(255) NOT NULL DEFAULT '',
    `host` varchar(255) NOT NULL DEFAULT '',
    `port` int NOT NULL DEFAULT 587,
    `username` varchar(255) DEFAULT '',
    `password` varchar(255) DEFAULT '',
    `from_email` varchar(255) NOT NULL DEFAULT '',
    `from_name` varchar(255) DEFAULT '',
    `is_default` tinyint NOT NULL DEFAULT 0,
    `rate_limit` int DEFAULT NULL,
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_smtp_providers_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
