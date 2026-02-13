-- Token Points System Database Schema
-- MySQL 8.0.36
-- No stored procedures - all logic in application layer

CREATE DATABASE IF NOT EXISTS token_points_system
CHARACTER SET utf8mb4
COLLATE utf8mb4_unicode_ci;

USE token_points_system;

-- User total balance table
CREATE TABLE user_balances (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    chain_id VARCHAR(50) NOT NULL COMMENT 'Chain identifier (sepolia, base-sepolia)',
    user_address VARCHAR(42) NOT NULL COMMENT 'User wallet address',
    balance DECIMAL(65,0) NOT NULL DEFAULT 0 COMMENT 'Current token balance',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_chain_user (chain_id, user_address),
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB COMMENT='User total balance table';

-- User total points table
CREATE TABLE user_points (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    chain_id VARCHAR(50) NOT NULL,
    user_address VARCHAR(42) NOT NULL,
    total_points DECIMAL(65,18) NOT NULL DEFAULT 0 COMMENT 'Accumulated total points',
    last_calculated_at TIMESTAMP NULL COMMENT 'Last calculation timestamp',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_chain_user (chain_id, user_address),
    INDEX idx_last_calculated (last_calculated_at)
) ENGINE=InnoDB COMMENT='User total points table';

-- Balance change history table
CREATE TABLE balance_history (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    chain_id VARCHAR(50) NOT NULL,
    user_address VARCHAR(42) NOT NULL,
    balance_before DECIMAL(65,0) NOT NULL COMMENT 'Balance before change',
    balance_after DECIMAL(65,0) NOT NULL COMMENT 'Balance after change',
    change_amount DECIMAL(65,0) NOT NULL COMMENT 'Change amount (positive/negative)',
    change_type ENUM('transfer', 'mint', 'burn') NOT NULL COMMENT 'Change type',
    tx_hash VARCHAR(66) NOT NULL COMMENT 'Transaction hash',
    block_number BIGINT NOT NULL COMMENT 'Block number',
    timestamp TIMESTAMP NOT NULL COMMENT 'Block timestamp',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_chain_user_time (chain_id, user_address, timestamp),
    INDEX idx_tx_hash (tx_hash),
    INDEX idx_block_number (chain_id, block_number)
) ENGINE=InnoDB COMMENT='Balance change history table';

-- Processed blocks table
CREATE TABLE processed_blocks (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    chain_id VARCHAR(50) NOT NULL,
    block_number BIGINT NOT NULL,
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_chain_block (chain_id, block_number),
    INDEX idx_processed_at (processed_at)
) ENGINE=InnoDB COMMENT='Processed blocks tracking table';

-- Point calculation records table (for idempotency)
CREATE TABLE point_calculations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    chain_id VARCHAR(50) NOT NULL,
    user_address VARCHAR(42) NOT NULL,
    period_start TIMESTAMP NOT NULL COMMENT 'Calculation period start',
    period_end TIMESTAMP NOT NULL COMMENT 'Calculation period end',
    points_earned DECIMAL(65,18) NOT NULL COMMENT 'Points earned in this period',
    calculation_hash VARCHAR(64) NOT NULL COMMENT 'SHA256 hash for idempotency',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_calculation_hash (calculation_hash),
    INDEX idx_chain_user_period (chain_id, user_address, period_start, period_end)
) ENGINE=InnoDB COMMENT='Point calculation records table';

-- Calculation backup table (for recovery)
CREATE TABLE calculation_backups (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    chain_id VARCHAR(50) NOT NULL,
    backup_type ENUM('balance_snapshot', 'points_snapshot', 'calculation_state') NOT NULL,
    backup_data JSON NOT NULL COMMENT 'Backup data in JSON format',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_chain_type_time (chain_id, backup_type, created_at)
) ENGINE=InnoDB COMMENT='Calculation backup table for recovery';

-- System configuration table
CREATE TABLE system_config (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value TEXT NOT NULL,
    description VARCHAR(255),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB COMMENT='System configuration table';

-- Insert default configuration
INSERT INTO system_config (config_key, config_value, description) VALUES
('points_rate', '0.05', 'Points calculation rate'),
('confirmation_blocks', '6', 'Number of confirmation blocks required'),
('calculation_interval', '3600', 'Points calculation interval in seconds'),
('pull_interval', '10', 'Block data pull interval in seconds');
