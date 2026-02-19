-- DDL for goctl model generation (MySQL syntax for parser compatibility)
-- Actual SQLite schema is in pkg/db/db.go Migrate()
CREATE TABLE `emails` (
    `id` varchar(36) NOT NULL DEFAULT '',
    `template_slug` varchar(255) NOT NULL DEFAULT '',
    `recipients` text NOT NULL,
    `subject` varchar(500) NOT NULL DEFAULT '',
    `data` text,
    `status` varchar(50) NOT NULL DEFAULT 'pending',
    `priority` int NOT NULL DEFAULT 1,
    `attempts` int NOT NULL DEFAULT 0,
    `max_attempts` int NOT NULL DEFAULT 3,
    `scheduled_at` datetime DEFAULT NULL,
    `sent_at` datetime DEFAULT NULL,
    `message_id` varchar(255) DEFAULT '',
    `error` text,
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
