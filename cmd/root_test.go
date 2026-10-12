package cmd

import (
	"bytes"
	"testing"

	"github.com/Hyphen/cli/cmd/link"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkCommandGate(t *testing.T) {
	for _, test := range []struct {
		name    string
		link    interface{}
		netInfo bool
		enabled bool
	}{
		{"both disabled", false, false, false},
		{"only net.info enabled", false, true, false},
		{"only Link enabled", true, false, true},
		{"both enabled", true, true, true},
		{"Link flag missing", nil, true, false},
		{"Link evaluation error", "invalid boolean", true, false},
	} {
		for _, args := range [][]string{
			{"--help"},
			{"link", "example.com", "--qr"},
			{"link", "--help"},
			{"help", "link"},
		} {
			t.Run(test.name+"/"+args[0]+"/"+args[len(args)-1], func(t *testing.T) {
				flags := map[string]memprovider.InMemoryFlag{
					"canUseNetInfo": {State: memprovider.Enabled, DefaultVariant: "value", Variants: map[string]interface{}{"value": test.netInfo}},
				}
				if test.link != nil {
					flags["canUseLink"] = memprovider.InMemoryFlag{State: memprovider.Enabled, DefaultVariant: "value", Variants: map[string]interface{}{"value": test.link}}
				}
				require.NoError(t, openfeature.SetProviderAndWait(memprovider.NewInMemoryProvider(flags)))
				t.Cleanup(func() { require.NoError(t, openfeature.SetProviderAndWait(openfeature.NoopProvider{})) })

				output := new(bytes.Buffer)
				previousOut, previousErr := rootCmd.OutOrStdout(), rootCmd.ErrOrStderr()
				previousPreRun := rootCmd.PersistentPreRunE
				previousLinkPreRun, previousLinkRun := link.LinkCmd.PersistentPreRunE, link.LinkCmd.RunE
				authCalls, serviceCalls := 0, 0
				rootCmd.PersistentPreRunE = func(*cobra.Command, []string) error { return nil }
				link.LinkCmd.PersistentPreRunE = func(*cobra.Command, []string) error { authCalls++; return nil }
				link.LinkCmd.RunE = func(*cobra.Command, []string) error { serviceCalls++; return nil }
				rootCmd.SetOut(output)
				rootCmd.SetErr(output)
				rootCmd.SetArgs(args)
				t.Cleanup(func() {
					rootCmd.RemoveCommand(link.LinkCmd)
					rootCmd.PersistentPreRunE = previousPreRun
					link.LinkCmd.PersistentPreRunE, link.LinkCmd.RunE = previousLinkPreRun, previousLinkRun
					rootCmd.SetOut(previousOut)
					rootCmd.SetErr(previousErr)
					rootCmd.SetArgs(nil)
					for _, cmd := range []*cobra.Command{rootCmd, link.LinkCmd} {
						if cmd.Flags().Lookup("help") != nil {
							require.NoError(t, cmd.Flags().Set("help", "false"))
						}
					}
					require.NoError(t, link.LinkCmd.Flags().Set("qr", "false"))
				})

				err := execute()
				if args[0] == "--help" {
					require.NoError(t, err)
					assert.Equal(t, test.enabled, bytes.Contains(output.Bytes(), []byte("  link ")))
				} else if !test.enabled {
					if args[0] == "help" {
						assert.Contains(t, output.String(), "Unknown help topic")
					} else {
						require.ErrorContains(t, err, `unknown command "link"`)
					}
					assert.NotContains(t, output.String(), "Generate a QR code")
				} else {
					require.NoError(t, err)
				}

				if test.enabled && args[0] == "link" && args[1] == "example.com" {
					assert.Equal(t, 1, authCalls)
					assert.Equal(t, 1, serviceCalls)
					assert.Equal(t, "true", link.LinkCmd.Flags().Lookup("qr").Value.String())
				} else {
					assert.Zero(t, authCalls)
					assert.Zero(t, serviceCalls)
				}
			})
		}
	}
}
