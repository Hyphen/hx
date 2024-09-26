package flags

import (
	"os"
)

func GetAPIKey() *string {
	if ApiKeyFlag != "" {
		return &ApiKeyFlag
	}

	if envVal, ok := os.LookupEnv("HX_API_KEY"); ok {
		return &envVal
	}

	return nil
}
