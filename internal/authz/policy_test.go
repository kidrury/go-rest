package authz

import "testing"

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		role       Role
		permission Permission
		want       bool
	}{
		{
			name:       "user can read self",
			role:       RoleUser,
			permission: PermissionReadSelf,
			want:       true,
		},
		{
			name:       "user cannot read any user",
			role:       RoleUser,
			permission: PermissionReadAny,
			want:       false,
		},
		{
			name:       "admin can read self",
			role:       RoleAdmin,
			permission: PermissionReadSelf,
			want:       true,
		},
		{
			name:       "admin can read any user",
			role:       RoleAdmin,
			permission: PermissionReadAny,
			want:       true,
		},
		{
			name:       "unknown role has no permissions",
			role:       Role("unknown"),
			permission: PermissionReadSelf,
			want:       false,
		},
		{
			name:       "unknown permission is denied",
			role:       RoleAdmin,
			permission: Permission("user:delete"),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPermission(tt.role, tt.permission)

			if got != tt.want {
				t.Fatalf(
					"HasPermission(%q, %q) = %v, want %v",
					tt.role,
					tt.permission,
					got,
					tt.want,
				)
			}
		})
	}
}
