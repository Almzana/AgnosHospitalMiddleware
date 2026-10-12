package model

import "testing"

func TestPatientSearchFilter_Normalize(t *testing.T) {
	f := PatientSearchFilter{
		NationalID: " 123 ", PassportID: "\tAB1 ", FirstName: " Somchai ", MiddleName: " ",
		LastName: "Jaidee\n", PhoneNumber: " 081 ", Email: " a@b.com ",
	}
	f.Normalize()

	want := PatientSearchFilter{
		NationalID: "123", PassportID: "AB1", FirstName: "Somchai", MiddleName: "",
		LastName: "Jaidee", PhoneNumber: "081", Email: "a@b.com",
	}
	if f != want {
		t.Errorf("got %+v, want %+v", f, want)
	}
}

func TestPatientSearchFilter_Identifier(t *testing.T) {
	tests := []struct {
		name string
		f    PatientSearchFilter
		want string
	}{
		{"none", PatientSearchFilter{FirstName: "x"}, ""},
		{"national only", PatientSearchFilter{NationalID: "1"}, "1"},
		{"passport only", PatientSearchFilter{PassportID: "P1"}, "P1"},
		{"national wins over passport", PatientSearchFilter{NationalID: "1", PassportID: "P1"}, "1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.f.Identifier(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
