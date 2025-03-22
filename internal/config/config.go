package config

import (
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/ruslantos/gophemart-service/internal/logger"
)

type Config struct {
	RunAddress  string
	DatabaseURI string
	AccrualURL  string
}

func Parse() (*Config, error) {
	runAddress := flag.String("a", "", "Адрес и порт запуска сервиса")
	databaseURI := flag.String("d", "", "Адрес подключения к базе данных")
	accrualSystemAddress := flag.String("r", "", "Адрес системы расчёта начислений")

	flag.Parse()

	// todo удалить
	//os.Setenv("DATABASE_URI", "user=videos password=password dbname=shortenerdatabase sslmode=disable")
	//os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://localhost:8080")
	//os.Setenv("RUN_ADDRESS", ":8081")

	config := &Config{
		RunAddress:  *runAddress,
		DatabaseURI: *databaseURI,
		AccrualURL:  *accrualSystemAddress,
	}

	if envRunAddress := os.Getenv("RUN_ADDRESS"); envRunAddress != "" {
		config.RunAddress = envRunAddress
	}
	if envDatabaseURI := os.Getenv("DATABASE_URI"); envDatabaseURI != "" {
		config.DatabaseURI = envDatabaseURI
	}
	if envAccrualSystemAddress := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualSystemAddress != "" {
		config.AccrualURL = envAccrualSystemAddress
	}

	if config.DatabaseURI == "" {
		return nil, fmt.Errorf("DATABASE_URI не задан")
	}
	if config.AccrualURL == "" {
		return nil, fmt.Errorf("ACCRUAL_SYSTEM_ADDRESS не задан")
	}

	logger.Get().Info("init service config:",
		zap.String("RUN_ADDRESS", config.RunAddress),
		zap.String("DATABASE_URI", config.DatabaseURI),
		zap.String("ACCRUAL_SYSTEM_ADDRESS", config.AccrualURL),
	)

	return config, nil
}
