package profile

import "testing"

// The example file in the repository must stay a valid profiles file.
func TestTheExampleProfilesFileIsValid(t *testing.T) {
	set, err := Load("../../profiles.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if set["summarize"].Output == "" || set["book-to-pack"].OutputDir != "pack" {
		t.Errorf("unexpected profiles: %v", set.Names())
	}
}
