package flags

import (
	"os"
)

func GetAPIKey() *string {
	if SetApiKeyFlag != "" {
		return &SetApiKeyFlag
	}

	if envVal, ok := os.LookupEnv("HX_API_KEY"); ok {
		return &envVal
	}

	return nil
}
