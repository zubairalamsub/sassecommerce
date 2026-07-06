package service

import "testing"

func TestValidatePasswordPolicy(t *testing.T) {
	cases := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"strong password accepted", "str0ng-Test-Passw0rd", nil},
		{"too short rejected", "hunter2!!", errPasswordTooShort},
		{"eleven chars rejected", "password123", errPasswordTooShort},
		{"common password rejected", "password1234", errPasswordCommon},
		{"common password rejected case-insensitively", "PASSWORD1234", errPasswordCommon},
		{"keyboard walk rejected", "qwertyuiop12", errPasswordCommon},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validatePasswordPolicy(tc.password); err != tc.wantErr {
				t.Errorf("validatePasswordPolicy(%q) = %v, want %v", tc.password, err, tc.wantErr)
			}
		})
	}
}
