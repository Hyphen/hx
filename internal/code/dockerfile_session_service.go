package code

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Hyphen/cli/pkg/apiconf"
	"github.com/Hyphen/cli/pkg/errors"
	"github.com/Hyphen/cli/pkg/httputil"
)

const dockerfileSessionRequestTimeout = 2 * time.Minute

type DockerfileSessionStatus string

const (
	DockerfileSessionStatusReady               DockerfileSessionStatus = "ready"
	DockerfileSessionStatusRequiresToolResults DockerfileSessionStatus = "requires_tool_results"
	DockerfileSessionStatusFailed              DockerfileSessionStatus = "failed"
	DockerfileSessionStatusClosed              DockerfileSessionStatus = "closed"
)

type DockerfileSessionClient interface {
	StartSession(organizationID, appID string) (*DockerfileSessionCreateResponse, error)
	ContinueSession(organizationID, appID, sessionID string, results []DockerfileToolResult) (*DockerfileTurnResponse, error)
}

type DockerfileSessionService struct {
	baseURL         string
	httpClient      httputil.Client
	verboseCallback func(string)
}

type DockerfileSessionCreateResponse struct {
	Session DockerfileSession      `json:"session"`
	Output  DockerfileTurnResponse `json:"output"`
}

type DockerfileSession struct {
	ID     string                  `json:"id"`
	Status DockerfileSessionStatus `json:"status"`
}

type DockerfileTurnResponse struct {
	SessionID string                     `json:"session_id"`
	Status    DockerfileSessionStatus    `json:"status"`
	Message   DockerfileAssistantMessage `json:"message"`
}

type DockerfileAssistantMessage struct {
	Role      string               `json:"role"`
	Content   *string              `json:"content"`
	ToolCalls []DockerfileToolCall `json:"tool_calls"`
}

type DockerfileToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function DockerfileToolFunction `json:"function"`
}

type DockerfileToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type DockerfileToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     any    `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

type dockerfileContinueRequest struct {
	SessionID string                 `json:"sessionId"`
	Results   []DockerfileToolResult `json:"results"`
}

func NewDockerfileSessionService() *DockerfileSessionService {
	return &DockerfileSessionService{
		baseURL:    apiconf.GetBaseApixUrl(),
		httpClient: httputil.NewHyphenHTTPClientWithTimeout(dockerfileSessionRequestTimeout),
	}
}

func (s *DockerfileSessionService) SetVerboseCallback(cb func(string)) {
	s.verboseCallback = cb
}

func (s *DockerfileSessionService) StartSession(
	organizationID, appID string,
) (*DockerfileSessionCreateResponse, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dockerfile", s.baseURL, organizationID, appID)
	s.logf("Starting Dockerfile session: POST %s", url)
	response := &DockerfileSessionCreateResponse{}
	if err := s.postJSON(url, map[string]any{}, http.StatusCreated, response); err != nil {
		return nil, err
	}
	s.logf(
		"Started Dockerfile session %s with status %s and %d pending tool call(s)",
		response.Session.ID,
		response.Output.Status,
		len(response.Output.Message.ToolCalls),
	)
	s.logf("Session response: %s", summarizeCreateResponseForLog(response))
	return response, nil
}

func (s *DockerfileSessionService) ContinueSession(
	organizationID, appID, sessionID string,
	results []DockerfileToolResult,
) (*DockerfileTurnResponse, error) {
	url := fmt.Sprintf("%s/api/organizations/%s/apps/%s/dockerfile", s.baseURL, organizationID, appID)
	s.logf(
		"Continuing Dockerfile session %s with %d tool result(s): POST %s",
		sessionID,
		len(results),
		url,
	)
	for _, result := range results {
		s.logf("Submitting tool result: %s", summarizeToolResultForLog(result, nil))
	}
	response := &DockerfileTurnResponse{}
	if err := s.postJSON(
		url,
		dockerfileContinueRequest{
			SessionID: sessionID,
			Results:   results,
		},
		http.StatusOK,
		response,
	); err != nil {
		return nil, err
	}
	s.logf(
		"Dockerfile session %s returned status %s with %d pending tool call(s)",
		response.SessionID,
		response.Status,
		len(response.Message.ToolCalls),
	)
	s.logf("Session response: %s", summarizeTurnResponseForLog(response))
	return response, nil
}

func (s *DockerfileSessionService) postJSON(url string, body any, expectedStatusCode int, destination any) error {
	requestBody, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal request body")
	}

	req, err := http.NewRequest(http.MethodPost, url, io.NopCloser(bytes.NewReader(requestBody)))
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	s.logf("Received HTTP %d from %s %s", resp.StatusCode, req.Method, req.URL.String())

	if resp.StatusCode != expectedStatusCode {
		return errors.HandleHTTPError(resp)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Failed to read response body")
	}

	if err := json.Unmarshal(responseBody, destination); err != nil {
		return errors.Wrap(err, "Failed to parse JSON response")
	}

	return nil
}

func (m DockerfileAssistantMessage) ContentOrEmpty() string {
	if m.Content == nil {
		return ""
	}
	return *m.Content
}

func (s *DockerfileSessionService) logf(format string, args ...any) {
	if s.verboseCallback == nil {
		return
	}

	s.verboseCallback(fmt.Sprintf(format, args...))
}

func summarizeCreateResponseForLog(response *DockerfileSessionCreateResponse) string {
	if response == nil {
		return "empty"
	}

	return fmt.Sprintf(
		"session=%s status=%s turn=%s",
		response.Session.ID,
		response.Session.Status,
		summarizeTurnResponseForLog(&response.Output),
	)
}

func summarizeTurnResponseForLog(response *DockerfileTurnResponse) string {
	if response == nil {
		return "empty"
	}

	parts := []string{
		fmt.Sprintf("session=%s", response.SessionID),
		fmt.Sprintf("status=%s", response.Status),
	}

	if content := strings.TrimSpace(response.Message.ContentOrEmpty()); content != "" {
		parts = append(parts, fmt.Sprintf("assistant=%q", previewTextForLog(content, 120)))
	}
	if len(response.Message.ToolCalls) > 0 {
		parts = append(parts, fmt.Sprintf("tool_calls=[%s]", strings.Join(summarizeToolCallsForLog(response.Message.ToolCalls), ", ")))
	} else {
		parts = append(parts, "tool_calls=[]")
	}

	return strings.Join(parts, " ")
}

func summarizeToolCallsForLog(toolCalls []DockerfileToolCall) []string {
	summaries := make([]string, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		summaries = append(summaries, summarizeToolCallForLog(toolCall))
	}
	return summaries
}

func summarizeToolCallForLog(toolCall DockerfileToolCall) string {
	return fmt.Sprintf("%s:%s(%s)", toolCall.ID, toolCall.Function.Name, summarizeToolArgumentsForLog(toolCall.Function.Name, toolCall.Function.Arguments))
}

func summarizeToolResultForLog(result DockerfileToolResult, toolCall *DockerfileToolCall) string {
	name := "unknown"
	if toolCall != nil {
		name = toolCall.Function.Name
	}

	if strings.TrimSpace(result.Error) != "" {
		return fmt.Sprintf("%s:%s error=%q", result.ToolCallID, name, previewTextForLog(result.Error, 120))
	}

	return fmt.Sprintf("%s:%s output=%s", result.ToolCallID, name, summarizeToolOutputForLog(result.Output))
}

func summarizeToolArgumentsForLog(toolName, rawArguments string) string {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(rawArguments), &parsed); err != nil {
		return fmt.Sprintf("raw=%q", previewTextForLog(rawArguments, 80))
	}

	switch toolName {
	case "list_files":
		return fmt.Sprintf(
			"path=%s max_depth=%v max_results=%v",
			stringValueOrDefault(parsed["path"], "."),
			parsed["max_depth"],
			parsed["max_results"],
		)
	case "read_file":
		return fmt.Sprintf(
			"path=%s start_line=%v end_line=%v",
			stringValueOrDefault(parsed["path"], "."),
			parsed["start_line"],
			parsed["end_line"],
		)
	case "write_file":
		content := stringValue(parsed["content"])
		return fmt.Sprintf(
			"path=%s append=%v content_bytes=%d content_lines=%d",
			stringValueOrDefault(parsed["path"], "."),
			parsed["append"],
			len([]byte(content)),
			countLinesForLog(content),
		)
	case "search":
		return fmt.Sprintf(
			"path=%s pattern=%q case_sensitive=%v max_results=%v",
			stringValueOrDefault(parsed["path"], "."),
			stringValue(parsed["pattern"]),
			parsed["case_sensitive"],
			parsed["max_results"],
		)
	default:
		keys := make([]string, 0, len(parsed))
		for key := range parsed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return fmt.Sprintf("keys=%v", keys)
	}
}

func summarizeToolOutputForLog(output any) string {
	switch value := output.(type) {
	case nil:
		return "null"
	case string:
		return fmt.Sprintf("string(len=%d preview=%q)", len(value), previewTextForLog(value, 80))
	case bool, int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", value)
	case listFilesResult:
		return fmt.Sprintf("path=%s entry_count=%d truncated=%v", value.Path, len(value.Entries), value.Truncated)
	case readFileResult:
		return fmt.Sprintf(
			"path=%s start_line=%d end_line=%d total_lines=%d content_bytes=%d",
			value.Path,
			value.StartLine,
			value.EndLine,
			value.TotalLines,
			len([]byte(value.Content)),
		)
	case writeFileResult:
		return fmt.Sprintf("path=%s bytes_written=%d appended=%v", value.Path, value.BytesWritten, value.Appended)
	case searchResult:
		return fmt.Sprintf("path=%s pattern=%q match_count=%d truncated=%v", value.Path, value.Pattern, len(value.Matches), value.Truncated)
	case map[string]any:
		if entries, ok := value["entries"].([]any); ok {
			return fmt.Sprintf("path=%s entry_count=%d truncated=%v", stringValueOrDefault(value["path"], "."), len(entries), value["truncated"])
		}
		if content, ok := value["content"].(string); ok {
			return fmt.Sprintf(
				"path=%s start_line=%v end_line=%v total_lines=%v content_bytes=%d",
				stringValueOrDefault(value["path"], "."),
				value["start_line"],
				value["end_line"],
				value["total_lines"],
				len([]byte(content)),
			)
		}
		if bytesWritten, ok := value["bytes_written"]; ok {
			return fmt.Sprintf(
				"path=%s bytes_written=%v appended=%v",
				stringValueOrDefault(value["path"], "."),
				bytesWritten,
				value["appended"],
			)
		}
		if matches, ok := value["matches"].([]any); ok {
			return fmt.Sprintf(
				"path=%s pattern=%q match_count=%d truncated=%v",
				stringValueOrDefault(value["path"], "."),
				stringValue(value["pattern"]),
				len(matches),
				value["truncated"],
			)
		}

		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return fmt.Sprintf("object(keys=%v)", keys)
	case []any:
		return fmt.Sprintf("array(len=%d)", len(value))
	default:
		return fmt.Sprintf("%v", value)
	}
}

func previewTextForLog(value string, maxLength int) string {
	compact := strings.Join(strings.Fields(value), " ")
	if len(compact) <= maxLength {
		return compact
	}
	return compact[:maxLength-3] + "..."
}

func stringValue(value any) string {
	if parsed, ok := value.(string); ok {
		return parsed
	}
	return ""
}

func stringValueOrDefault(value any, fallback string) string {
	if parsed := stringValue(value); parsed != "" {
		return parsed
	}
	return fallback
}

func countLinesForLog(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}

var _ DockerfileSessionClient = (*DockerfileSessionService)(nil)
