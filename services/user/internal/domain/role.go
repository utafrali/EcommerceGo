package domain

// Role constants define the allowed user roles.
const (
	RoleCustomer = "customer"
	RoleAdmin    = "admin"
	RoleSeller   = "seller"
)

// ValidRoles returns the set of valid user roles.
func ValidRoles() []string {
	return []string{RoleCustomer, RoleAdmin, RoleSeller}
}

// IsValidRole checks whether the given role string is a valid user role.
func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if r == role {
			return true
		}
	}
	return false
}
