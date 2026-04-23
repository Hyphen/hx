package cprint

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// captureStdout redirects os.Stdout while f runs and returns what was
// written. The CPrinter methods write to os.Stdout via fmt.Printf, so this
// is how we verify their actual output behavior.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	// Restore os.Stdout immediately after f() so the rest of the subtest
	// (assertions, t.Logf, etc.) writes to the real stdout. t.Cleanup is a
	// safety net in case f panics before we get here.
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = originalStdout })

	f()
	os.Stdout = originalStdout
	w.Close()

	out, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("failed to read captured stdout: %v", readErr)
	}
	return string(out)
}

func TestCPrinterHumanMode(t *testing.T) {
	methods := []struct {
		name string
		call func(p *CPrinter)
	}{
		{"Print", func(p *CPrinter) { p.Print("the message") }},
		{"PrintNorm", func(p *CPrinter) { p.PrintNorm("the message") }},
		{"Info", func(p *CPrinter) { p.Info("the message") }},
		{"Success", func(p *CPrinter) { p.Success("the message") }},
		{"Warning", func(p *CPrinter) { p.Warning("the message") }},
		{"YellowPrint", func(p *CPrinter) { p.YellowPrint("the message") }},
		{"GreenPrint", func(p *CPrinter) { p.GreenPrint("the message") }},
		{"PrintHeader", func(p *CPrinter) { p.PrintHeader("the message") }},
		{"PrintDetail", func(p *CPrinter) { p.PrintDetail("aLabel", "the message") }},
		{"OrganizationInfo", func(p *CPrinter) { p.OrganizationInfo("the message") }},
		{"Prompt", func(p *CPrinter) { p.Prompt("the message") }},
	}

	for _, m := range methods {
		t.Run(m.name+"_streams_to_stdout", func(t *testing.T) {
			printer := NewCPrinter(false)

			output := captureStdout(t, func() { m.call(printer) })

			assert.Contains(t, output, "the message", "expected %s to stream the message", m.name)
		})

		t.Run(m.name+"_records_the_message", func(t *testing.T) {
			printer := NewCPrinter(false)

			captureStdout(t, func() { m.call(printer) })

			assert.Len(t, printer.Messages(), 1, "expected %s to record exactly one message", m.name)
			assert.Contains(t, printer.Messages()[0].Text, "the message")
		})
	}
}

func TestCPrinterJSONMode(t *testing.T) {
	methods := []struct {
		name string
		call func(p *CPrinter)
	}{
		{"Print", func(p *CPrinter) { p.Print("the message") }},
		{"PrintNorm", func(p *CPrinter) { p.PrintNorm("the message") }},
		{"Info", func(p *CPrinter) { p.Info("the message") }},
		{"Success", func(p *CPrinter) { p.Success("the message") }},
		{"Warning", func(p *CPrinter) { p.Warning("the message") }},
		{"YellowPrint", func(p *CPrinter) { p.YellowPrint("the message") }},
		{"GreenPrint", func(p *CPrinter) { p.GreenPrint("the message") }},
		{"PrintHeader", func(p *CPrinter) { p.PrintHeader("the message") }},
		{"PrintDetail", func(p *CPrinter) { p.PrintDetail("aLabel", "the message") }},
		{"OrganizationInfo", func(p *CPrinter) { p.OrganizationInfo("the message") }},
		{"Prompt", func(p *CPrinter) { p.Prompt("the message") }},
	}

	for _, m := range methods {
		t.Run(m.name+"_writes_nothing_to_stdout", func(t *testing.T) {
			printer := NewCPrinter(false)
			printer.SetFormat(FormatJSON)

			output := captureStdout(t, func() { m.call(printer) })

			assert.Empty(t, output, "expected %s to suppress stdout in json mode", m.name)
		})

		t.Run(m.name+"_still_records_the_message", func(t *testing.T) {
			printer := NewCPrinter(false)
			printer.SetFormat(FormatJSON)

			captureStdout(t, func() { m.call(printer) })

			assert.Len(t, printer.Messages(), 1, "expected %s to record even when silent", m.name)
			assert.Contains(t, printer.Messages()[0].Text, "the message")
		})
	}
}

