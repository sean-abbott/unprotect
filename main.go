package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/manifoldco/promptui"
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

type TerraformInstanceResult struct {
	Instances map[string]ResourceInstance
	Error     error
}

// end my structs

// helper functions
func resourceInState(a string, list map[string]ResourceInstance) bool {
	for k := range list {
		if k == a {
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

	_, err := e.ModifyInstanceAttribute(input)
	if err != nil {
		fmt.Println(err.Error())
	}

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

func getTerraformState() (*TerraformState, error) {
	var (
		cmdOut []byte
		err    error
	)

	cmd := "terraform"
	args := []string{"state", "pull"}
	if cmdOut, err = exec.Command(cmd, args...).Output(); err != nil {
		fmt.Printf("'terraform state pull' failed. Are you in a terraform directory?")
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	if len(cmdOut) == 0 {
		fmt.Printf("terraform state pull did not return anything. Are you sure you're in a terraform module?\n")
		return nil, errors.New("Terraform state pull empty.")
	}

	terraformState := &TerraformState{}

	if err := json.Unmarshal(cmdOut, terraformState); err != nil {
		fmt.Printf("Failed to unmarshall terraform state pull output.\n")
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	return terraformState, nil

}

func getInstanceMap(terraformState *TerraformState) map[string]ResourceInstance {
	instances := map[string]ResourceInstance{}

	for i, module := range terraformState.Modules {
		for key, k := range module.Resources {
			if strings.HasPrefix(key, "aws_instance") {
				fmt.Printf("Module %d has an aws_instance.\n", i)
				id := k.(map[string]interface{})["primary"].(map[string]interface{})["id"].(string)
				r := ResourceInstance{Resource: key, Id: id}
				instances[key] = r
			}
		}
	}

	return instances
}

func getAwsProfile(c chan string) {
	if profile == "" {
		var p []string
		files, err := ioutil.ReadDir(".")
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			if filepath.Ext(file.Name()) == ".tf" {
				p = append(p, getProfilesFromFile(file.Name())...)
			}
		}

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
	c <- profile
}

func getInstances(ch chan TerraformInstanceResult) {
	fmt.Printf("Getting terraform state...")
	terraformState, err := getTerraformState()
	if err != nil {
		ch <- TerraformInstanceResult{nil, err}
	}
	instanceMap := getInstanceMap(terraformState)
	ch <- TerraformInstanceResult{instanceMap, nil}
}

func promptForInstance(instances map[string]ResourceInstance) string {
	var nameSlice []string
	for k := range instances {
		nameSlice = append(nameSlice, k)
	}

	prompt := promptui.Select{
		Label: "Select Instance",
		Items: nameSlice,
	}

	_, result, err := prompt.Run()

	if err != nil {
		log.Fatal("Prompt failed %v\n", err)
	}

	return result
}

func validateInstance(t TerraformInstanceResult) ResourceInstance {
	if t.Error != nil {
		fmt.Printf("Something went wrong getting the terraform state.\n")
		fmt.Println(t.Error)
		os.Exit(1)
	}

	instanceMap := t.Instances

	if resource == "" {
		resource = promptForInstance(instanceMap)
	}

	if !resourceInState(resource, instanceMap) {
		fmt.Printf("Could not find resource %s. Instances available: %v\n", resource, instanceMap)
	}

	return instanceMap[resource]
}

func main() {
	flag.Parse()

	// run terraform to get the state
	c := make(chan TerraformInstanceResult)
	go getInstances(c)

	// grab any profiles from the local terraform files
	p := make(chan string)
	go getAwsProfile(p)

	instanceResult := <-c
	instance := validateInstance(instanceResult)

	profile = <-p
	fmt.Printf("Unprotecting %s using profile %s.\n", instance.Resource, profile)

	if unprotectInstance(profile, instance) {
		fmt.Printf("Instance %s unprotected.\n", instance.Resource)
	} else {
		fmt.Printf("Failed to disable termination protection for %s.", instance.Resource)
	}
}

func init() {
	flag.StringVarP(&resource, "resource", "r", "", "Resource to unprotect")
	flag.StringVarP(&profile, "profile", "p", "", "AWS profile to use")
}
