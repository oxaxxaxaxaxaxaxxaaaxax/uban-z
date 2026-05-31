package domain

import "fmt"

// Role names a closed set of user roles in priority order for cancellation rights.
type Role string

const (
	RoleStudentB Role = "student_b"
	RoleStudentM Role = "student_m"
	RoleStudentA Role = "student_a"
	RoleTeacher  Role = "teacher"
	RoleAdmin    Role = "admin"
)

// rank returns a strict-ordering integer; higher = more authority.
// Unknown roles return 0 so they can never out-rank a known role.
func (r Role) Rank() int {
	switch r {
	case RoleStudentB:
		return 1
	case RoleStudentM:
		return 2
	case RoleStudentA:
		return 3
	case RoleTeacher:
		return 4
	case RoleAdmin:
		return 5
	default:
		return 0
	}
}

func (r Role) IsKnown() bool {
	return r.Rank() > 0
}

// CanCancelOther reports whether a request from r may cancel a booking
// whose creator had role `other`. Strict inequality — equal rank cannot
// cancel each other's bookings (only the owner themselves can).
//
// Admin is a superuser exception: admin may cancel any known role, including
// other admins. Parser-service writes class rows with creator_role=admin, and
// only an admin caller is allowed to cancel them.
func (r Role) CanCancelOther(other Role) bool {
	if !r.IsKnown() || !other.IsKnown() {
		return false
	}
	if r == RoleAdmin {
		return true
	}
	return r.Rank() > other.Rank()
}

// ParseRole validates an external string and returns the typed Role.
func ParseRole(s string) (Role, error) {
	role := Role(s)
	if !role.IsKnown() {
		return "", fmt.Errorf("unknown role %q", s)
	}
	return role, nil
}
