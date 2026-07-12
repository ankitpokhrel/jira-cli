package list

import "testing"

func TestResolveBoardID(t *testing.T) {
	tests := []struct {
		name         string
		configuredID int
		override     int
		overridden   bool
		want         int
	}{
		{name: "flag not given uses configured board", configuredID: 10, override: 0, overridden: false, want: 10},
		{name: "flag replaces configured board", configuredID: 10, override: 20, overridden: true, want: 20},
		{name: "flag is honored even when set to zero", configuredID: 10, override: 0, overridden: true, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveBoardID(tc.configuredID, tc.override, tc.overridden)
			if got != tc.want {
				t.Errorf("resolveBoardID(%d, %d, %t) = %d, want %d", tc.configuredID, tc.override, tc.overridden, got, tc.want)
			}
		})
	}
}

func TestResolveBoardName(t *testing.T) {
	tests := []struct {
		name           string
		configuredID   int
		configuredName string
		boardID        int
		want           string
	}{
		{
			name:           "configured board keeps its name",
			configuredID:   10,
			configuredName: "My Board",
			boardID:        10,
			want:           "My Board",
		},
		{
			name:           "overridden board shows its ID instead of the configured name",
			configuredID:   10,
			configuredName: "My Board",
			boardID:        20,
			want:           "#20",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveBoardName(tc.configuredID, tc.configuredName, tc.boardID)
			if got != tc.want {
				t.Errorf("resolveBoardName(%d, %q, %d) = %q, want %q", tc.configuredID, tc.configuredName, tc.boardID, got, tc.want)
			}
		})
	}
}
