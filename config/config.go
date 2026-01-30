package config

import (
	"encoding/json"
	"log"
	"os"

	"github.com/joho/godotenv"
)

const (
	defoltCfgPath = "config/config.json"
)

type Config struct {
	Env           string  `json:"env"`
	Storage       Storage `json:"storage"`
	CacheCap      int     `json:"cache_cap"`
	ConsmerNumber int     `json:"consumer_number"`
}

type Storage struct {
	Host       string `json:"db_host"`
	Port       string `json:"db_port"`
	DBUser     string
	DBName     string
	DBPassword string
	ServerPort string
}

func MustLoad() *Config {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("can`t load config, check configuration or .env files", err)
	}

	cfgPath, ok := os.LookupEnv("CONFIG_PATH")
	if !ok {
		cfgPath = defoltCfgPath
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		log.Fatalf("config file by way %s doesn`t exist", cfgPath)
	} else if err != nil {
		log.Fatalf("error %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("failed to parse config JSON: %v", err)
	}

	if user, ok := os.LookupEnv("DB_USER"); ok {
		cfg.Storage.DBUser = user
	} else {
		log.Fatal("DB_USER environment variable is not set")
	}

	if password, ok := os.LookupEnv("DB_PASSWORD"); ok {
		cfg.Storage.DBPassword = password
	} else {
		log.Fatal("DB_PASSWORD environment variable is not set")
	}

	if dbName, ok := os.LookupEnv("DB_NAME"); ok {
		cfg.Storage.DBName = dbName
	} else {
		log.Fatal("DB_NAME environment variable is not set")
	}

	return &cfg
}

// структура данных config
