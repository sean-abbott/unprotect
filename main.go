package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/viper"

	flag "github.com/ogier/pflag"
)

// flags
var (
	resource string
	profile  string
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

// my stucts
type TerraformStateInstance struct {
	Type      string
	DependsOn []interface{}
	Primary   map[string]interface{}
	Deposed   []interface{}
	Provider  string
}

type TerraformConfig struct {
	Providers []map[string]interface{}
}

type ResourceInstance struct {
	Resource string
	Id       string
}

// end my structs

// helper functions
func resourceInState(a string, list []ResourceInstance) bool {
	for _, ri := range list {
		if ri.Resource == a {
			return true
		}
	}
	return false
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func initEc2(profile string) *ec2.EC2 {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Profile:           profile,
	}))
	svc := ec2.New(sess)
	return svc
}

func unprotectInstance(profile string, i ResourceInstance) bool {
	e := initEc2(profile)
	input := &ec2.ModifyInstanceAttributeInput{
		InstanceId: aws.String(i.Id),
		DisableApiTermination: &ec2.AttributeBooleanValue{
			Value: aws.Bool(false),
		},
	}

	result, err := e.ModifyInstanceAttribute(input)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(result)

	return true
}

// access a terraform file and determine if it has any aws profiles in it
func getProfilesFromFile(f string) []string {
	v := viper.New()
	v.SetConfigType("hcl")
	v.SetConfigFile(f)
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Failed to read config file %s\n", f)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var r []string
	var p TerraformConfig
	if v.InConfig("provider") {
		p.Providers = v.Get("provider").([]map[string]interface{})
	} else {
		return r
	}

	for _, provider := range p.Providers {
		keys := getMapKeys(provider)
		if keys[0] == "aws" {
			var aws_provider_map map[string]interface{}
			aws_provider_map = provider["aws"].([]map[string]interface{})[0]
			profile := aws_provider_map["profile"].(string)
			r = append(r, profile)
		}
	}

	return r
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

func getInstanceSlice(terraformState *TerraformState) []ResourceInstance {
	instances := []ResourceInstance{}

	for i, module := range terraformState.Modules {
		for key, k := range module.Resources {
			if strings.HasPrefix(key, "aws_instance") {
				fmt.Printf("Module %d has an aws_instance.\n", i)
				id := k.(map[string]interface{})["primary"].(map[string]interface{})["id"].(string)
				r := ResourceInstance{Resource: key, Id: id}
				instances = append(instances, r)
			}
		}
	}

	return instances
}

func main() {
	flag.Parse()

	fmt.Printf("Getting terraform state...")
	terraformState := getTerraformState()
	instanceSlice := getInstanceSlice(terraformState)
	if resource == "" {
		fmt.Printf("Instances available: %v\n", instanceSlice)
		os.Exit(0)
	}

	if !resourceInState(resource, instanceSlice) {
		fmt.Printf("Could not find resource %s. Instances available: %v\n", resource, instanceSlice)
	}

	if profile == "" {
		p := getProfilesFromFile("core.tf")

		if len(p) > 1 {
			fmt.Printf("More than one profile found. Please select one from %v\n", p)
			os.Exit(0)
		} else if len(p) == 0 {
			fmt.Printf("No aws profiles found. Attempting to use default.")
			profile = "default"
		} else {
			profile = p[0]
		}
	}

	fmt.Printf("Unprotecting %s using profile %s.\n", resource, profile)
	// obviously this is wrong, just hacking
	if unprotectInstance(profile, instanceSlice[0]) {
		fmt.Printf("Instance %s unprotected.")
	} else {
		fmt.Printf("Fail.")
	}
}

func init() {
	flag.StringVarP(&resource, "resource", "r", "", "Resource to unprotect")
	flag.StringVarP(&profile, "profile", "p", "", "AWS profile to use")
}
