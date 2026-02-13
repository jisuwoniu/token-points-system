package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig   `mapstructure:"database"`
	Server   ServerConfig     `mapstructure:"server"`
	Chains   []ChainConfig    `mapstructure:"chains"`
	Points   PointsConfig     `mapstructure:"points"`
	Backup   BackupConfig     `mapstructure:"backup"`
	Logging  LoggingConfig    `mapstructure:"logging"`
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.DBName)
}

type ServerConfig struct {
	Port         int `mapstructure:"port"`
	ReadTimeout  int `mapstructure:"read_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`
}

type ChainConfig struct {
	ID                string `mapstructure:"id"`
	Name              string `mapstructure:"name"`
	RPCURL            string `mapstructure:"rpc_url"`
	WSURL             string `mapstructure:"ws_url"`
	ChainID           uint64 `mapstructure:"chain_id"`
	ContractAddress   string `mapstructure:"contract_address"`
	StartBlock        int64  `mapstructure:"start_block"`
	ConfirmationBlocks int   `mapstructure:"confirmation_blocks"`
	PullInterval      int    `mapstructure:"pull_interval"`
	Enabled           bool   `mapstructure:"enabled"`
	
	WorkerPoolSize    int `mapstructure:"worker_pool_size"`
	QueueSize         int `mapstructure:"queue_size"`
	BatchSize         int `mapstructure:"batch_size"`
	MaxRetries        int `mapstructure:"max_retries"`
	AdaptiveMode      bool `mapstructure:"adaptive_mode"`
}

type PointsConfig struct {
	CalculationRate     float64 `mapstructure:"calculation_rate"`
	CalculationInterval int     `mapstructure:"calculation_interval"`
	CalculationCron     string  `mapstructure:"calculation_cron"`
}

type BackupConfig struct {
	Enabled      bool `mapstructure:"enabled"`
	Interval     int  `mapstructure:"interval"`
	RetentionDays int `mapstructure:"retention_days"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()
	
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

func (c *Config) GetChainConfig(chainID string) (*ChainConfig, error) {
	for _, chain := range c.Chains {
		if chain.ID == chainID {
			return &chain, nil
		}
	}
	return nil, fmt.Errorf("chain config not found: %s", chainID)
}

func (c *Config) GetEnabledChains() []ChainConfig {
	var enabled []ChainConfig
	for _, chain := range c.Chains {
		if chain.Enabled {
			enabled = append(enabled, chain)
		}
	}
	return enabled
}
