package rbac

import "github.com/google/uuid"

const (
	RoleObserver     = "observer"
	RoleDeveloper    = "developer"
	RoleServiceOwner = "service_owner"
)

func EffectiveRole(serviceOwnerID, userID uuid.UUID, accessRole string) string {
	if serviceOwnerID == userID {
		return RoleServiceOwner
	}
	if accessRole != "" {
		return accessRole
	}
	return RoleObserver
}
