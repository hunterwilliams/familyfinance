package namematch

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hunterwilliams/familyfinance/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func members() (models.Member, models.Member) {
	a := models.Member{ID: uuid.New(), Name: "Hunter Williams", NameTh: "ฮันเตอร์ วิลเลียมส์"}
	b := models.Member{ID: uuid.New(), Name: "Jane Doe", NameTh: "เจน โด"}
	return a, b
}

func TestMatchEnglishName(t *testing.T) {
	a, b := members()
	got := Match("HUNTER WILLIAMS", []models.Member{a, b})
	require.NotNil(t, got.MemberID)
	assert.Equal(t, a.ID, *got.MemberID)
	assert.False(t, got.IsJoint)
}

func TestMatchThaiName(t *testing.T) {
	a, b := members()
	got := Match("คุณฮันเตอร์ วิลเลียมส์", []models.Member{a, b})
	require.NotNil(t, got.MemberID)
	assert.Equal(t, a.ID, *got.MemberID)
}

func TestMatchTokenOrder(t *testing.T) {
	a, b := members()
	got := Match("Williams Hunter", []models.Member{a, b})
	require.NotNil(t, got.MemberID)
	assert.Equal(t, a.ID, *got.MemberID)
}

func TestMatchIgnoresBankPrefix(t *testing.T) {
	a, b := members()
	got := Match("SCB HUNTER WILLIAMS", []models.Member{a, b})
	require.NotNil(t, got.MemberID)
	assert.Equal(t, a.ID, *got.MemberID)
	assert.False(t, got.IsJoint)
}

func TestJointCombinedEnglish(t *testing.T) {
	a, b := members()
	got := Match("HUNTER WILLIAMS / JANE DOE", []models.Member{a, b})
	assert.Nil(t, got.MemberID)
	assert.True(t, got.IsJoint)
}

func TestJointAndConjunction(t *testing.T) {
	a, b := members()
	got := Match("Hunter Williams and Jane Doe", []models.Member{a, b})
	assert.True(t, got.IsJoint)
}

func TestJointThaiCombined(t *testing.T) {
	a, b := members()
	got := Match("ฮันเตอร์ วิลเลียมส์ และ เจน โด", []models.Member{a, b})
	assert.True(t, got.IsJoint)
}

func TestJointConcatenated(t *testing.T) {
	a, b := members()
	got := Match("HUNTERWILLIAMSJANEDOE", []models.Member{a, b})
	assert.True(t, got.IsJoint)
}

func TestLongerUnmatchedNameTreatedAsJoint(t *testing.T) {
	a, b := members()
	got := Match("HUNTER WILLIAMS SOMETHINGELSE", []models.Member{a, b})
	assert.Nil(t, got.MemberID)
	assert.True(t, got.IsJoint)
}

func TestNoMatch(t *testing.T) {
	a, b := members()
	got := Match("7-Eleven Silom", []models.Member{a, b})
	assert.Nil(t, got.MemberID)
	assert.False(t, got.IsJoint)
}

func TestEmptyHolder(t *testing.T) {
	a, b := members()
	got := Match("", []models.Member{a, b})
	assert.Equal(t, Result{}, got)
}

func TestFirstNameOnly(t *testing.T) {
	a, b := members()
	got := Match("Hunter", []models.Member{a, b})
	require.NotNil(t, got.MemberID)
	assert.Equal(t, a.ID, *got.MemberID)
	assert.False(t, got.IsJoint)
}

func TestTypoHutner(t *testing.T) {
	a, b := members()
	got := Match("HUTNER WILLIAMS", []models.Member{a, b})
	require.NotNil(t, got.MemberID)
	assert.Equal(t, a.ID, *got.MemberID)
}

func TestMatchAllPrefersUnique(t *testing.T) {
	a, b := members()
	got := MatchAll([]string{"", "Hunter"}, []models.Member{a, b})
	require.NotNil(t, got.MemberID)
	assert.Equal(t, a.ID, *got.MemberID)
}
