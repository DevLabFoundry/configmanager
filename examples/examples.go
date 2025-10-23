package examples

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DevLabFoundry/configmanager/v3"
)

const DO_STUFF_WITH_VALS_HERE = "connstring:user@%v:host=%s/someschema..."

// retrieveExample uses the standard Retrieve method on the API
// this will return generator.ParsedMap which can be later used for more complex use cases
func retrieveExample() {
	cm := configmanager.New(context.TODO())
	cm.Config.WithTokenSeparator("://")

	pm, err := cm.Retrieve([]string{"token1", "token2"})

	if err != nil {
		panic(err)
	}

	// put in a loop for many config params
	// or use the helper methods to return a yaml replaced struct
	//
	if pwd, ok := pm["token1"]; ok {
		if host, ok := pm["token2"]; ok {
			fmt.Println(fmt.Sprintf(DO_STUFF_WITH_VALS_HERE, pwd, fmt.Sprintf("%s", host)))
		}
	}
}

// retrieveStringOut accepts a string as an input
func retrieveStringOut() {
	cm := configmanager.New(context.TODO())
	// JSON Marshal K8s CRD into
	exampleK8sCrdMarshalled := `apiVersion: crd.foo.custom/v1alpha1
kind: CustomFooCrd
metadata:
	name: foo
	namespace: bar
spec:
	name: baz
	secret_val: AWSSECRETS#/customfoo/secret-val
	owner: test_10016@example.com
`
	pm, err := cm.RetrieveWithInputReplaced(exampleK8sCrdMarshalled)

	if err != nil {
		panic(err)
	}
	fmt.Println(pm)
}

// ConfigTokenReplace uses configmanager to replace all occurences of
// replaceable tokens inside a []byte
// this is a re-useable method on all controllers
// will just ignore any non specs without tokens
func SpecConfigTokenReplace[T any](inputType T) (*T, error) {
	outType := new(T)
	rawBytes, err := json.Marshal(inputType)
	if err != nil {
		return nil, err
	}

	cm := configmanager.New(context.TODO())
	// use custom token separator
	cm.Config.WithTokenSeparator("://")

	replaced, err := cm.RetrieveWithInputReplaced(string(rawBytes))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(replaced), outType); err != nil {
		return nil, err
	}
	return outType, nil
}

// Example
func exampleRetrieveYamlUnmarshalled() {

	type config struct {
		DbHost   string `yaml:"dbhost"`
		Username string `yaml:"user"`
		Password string `yaml:"pass"`
	}
	configMarshalled := `
user: AWSPARAMSTR:///int-test/pocketbase/config|user
pass: AWSPARAMSTR:///int-test/pocketbase/config|pwd
dbhost: AWSPARAMSTR:///int-test/pocketbase/config|host
`

	appConf := &config{}
	cm := configmanager.New(context.TODO())
	// use custom token separator inline with future releases
	cm.Config.WithTokenSeparator("://")
	err := cm.RetrieveUnmarshalledFromYaml([]byte(configMarshalled), appConf)
	if err != nil {
		panic(err)
	}
	fmt.Println(appConf.DbHost)
	fmt.Println(appConf.Username)
	fmt.Println(appConf.Password)
}

// ### exampleRetrieveYamlMarshalled
func exampleRetrieveYamlMarshalled() {
	type config struct {
		DbHost   string `yaml:"dbhost"`
		Username string `yaml:"user"`
		Password string `yaml:"pass"`
	}

	appConf := &config{
		DbHost:   "AWSPARAMSTR:///int-test/pocketbase/config|host",
		Username: "AWSPARAMSTR:///int-test/pocketbase/config|user",
		Password: "AWSPARAMSTR:///int-test/pocketbase/config|pwd",
	}

	cm := configmanager.New(context.TODO())
	cm.Config.WithTokenSeparator("://")
	err := cm.RetrieveMarshalledYaml(appConf)
	if err != nil {
		panic(err)
	}
	if appConf.DbHost == "AWSPARAMSTR:///int-test/pocketbase/config|host" {
		panic(fmt.Errorf("value of DbHost should have been replaced with a value from token"))
	}
	fmt.Println(appConf.DbHost)
	fmt.Println(appConf.Username)
	fmt.Println(appConf.Password)
}
