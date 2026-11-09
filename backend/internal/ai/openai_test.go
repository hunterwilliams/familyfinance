package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeKey(t *testing.T) {
	assert.Equal(t, "sk-abc", NormalizeKey("  sk-abc \n"))
	assert.Equal(t, "sk-abc", NormalizeKey(`"sk-abc"`))
	assert.Equal(t, "sk-abc", NormalizeKey("Bearer sk-abc"))
	assert.Equal(t, "xai-abc", NormalizeKey("bearer xai-abc"))
}

func TestBaseURLAndModel(t *testing.T) {
	assert.Equal(t, "https://api.openai.com/v1", BaseURLForKey("sk-proj-abc", ""))
	assert.Equal(t, "https://openrouter.ai/api/v1", BaseURLForKey("sk-or-v1-abc", ""))
	assert.Equal(t, "https://api.x.ai/v1", BaseURLForKey("xai-abc", ""))
	assert.Equal(t, "http://localhost:4000/v1", BaseURLForKey("sk-abc", "http://localhost:4000/v1/"))
	assert.Equal(t, "gpt-4o", DefaultModel("sk-abc", ""))
	assert.Equal(t, "openai/gpt-4o", DefaultModel("sk-or-v1-abc", "gpt-4o"))
	assert.Equal(t, "anthropic/claude-sonnet-4", DefaultModel("sk-or-v1-abc", "anthropic/claude-sonnet-4"))
	assert.Equal(t, "grok-2-vision-1212", DefaultModel("xai-abc", "gpt-4o"))
	assert.Equal(t, "grok-4", DefaultModel("xai-abc", "grok-4"))
}

func TestValidateInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Incorrect API key provided","code":"invalid_api_key"}}`))
	}))
	defer srv.Close()

	c := NewWithBase("sk-bad", "gpt-4o", srv.URL)
	err := c.Validate(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

func TestValidateOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/models", r.URL.Path)
		assert.Equal(t, "Bearer sk-good", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	c := NewWithBase(" sk-good ", "gpt-4o", srv.URL)
	require.NoError(t, c.Validate(context.Background()))
}
