package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func LoadFile(path string) error {
	return godotenv.Load(path)
}

func ValidateRequired() error {
	required := []string{
		"PHYLAX_ENV",
		"PHYLAX_TOKEN_SECRET",
		"PHYLAX_CREDENTIAL_ENCRYPTION_KEY",
		"DATABASE_URL",
	}

	var missing []string
	for _, key := range required {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			missing = append(missing, key)
		}
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("MAIL_DRIVER")), "smtp") {
		for _, key := range []string{"SMTP_HOST", "SMTP_PORT", "MAIL_FROM_EMAIL"} {
			if strings.TrimSpace(os.Getenv(key)) == "" {
				missing = append(missing, key)
			}
		}
		if value := strings.TrimSpace(os.Getenv("SMTP_PORT")); value != "" {
			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("SMTP_PORT must be a number")
			}
		}
	}

	if len(missing) == 0 {
		for _, key := range []string{"PHYLAX_TOKEN_TTL_SECONDS", "PHYLAX_CHALLENGE_TTL_SECONDS"} {
			if value := strings.TrimSpace(os.Getenv(key)); value != "" {
				if _, err := strconv.Atoi(value); err != nil {
					return fmt.Errorf("%s must be a number", key)
				}
			}
		}
		return nil
	}

	return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
}
