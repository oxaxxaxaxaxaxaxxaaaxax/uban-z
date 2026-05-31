package domain_test

import (
	"testing"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

func TestRole_Rank(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role domain.Role
		want int
	}{
		{domain.RoleStudentB, 1},
		{domain.RoleStudentM, 2},
		{domain.RoleStudentA, 3},
		{domain.RoleTeacher, 4},
		{domain.RoleAdmin, 5},
		{domain.Role("garbage"), 0},
		{domain.Role(""), 0},
	}

	for _, tc := range cases {
		if got := tc.role.Rank(); got != tc.want {
			t.Errorf("%q.Rank() = %d, want %d", tc.role, got, tc.want)
		}
	}
}

func TestRole_CanCancelOther(t *testing.T) {
	t.Parallel()

	type pair struct {
		canceller domain.Role
		owner     domain.Role
		want      bool
	}

	cases := []pair{
		{domain.RoleAdmin, domain.RoleTeacher, true},
		{domain.RoleAdmin, domain.RoleAdmin, true}, // admin bypass: can cancel parser rows
		{domain.RoleTeacher, domain.RoleStudentA, true},
		{domain.RoleStudentA, domain.RoleStudentM, true},
		{domain.RoleStudentM, domain.RoleStudentB, true},
		{domain.RoleTeacher, domain.RoleTeacher, false},   // equal rank, non-admin
		{domain.RoleStudentB, domain.RoleStudentB, false}, // equal rank
		{domain.RoleStudentB, domain.RoleAdmin, false},    // lower vs higher
		{domain.RoleStudentM, domain.RoleTeacher, false},
		{domain.RoleTeacher, domain.RoleAdmin, false}, // teacher cannot cancel parser/admin row
		{domain.Role("garbage"), domain.RoleStudentB, false},
		{domain.RoleAdmin, domain.Role("garbage"), false},
	}

	for _, tc := range cases {
		got := tc.canceller.CanCancelOther(tc.owner)
		if got != tc.want {
			t.Errorf("%q.CanCancelOther(%q) = %v, want %v", tc.canceller, tc.owner, got, tc.want)
		}
	}
}

func TestParseRole(t *testing.T) {
	t.Parallel()

	for _, valid := range []string{"student_b", "student_m", "student_a", "teacher", "admin"} {
		if _, err := domain.ParseRole(valid); err != nil {
			t.Errorf("ParseRole(%q) unexpected err: %v", valid, err)
		}
	}

	for _, invalid := range []string{"", "STUDENT_B", "root", "user"} {
		if _, err := domain.ParseRole(invalid); err == nil {
			t.Errorf("ParseRole(%q) expected err", invalid)
		}
	}
}
