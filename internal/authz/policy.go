package authz

type Role string
type Permission string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

const (
	PermissionReadSelf Permission = "user:read:self"
	PermissionReadAny  Permission = "user:read:any"
)

var rolePermissions = map[Role]map[Permission]struct{}{
	RoleUser: {
		PermissionReadSelf: {},
	},
	RoleAdmin: {
		PermissionReadSelf: {},
		PermissionReadAny:  {},
	},
}

func HasPermission(role Role, permission Permission) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = permissions[permission]
	return ok
}
