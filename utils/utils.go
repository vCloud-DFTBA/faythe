package utils

import "os"

// Getenv returns default value if environment variable
// doesn't exist.
func Getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
