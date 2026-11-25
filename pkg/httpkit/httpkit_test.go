package httpkit_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/waneock/app-lora/pkg/httpkit"
)

func TestWriteError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("typed error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := httpkit.NewError(http.StatusConflict, "login_taken", "the login is taken")
		httpkit.WriteError(rec, req, logger, err)
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.JSONEq(t, `{"code":"login_taken","message":"the login is taken"}`, rec.Body.String())
	})

	t.Run("plain error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpkit.WriteError(rec, req, logger, errors.New("boom"))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.JSONEq(t, `{"code":"internal","message":"internal error"}`, rec.Body.String())
	})
}

func TestDecodeJSON(t *testing.T) {
	type body struct {
		Name string `json:"name"`
	}
	decode := func(contentType, payload string) (body, error) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
		req.Header.Set("Content-Type", contentType)
		var dst body
		err := httpkit.DecodeJSON(httptest.NewRecorder(), req, &dst)
		return dst, err
	}
	code := func(err error) string {
		herr, ok := errors.AsType[*httpkit.Error](err)
		if !ok {
			return ""
		}
		return herr.Code
	}

	t.Run("valid", func(t *testing.T) {
		got, err := decode("application/json; charset=utf-8", `{"name":"a"}`)
		require.NoError(t, err)
		assert.Equal(t, "a", got.Name)
	})

	t.Run("form content type", func(t *testing.T) {
		_, err := decode("application/x-www-form-urlencoded", `{"name":"a"}`)
		assert.Equal(t, "unsupported_media_type", code(err))
	})

	t.Run("unknown field", func(t *testing.T) {
		_, err := decode("application/json", `{"name":"a","extra":1}`)
		assert.Equal(t, "invalid_json", code(err))
	})

	t.Run("two values", func(t *testing.T) {
		_, err := decode("application/json", `{"name":"a"}{"name":"b"}`)
		assert.Equal(t, "invalid_json", code(err))
	})
}
