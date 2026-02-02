package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Node struct {
		ID         string `mapstructure:"id"`
		Role       string `mapstructure:"role"`
		ListenAddr string `mapstructure:"listen_addr"`
		RPCEndpoint string `mapstructure:"rpc_endpoint"`
	} `mapstructure:"node"`

	Blockchain struct {
		DataDir       string `mapstructure:"data_dir"`
		MaxBlockSize  int    `mapstructure:"max_block_size"`
		BlockTime     string `mapstructure:"block_time"`
		Consensus     string `mapstructure:"consensus"`
	} `mapstructure:"blockchain"`

	Wallet struct {
		PrivateKeyPath string `mapstructure:"private_key_path"`
		ContractAddr   string `mapstructure:"contract_address"`
		NetworkURL     string `mapstructure:"network_url"`
	} `mapstructure:"wallet"`

	Cache struct {
		RedisURL      string `mapstructure:"redis_url"`
		TTL           string `mapstructure:"ttl"`
		MaxConnections int    `mapstructure:"max_connections"`
	} `mapstructure:"cache"`

	Indexing struct {
		Semantic struct {
			Enabled   bool `mapstructure:"enabled"`
			Dimension int  `mapstructure:"dimension"`
			M         int  `mapstructure:"hnsw_m"`
			EFConstruction int `mapstructure:"hnsw_ef_construction"`
		} `mapstructure:"semantic"`
		Temporal struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"temporal"`
		Category struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"category"`
		FullText struct {
			Enabled bool `mapstructure:"enabled"`
		} `mapstructure:"fulltext"`
	} `mapstructure:"indexing"`

	Security struct {
		TLS struct {
			Enabled   bool   `mapstructure:"enabled"`
			CertFile  string `mapstructure:"cert_file"`
			KeyFile   string `mapstructure:"key_file"`
		} `mapstructure:"tls"`
		JWT struct {
			Secret        string `mapstructure:"secret"`
			TokenDuration string `mapstructure:"token_duration"`
		} `mapstructure:"jwt"`
		Encryption struct {
			Enabled         bool   `mapstructure:"enabled"`
			KeyRotationDays int    `mapstructure:"key_rotation_days"`
		} `mapstructure:"encryption"`
	} `mapstructure:"security"`

	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
		Output string `mapstructure:"output"`
	} `mapstructure:"logging"`

	Monitoring struct {
		Prometheus struct {
			Enabled bool   `mapstructure:"enabled"`
			Path    string `mapstructure:"path"`
		} `mapstructure:"prometheus"`
		Tracing struct {
			Enabled       bool   `mapstructure:"enabled"`
			JaegerEndpoint string `mapstructure:"jaeger_endpoint"`
		} `mapstructure:"tracing"`
	} `mapstructure:"monitoring"`

	Performance struct {
		MaxGoroutines    int    `mapstructure:"max_goroutines"`
		RequestTimeout   string `mapstructure:"request_timeout"`
		WriteBufferSize  int    `mapstructure:"write_buffer_size"`
		ReadBufferSize   int    `mapstructure:"read_buffer_size"`
	} `mapstructure:"performance"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}