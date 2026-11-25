package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDevEUI(t *testing.T) {
	for _, in := range []string{
		"70b3d57ed0001234",
		"70B3D57ED0001234",
		"70:B3:D5:7E:D0:00:12:34",
		"70-b3-d5-7e-d0-00-12-34",
		" 70 b3 d5 7e d0 00 12 34 ",
	} {
		got, err := normalizeDevEUI(in)
		require.NoError(t, err, in)
		assert.Equal(t, "70b3d57ed0001234", got, in)
	}
	for _, in := range []string{"", "70b3d57ed000123", "70b3d57ed00012345", "70b3d57ed000123g"} {
		_, err := normalizeDevEUI(in)
		assert.ErrorIs(t, err, errDevEUIInvalid, in)
	}
}

func TestNormalizeMeterType(t *testing.T) {
	cases := map[string]string{
		"": "", " ": "",
		"water": "water", "Apă": "water", "APA": "water", "Вода": "water",
		"gas": "gas", "GAZ": "gas", "газ": "gas",
	}
	for in, want := range cases {
		got, err := normalizeMeterType(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	_, err := normalizeMeterType("steam")
	assert.ErrorIs(t, err, errMeterTypeInvalid)
}

func TestSearchPattern(t *testing.T) {
	assert.Equal(t, "%str. mare%", searchPattern("  Str. Mare "))
	assert.Equal(t, "%70b3d5%", searchPattern("70:B3:D5"), "a DevEUI loses its separators")
	assert.Equal(t, "%ap-12%", searchPattern("ap-12"), "a text that is not hex keeps its dashes")
	assert.Equal(t, `%50\%\_off%`, searchPattern("50%_off"), "wildcards match only themselves")
}
