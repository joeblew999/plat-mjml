-- DDL for goctl model generation (MySQL syntax for parser compatibility)
-- Actual SQLite schema is in pkg/db/db.go Migrate()
CREATE TABLE `templates` (
    `id` varchar(36) NOT NULL DEFAULT '',
    `slug` varchar(255) NOT NULL DEFAULT '',
    `name` varchar(255) NOT NULL DEFAULT '',
    `content` text NOT NULL,
    `version` int NOT NULL DEFAULT 1,
    `status` varchar(50) NOT NULL DEFAULT 'draft',
    `category` varchar(255) DEFAULT '',
    `metadata` text,
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `published_at` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_templates_slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
