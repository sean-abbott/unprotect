package main

import "testing"

func TestValidateInstance(t *testing.T) {
	resource = "myinstance"
	fakeInstance := ResourceInstance{Resource: "myinstance", Id: "i-12356"}
	fakeMap := map[string]ResourceInstance{"myinstance": fakeInstance}
	fakeResult := TerraformInstanceResult{Instances: fakeMap, Error: nil}

	result := validateInstance(fakeResult)

	if result.Resource != "myinstance" {
		t.Errorf("validateInstance failed. got: %s, want: %s", result.Resource, "myinstance")
	}
}

func TestValidateAwsProfile(t *testing.T) {
	tables := []struct {
		foundProfiles []string
		givenProfile  string
		expected      string
	}{
		{[]string{"myprofile"}, "myprofile", "myprofile"},
		{[]string{}, "", "default"},
	}

	for _, table := range tables {
		profile = table.givenProfile
		result, _ := validateAwsProfile(table.foundProfiles)
		if result != table.expected {
			t.Errorf("validateAwsProfile failed. got: %s, want: %s\n", result, table.expected)
		}
	}
}
