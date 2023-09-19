package environment

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type EnvConfig struct {
	SqlxConnString string
	AMQPConnString string
}

func ReadEnvConfig() *EnvConfig {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	result := &EnvConfig{}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbSslMode := os.Getenv("DB_SSLMODE")

	result.SqlxConnString = fmt.Sprintf(
		`host=%s port=%s user=%s password=%s dbname=%s sslmode=%s`,
		dbHost, dbPort, dbUser, dbPass, dbName, dbSslMode)

	rmqHost := os.Getenv("RMQ_HOST")
	rmqPort := os.Getenv("RMQ_PORT")
	rmqUser := os.Getenv("RMQ_USER")
	rmqPass := os.Getenv("RMQ_PASS")

	result.AMQPConnString = fmt.Sprintf(
		`amqp://%s:%s@%s:%s/`,
		rmqUser, rmqPass, rmqHost, rmqPort)

	return result
}
