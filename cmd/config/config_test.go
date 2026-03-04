package configcmd

import "testing"

func TestParseAutoUpdateEnabledArg(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
		wantErr  bool
	}{
		{name: "on", input: "on", expected: true},
		{name: "enable", input: "enable", expected: true},
		{name: "true", input: "true", expected: true},
		{name: "off", input: "off", expected: false},
		{name: "disable", input: "disable", expected: false},
		{name: "false", input: "false", expected: false},
		{name: "invalid", input: "maybe", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, err := parseAutoUpdateEnabledArg(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}

			if value != tc.expected {
				t.Fatalf("expected %v for input %q, got %v", tc.expected, tc.input, value)
			}
		})
	}
}
