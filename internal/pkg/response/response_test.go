package response

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorHidesRawDetailsWhenDisabled(t *testing.T) {
	t.Cleanup(func() { SetExposeErrorDetails(true) })
	raw := `get user: ERROR: relation "users" does not exist (SQLSTATE 42P01)`

	SetExposeErrorDetails(true)
	rec := httptest.NewRecorder()
	Error(rec, 500, "failed to get user", raw)
	if !strings.Contains(rec.Body.String(), "SQLSTATE") {
		t.Fatalf("development: want raw detail in body, got %s", rec.Body.String())
	}

	SetExposeErrorDetails(false)
	rec = httptest.NewRecorder()
	ErrorWithLog(rec, nil, 500, "failed to get user", raw)
	body := rec.Body.String()
	if strings.Contains(body, "SQLSTATE") || strings.Contains(body, "relation") {
		t.Fatalf("production: raw detail leaked: %s", body)
	}
	if !strings.Contains(body, `"message":"failed to get user"`) || !strings.Contains(body, `"errors":{}`) {
		t.Fatalf("production: unexpected body %s", body)
	}

	// Structured field errors stay visible: they are written for the client.
	rec = httptest.NewRecorder()
	Error(rec, 400, "missing required fields", map[string]string{"email": "required"})
	if !strings.Contains(rec.Body.String(), `"email":"required"`) {
		t.Fatalf("production: field errors dropped: %s", rec.Body.String())
	}
}
