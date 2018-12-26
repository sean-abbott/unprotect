package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func openAndUnmarshal(i interface{}, path string) error {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}

	defer f.Close()

	byteValue, _ := ioutil.ReadAll(f)

	err = json.Unmarshal([]byte(byteValue), &i)

	if err != nil {
		fmt.Println(err)
		return err
	}
	return err

}

func TestValidateInstance(t *testing.T) {
	// in main(), resource is provided by flag.Parse as a global
	resource = "myinstance"

	var fakeResult TerraformInstanceResult
	openAndUnmarshal(&fakeResult, "testdata/fakeInstanceResult.json")

	result := validateInstance(fakeResult)

	if result.Resource != "myinstance" {
		t.Errorf("validateInstance failed. got: %s, want: %s", result.Resource, "myinstance")
	}
}

type resourceToStateKeyTest struct {
	file     string
	expected string
}

type resourceAndPath struct {
	Path        []string
	ResourceKey string
}

func TestResourceToStateKeyStr(t *testing.T) {
	tables := []resourceToStateKeyTest{
		{"testdata/resourceToStateKeyStrTests/root.json", "aws_instance.web"},
		{"testdata/resourceToStateKeyStrTests/onelevel.json", "module.prometheus_1.aws_instance.prometheus"},
	}

	for _, table := range tables {
		var testData resourceAndPath
		openAndUnmarshal(&testData, table.file)
		result, _ := resourceToStateKeyStr(testData.Path, testData.ResourceKey)
		if result != table.expected {
			t.Errorf("resourceToStateKeyStr failed. got: %s, expected: %s\n", result, table.expected)
		}
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
