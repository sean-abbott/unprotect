package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/hcl"

	flag "github.com/ogier/pflag"
)

// flags
var (
	resource string
)

// These structures were lifted from https://github.com/gruntwork-io/terragrunt/blob/master/remote/terraform_state_file.go
// The structure of the Terraform .tfstate file
type TerraformState struct {
	Version int
	Serial  int
	Backend *TerraformBackend
	Modules []TerraformStateModule
}

// The structure of the "backend" section of the Terraform .tfstate file
type TerraformBackend struct {
	Type   string
	Config map[string]interface{}
}

// The structure of a "module" section of the Terraform .tfstate file
type TerraformStateModule struct {
	Path      []string
	Outputs   map[string]interface{}
	Resources map[string]interface{}
}

// end lifted structs

// helper functions
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

//  https://stackoverflow.com/questions/37194739/how-check-a-file-contain-string-or-not-in-golang
// It might be hacky but easier to just treat the ACL as string to get the profile
func getAwsProfileFromFile() string {
}

func getTerraformState() *TerraformState {
	var (
		cmdOut []byte
		err    error
	)
	cmd := "terraform"
	args := []string{"state", "pull"}
	if cmdOut, err = exec.Command(cmd, args...).Output(); err != nil {
		fmt.Printf("'terraform state pull' failed. Are you in a terraform directory?")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(cmdOut) == 0 {
		fmt.Printf("terraform state pull did not return anything. Are you sure you're in a terraform module?\n")
		os.Exit(1)
	}

	terraformState := &TerraformState{}

	if err := json.Unmarshal(cmdOut, terraformState); err != nil {
		fmt.Printf("Failed to unmarshall terraform state pull output.\n")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return terraformState

}

func getInstanceSlice(terraformState *TerraformState) []string {
	instances := []string{}

	for i, module := range terraformState.Modules {
		for key, _ := range module.Resources {
			if strings.HasPrefix(key, "aws_instance") {
				fmt.Printf("Module %d has an aws_instance.\n", i)
				instances = append(instances, key)
			}
		}
	}

	return instances
}

func main() {
	flag.Parse()

	terraformState := getTerraformState()
	instanceSlice := getInstanceSlice(terraformState)
	if flag.NFlag() == 0 {
		fmt.Printf("Instances available: %v\n", instanceSlice)
		os.Exit(0)
	} else if stringInSlice(resource, instanceSlice) {
		fmt.Printf("Unprotecting %s!\n", resource)
	} else {
		fmt.Printf("Could not find resource %s. Instances available: %v\n", resource, instanceSlice)
	}
}

func init() {
	flag.StringVarP(&resource, "resource", "r", "", "Resource to unprotect")
}
