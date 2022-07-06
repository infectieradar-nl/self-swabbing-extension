package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/coneno/logger"
	"github.com/infectieradar-nl/self-swabbing-extension/pkg/types"
)

const (
	ENV_GIN_DEBUG_MODE = "GIN_DEBUG_MODE"
	ENV_LOG_LEVEL      = "LOG_LEVEL"

	ENV_SELF_SWABBING_EXTENSION_LISTEN_PORT = "SELF_SWABBING_EXT_LISTEN_PORT"
	ENV_CORS_ALLOW_ORIGINS                  = "CORS_ALLOW_ORIGINS"
	ENV_API_KEYS                            = "API_KEYS"
	ENV_ALLOW_ENTRY_CODE_UPLOAD             = "ALLOW_ENTRY_CODE_UPLOAD"

	ENV_SELF_SWABBING_EXT_DB_CONNECTION_STR    = "SELF_SWABBING_EXT_DB_CONNECTION_STR"
	ENV_SELF_SWABBING_EXT_DB_USERNAME          = "SELF_SWABBING_EXT_DB_USERNAME"
	ENV_SELF_SWABBING_EXT_DB_PASSWORD          = "SELF_SWABBING_EXT_DB_PASSWORD"
	ENV_SELF_SWABBING_EXT_DB_CONNECTION_PREFIX = "SELF_SWABBING_EXT_DB_CONNECTION_PREFIX"

	ENV_DB_TIMEOUT           = "DB_TIMEOUT"
	ENV_DB_IDLE_CONN_TIMEOUT = "DB_IDLE_CONN_TIMEOUT"
	ENV_DB_MAX_POOL_SIZE     = "DB_MAX_POOL_SIZE"
	ENV_DB_NAME_PREFIX       = "DB_DB_NAME_PREFIX"
)

// Config is the structure that holds all global configuration data
type Config struct {
	GinDebugMode         bool
	Port                 string
	AllowOrigins         []string
	APIKeys              []string
	AllowEntryCodeUpload bool
	LogLevel             logger.LogLevel
	DBConfig             types.DBConfig
}

func initConfig() Config {
	conf := Config{}
	conf.GinDebugMode = os.Getenv(ENV_GIN_DEBUG_MODE) == "true"
	conf.Port = os.Getenv(ENV_SELF_SWABBING_EXTENSION_LISTEN_PORT)
	conf.AllowOrigins = strings.Split(os.Getenv(ENV_CORS_ALLOW_ORIGINS), ",")
	conf.APIKeys = strings.Split(os.Getenv(ENV_API_KEYS), ",")
	conf.AllowEntryCodeUpload = os.Getenv(ENV_ALLOW_ENTRY_CODE_UPLOAD) == "true"

	conf.LogLevel = getLogLevel()
	conf.DBConfig = getDBConfig()

	return conf
}

func getLogLevel() logger.LogLevel {
	switch os.Getenv(ENV_LOG_LEVEL) {
	case "debug":
		return logger.LEVEL_DEBUG
	case "info":
		return logger.LEVEL_INFO
	case "error":
		return logger.LEVEL_ERROR
	case "warning":
		return logger.LEVEL_WARNING
	default:
		return logger.LEVEL_INFO
	}
}

func getDBConfig() types.DBConfig {
	connStr := os.Getenv(ENV_SELF_SWABBING_EXT_DB_CONNECTION_STR)
	username := os.Getenv(ENV_SELF_SWABBING_EXT_DB_USERNAME)
	password := os.Getenv(ENV_SELF_SWABBING_EXT_DB_PASSWORD)
	prefix := os.Getenv(ENV_SELF_SWABBING_EXT_DB_CONNECTION_PREFIX) // Used in test mode
	if connStr == "" || username == "" || password == "" {
		logger.Error.Fatal("Couldn't read DB credentials.")
	}
	URI := fmt.Sprintf(`mongodb%s://%s:%s@%s`, prefix, username, password, connStr)

	var err error
	Timeout, err := strconv.Atoi(os.Getenv(ENV_DB_TIMEOUT))
	if err != nil {
		logger.Error.Fatal("DB_TIMEOUT: " + err.Error())
	}
	IdleConnTimeout, err := strconv.Atoi(os.Getenv(ENV_DB_IDLE_CONN_TIMEOUT))
	if err != nil {
		logger.Error.Fatal("DB_IDLE_CONN_TIMEOUT" + err.Error())
	}
	mps, err := strconv.Atoi(os.Getenv(ENV_DB_MAX_POOL_SIZE))
	MaxPoolSize := uint64(mps)
	if err != nil {
		logger.Error.Fatal("DB_MAX_POOL_SIZE: " + err.Error())
	}

	DBNamePrefix := os.Getenv(ENV_DB_NAME_PREFIX)

	return types.DBConfig{
		URI:             URI,
		Timeout:         Timeout,
		IdleConnTimeout: IdleConnTimeout,
		MaxPoolSize:     MaxPoolSize,
		DBNamePrefix:    DBNamePrefix,
	}
}
