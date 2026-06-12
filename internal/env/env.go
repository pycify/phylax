package env

import "github.com/joho/godotenv"

func LoadFile(path string) error {
	return godotenv.Load(path)
}
