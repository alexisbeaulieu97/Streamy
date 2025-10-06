package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVersionConstraint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "valid", input: "1.x", want: 1},
		{name: "valid spaced", input: " 2.x ", want: 2},
		{name: "invalid format", input: "1", wantErr: true},
		{name: "invalid suffix", input: "1.y", wantErr: true},
		{name: "negative", input: "-1.x", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			constraint, err := ParseVersionConstraint(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, constraint)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, constraint)
			require.Equal(t, tc.want, constraint.MajorVersion)
		})
	}
}

func TestVersionConstraintSatisfies(t *testing.T) {
	constraint := MustParseVersionConstraint("1.x")

	require.True(t, constraint.Satisfies("1.0.0"))
	require.True(t, constraint.Satisfies("1.99.5"))
	require.False(t, constraint.Satisfies("2.0.0"))
	require.False(t, constraint.Satisfies("abc"))
}

func TestMustParseVersionConstraintPanicsOnInvalid(t *testing.T) {
	require.Panics(t, func() {
		MustParseVersionConstraint("1")
	})
}
