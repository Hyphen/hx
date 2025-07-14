package toggle

import (
	"context"
	"os"
	"strings"

	"github.com/Hyphen/cli/internal/models"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/hyphen/openfeature-provider-go/pkg/toggle"
	"github.com/open-feature/go-sdk/openfeature"
)

var client *openfeature.Client

func getEnvironment() string {
	if flags.DevFlag || strings.ToLower(os.Getenv("HYPHEN_DEV")) == "true" {
		return "development"
	}
	return "production"
}

func init() {
	provider, err := toggle.NewProvider(toggle.Config{
		PublicKey:   "public_b3JnXzY2ODU3NWMwZTE2OWNkZTk3NGE1Yzc2YTpwcm9qXzY3MGQ1M2Q4M2ViZDdiZmJkN2YxZjUwMTplb3hJczd2RUlVT3FmRzdwUXJmMg==",
		Application: "hx",
		Environment: getEnvironment(),
	})
	if err != nil {
		// do something here...
	}

	// Set as global provider
	openfeature.SetProviderAndWait(provider)

	// Create a client
	client = openfeature.NewClient("hx")

	// Create evaluation context
	ctx := openfeature.NewEvaluationContext(
		"anonymous",
		map[string]interface{}{},
	)

	client.SetEvaluationContext(ctx)
}

func HandleAuth(ec models.ExecutionContext) {
	// TODO: handle when there is no member...
	targetingKey := ec.Member.ID
	if targetingKey == "" {
		targetingKey = ec.User.ID
	}
	if targetingKey == "" {
		targetingKey = "anonymous"
	}

	user := map[string]interface{}{
		"id":    ec.Member.ID,
		"name":  ec.Member.Name,
		"email": ec.Member.Email,
		"customAttributes": map[string]interface{}{
			"organization": map[string]interface{}{
				"id":   ec.Member.Organization.ID,
				"name": ec.Member.Organization.Name,
			},
		},
	}

	ctx := openfeature.NewEvaluationContext(
		targetingKey,
		map[string]interface{}{
			"user": user,
		},
	)
	client.SetEvaluationContext(ctx)
}

func GetBooleanValue(key string, defaultValue bool) bool {
	if client == nil {
		return defaultValue
	}

	value, err := client.BooleanValue(context.Background(), key, defaultValue, client.EvaluationContext())
	if err != nil {
		return defaultValue
	}

	return value
}

func GetStringValue(key string, defaultValue string) string {
	if client == nil {
		return defaultValue
	}

	value, err := client.StringValue(context.Background(), key, defaultValue, client.EvaluationContext())
	if err != nil {
		return defaultValue
	}

	return value
}