func TestCPrinterRecordsLevels(t *testing.T) {
	t.Run("Info_records_info_level", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)

		printer.Info("the message")

		assert.Equal(t, "info", printer.Messages()[0].Level)
	})

	t.Run("Success_records_success_level", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)

		printer.Success("the message")

		assert.Equal(t, "success", printer.Messages()[0].Level)
	})

	t.Run("Warning_records_warning_level", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)

		printer.Warning("the message")

		assert.Equal(t, "warning", printer.Messages()[0].Level)
	})

	t.Run("PrintVerbose_records_verbose_level_when_verbose_enabled", func(t *testing.T) {
		printer := NewCPrinter(true)
		printer.SetFormat(FormatJSON)

		printer.PrintVerbose("the message")

		assert.Len(t, printer.Messages(), 1)
		assert.Equal(t, "verbose", printer.Messages()[0].Level)
	})

	t.Run("PrintVerbose_records_nothing_when_verbose_disabled", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)

		printer.PrintVerbose("a message")

		assert.Empty(t, printer.Messages())
	})
}

func TestCPrinterEmit(t *testing.T) {
	t.Run("writes_nothing_when_not_in_json_mode", func(t *testing.T) {
		printer := NewCPrinter(false)

		output := captureStdout(t, func() {
			assert.NoError(t, printer.Emit(map[string]any{"foo": "bar"}))
		})

		assert.Empty(t, output)
	})

	t.Run("emits_a_single_json_object_with_the_result_fields_and_a_messages_array", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)
		printer.Info("the first message")
		printer.Success("the second message")

		output := captureStdout(t, func() {
			assert.NoError(t, printer.Emit(map[string]any{
				"buildId": "abld_the_id",
				"appId":   "app_the_app",
			}))
		})

		var decoded struct {
			BuildID  string    `json:"buildId"`
			AppID    string    `json:"appId"`
			Messages []Message `json:"messages"`
		}
		assert.NoError(t, json.Unmarshal([]byte(output), &decoded))
		assert.Equal(t, "abld_the_id", decoded.BuildID)
		assert.Equal(t, "app_the_app", decoded.AppID)
		assert.Equal(t, []Message{
			{Level: "info", Text: "the first message"},
			{Level: "success", Text: "the second message"},
		}, decoded.Messages)
	})

	t.Run("emits_an_empty_messages_array_when_no_messages_were_recorded", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)

		output := captureStdout(t, func() {
			assert.NoError(t, printer.Emit(map[string]any{"status": "succeeded"}))
		})

		assert.Contains(t, output, `"messages":[]`)
	})

	t.Run("produces_a_single_line_of_output", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)
		printer.Info("the message")

		output := captureStdout(t, func() {
			assert.NoError(t, printer.Emit(map[string]any{"foo": "bar"}))
		})

		// Exactly one trailing newline from fmt.Println.
		assert.Equal(t, 1, lineCount(output), "expected a single json line in: %q", output)
	})

	t.Run("produces_valid_json", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)
		printer.Info("the message")

		output := captureStdout(t, func() {
			assert.NoError(t, printer.Emit(map[string]any{"foo": "bar"}))
		})

		assert.True(t, json.Valid([]byte(output)), "output must be valid JSON: %q", output)
	})

	t.Run("overwrites_any_messages_key_provided_by_the_caller", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)
		printer.Info("the real message")

		output := captureStdout(t, func() {
			assert.NoError(t, printer.Emit(map[string]any{
				"status":   "succeeded",
				"messages": "a caller-supplied value",
			}))
		})

		// The caller's "messages" must not survive; the printer owns that slot.
		assert.NotContains(t, output, "a caller-supplied value")
		assert.Contains(t, output, "the real message")
	})
}

func TestCPrinterConcurrentRecording(t *testing.T) {
	t.Run("record_is_safe_to_call_concurrently", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)

		var wg sync.WaitGroup
		workers := 16
		perWorker := 64
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < perWorker; j++ {
					printer.Info("the message")
				}
			}()
		}
		wg.Wait()

		assert.Len(t, printer.Messages(), workers*perWorker)
	})
}

func TestCPrinterIsJSON(t *testing.T) {
	t.Run("returns_false_by_default", func(t *testing.T) {
		printer := NewCPrinter(false)

		assert.False(t, printer.IsJSON())
	})

	t.Run("returns_true_when_format_is_json", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat(FormatJSON)

		assert.True(t, printer.IsJSON())
	})

	t.Run("returns_false_for_unknown_format", func(t *testing.T) {
		printer := NewCPrinter(false)
		printer.SetFormat("yaml")

		assert.False(t, printer.IsJSON())
	})
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	lines := 0
	for _, r := range s {
		if r == '\n' {
			lines++
		}
	}
	// Count the trailing non-newline content as a line, too.
	if s[len(s)-1] != '\n' {
		lines++
	}
	return lines
}
