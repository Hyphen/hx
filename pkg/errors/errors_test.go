package errors

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleHTTPErrorIncludesRequestContext(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:4000/api/organizations/org-1/apps/app-1/dockerfile", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"message":"invalid body"}`)),
		Request:    req,
	}

	wrapped := HandleHTTPError(resp)

	assert.EqualError(
		t,
		wrapped,
		"bad request for POST http://localhost:4000/api/organizations/org-1/apps/app-1/dockerfile: invalid body",
	)
}
