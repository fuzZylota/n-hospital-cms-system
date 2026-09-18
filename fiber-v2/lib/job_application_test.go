package lib

import "testing"

func TestJobApplicationUploadRootAvailable(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hasCV bool
		root  string
		want  bool
	}{
		{"no_cv_no_root", false, "", true},
		{"cv_no_root", true, "", false},
		{"no_cv_root", false, "/fixture", true},
		{"cv_root", true, "/fixture", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := JobApplicationUploadRootAvailable(tc.hasCV, tc.root); got != tc.want {
				t.Fatal("incorrect upload precondition")
			}
		})
	}
}
