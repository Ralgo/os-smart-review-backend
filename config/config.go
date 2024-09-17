package config

import (
	"fmt"
	"os"
)

type Config struct {
	Host     string
	Port     string
	DBname   string
	Username string
	Password string
}

func (store Config) Dsn() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		store.Host,
		store.Username,
		store.Password,
		store.DBname,
		store.Port,
	)
}

func New() *Config {
	return &Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		DBname:   os.Getenv("DB_NAME"),
		Username: os.Getenv("DB_USERNAME"),
		Password: os.Getenv("DB_PASSWORD"),
	}
}

func (store Config) ServerPort() string {
	return os.Getenv("SERVER_PORT")
}
