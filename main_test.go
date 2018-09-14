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
