package auth_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAndParse(t *testing.T) {
	secret := "test-secret"
	memberID := uuid.New()
	hh := uuid.New()
	token, err := auth.Issue(secret, auth.Claims{
		LineUserID:  "U123",
		HouseholdID: hh,
		MemberID:    memberID,
		DisplayName: "Alex",
	})
	require.NoError(t, err)

	got, err := auth.Parse(secret, token)
	require.NoError(t, err)
	assert.Equal(t, "U123", got.LineUserID)
	assert.Equal(t, hh, got.HouseholdID)
	assert.Equal(t, memberID, got.MemberID)
}

func TestParseRejectsWrongSecret(t *testing.T) {
	token, err := auth.Issue("a", auth.Claims{LineUserID: "U1"})
	require.NoError(t, err)
	_, err = auth.Parse("b", token)
	assert.Error(t, err)
}
