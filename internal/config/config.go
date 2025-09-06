package emsub

import "github.com/spf13/viper"

type Config struct {
	Port       string `mapstructure:"EMSUB_HTTP_PORT"`
	DBHost     string `mapstructure:"EMSUB_DB_HOST"`
	DBPort     string `mapstructure:"EMSUB_DB_PORT"`
	DBUser     string `mapstructure:"EMSUB_DB_USER"`
	DBPassword string `mapstructure:"EMSUB_DB_PASSWORD"`
	DBName     string `mapstructure:"EMSUB_DB_NAME"`
	DBSLL      string `mapstructure:"EMSUB_DB_SLL"`
}

func ConfigLoad() (c *Config, err error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AutomaticEnv()

	v.SetDefault("EMSUB_HTTP_PORT", 8080)
	v.SetDefault("EMSUB_DB_SLL", "disable")

	_ = v.ReadInConfig()

	v.SetConfigName(".env")
	v.SetConfigType("env")
	_ = v.MergeInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
