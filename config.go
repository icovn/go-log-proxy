package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// DotEnvVariable -> get .env
func DotEnvVariable(key string) string {
	// load .env file
	err := godotenv.Load("./.env")
	if err != nil {
		log.Println("(DotEnvVariable) No .env file found. Using system environment variables.")
	}

	return os.Getenv(key)
}

func DotEnvVariableWithDefault(key string, defaultValue string) string {
	var stringVal = DotEnvVariable(key)
	if stringVal == "" {
		return defaultValue
	}
	return stringVal
}

func DotEnvVariableBool(key string, defaultValue bool) bool {
	var stringVal = DotEnvVariable(key)
	if stringVal == "" {
		return defaultValue
	}
	result, err := strconv.ParseBool(stringVal)
	if err != nil {
		return defaultValue
	}
	return result
}

func DotEnvVariableInt(key string, defaultValue int64) int64 {
	var stringVal = DotEnvVariable(key)
	if stringVal == "" {
		return defaultValue
	}
	result, err := strconv.ParseInt(stringVal, 10, 64)
	if err != nil {
		return defaultValue
	}
	return result
}
