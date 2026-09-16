package apiconf

import (
	"fmt"
	"testing"

	"github.com/Hyphen/cli/pkg/flags"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentRouting(t *testing.T) {
	for _, local := range []bool{false, true} {
		for _, localApix := range []bool{false, true} {
			for _, localHorizon := range []bool{false, true} {
				for _, dev := range []bool{false, true} {
					for _, devFlag := range []bool{false, true} {
						name := fmt.Sprintf("local=%t/apix=%t/horizon=%t/dev=%t/flag=%t", local, localApix, localHorizon, dev, devFlag)
						t.Run(name, func(t *testing.T) {
							t.Setenv("HYPHEN_LOCAL", fmt.Sprint(local))
							t.Setenv("HYPHEN_LOCAL_APIX", fmt.Sprint(localApix))
							t.Setenv("HYPHEN_LOCAL_HORIZON", fmt.Sprint(localHorizon))
							t.Setenv("HYPHEN_DEV", fmt.Sprint(dev))
							previous := flags.DevFlag
							flags.DevFlag = devFlag
							t.Cleanup(func() { flags.DevFlag = previous })

							api, app, horizon := "https://api.hyphen.ai", "https://app.hyphen.ai", "https://toggle.hyphen.cloud"
							auth, clientID := "https://auth.hyphen.ai", "e6315ab1-5847-4c75-a003-65b5ed374dd1"
							if dev || devFlag {
								api, app, horizon = "https://dev-api.hyphen.ai", "https://dev-app.hyphen.ai", "https://dev-horizon.hyphen.ai"
							}
							ioURL := api
							if local || localApix {
								api, app = "http://localhost:4000", "http://localhost:3000"
								ioURL = "http://localhost:4100"
							}
							if local || localHorizon {
								horizon = "http://localhost:3333"
							}
							if dev || devFlag || local || localApix {
								auth, clientID = "https://dev-auth.hyphen.ai", "8d5fb36d-2886-4c53-ab70-e6203e781fbc"
							}
							assert.Equal(t, api, GetBaseApixUrl())
							assert.Equal(t, ioURL, GetIOBaseUrl())
							assert.Equal(t, auth, GetBaseAuthUrl())
							assert.Equal(t, clientID, GetAuthClientID())
							assert.Equal(t, app, GetBaseAppUrl())
							assert.Equal(t, horizon, GetBaseHorizonUrl())
						})
					}
				}
			}
		}
	}
}
