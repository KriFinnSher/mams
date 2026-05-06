package rbac

import (
	"testing"

	"github.com/google/uuid"
)

func TestEffectiveRole(t *testing.T) {
	t.Parallel()

	ownerID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name       string
		serviceOwn uuid.UUID
		userID     uuid.UUID
		accessRole string
		want       string
	}{
		{
			name:       "owner has service owner role",
			serviceOwn: ownerID,
			userID:     ownerID,
			accessRole: RoleDeveloper,
			want:       RoleServiceOwner,
		},
		{
			name:       "non owner gets explicit access role",
			serviceOwn: ownerID,
			userID:     otherID,
			accessRole: RoleDeveloper,
			want:       RoleDeveloper,
		},
		{
			name:       "non owner without access gets observer",
			serviceOwn: ownerID,
			userID:     otherID,
			accessRole: "",
			want:       RoleObserver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := EffectiveRole(tt.serviceOwn, tt.userID, tt.accessRole)
			if got != tt.want {
				t.Fatalf("EffectiveRole() = %q, want %q", got, tt.want)
			}
		})
	}
}
