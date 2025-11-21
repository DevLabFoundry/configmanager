# Examples

<!--
```go
rawStr := `somePAss: AWSPARAMSTR:///int-test/pocketbase/admin-pwd`
cm := configmanager.ConfigManager{}
// use custom token separator
// inline with v2 coming changes
cnf := generator.NewConfig().WithTokenSeparator("://")
// replaced will be a string which needs unmarshalling
replaced, err := cm.RetrieveWithInputReplaced(rawStr, *cnf)
```

Alternatively you can use the helper methods for Yaml or Json tagged structs - see [examples](./examples/examples.go) for more details

- `/path/to/parameter` the actual path to the secret or parameter in the target system e.g. AWS SecretsManager or ParameterStore (it does assume a path like pattern might throw a runtime error if not found)

If contents of the `AWSSECRETS#/appxyz/service1-password` are a string then `service1-password` will be the key and converted to UPPERCASE e.g. `SERVICE1_PASSWORD=som3V4lue` -->

## Working with environemt variables



## Working with files

Then you can access the single values like this `AWSSECRETS#/appxyz/service1-db-config|host` ==> `export SERVICE1_DB_CONFIG__HOST='db.internal'`

Alternatively if you are `configmanager`-ing a file via the fromstr command and the input is something like this:

(YAML)

```yaml
app:
  name: xyz
db:
  host: AWSSECRETS#/appxyz/service1-db-config|host
  port: AWSSECRETS#/appxyz/service1-db-config|port
  pass: AWSSECRETS#/appxyz/service1-db-config|pass
```

which would result in this

```yaml
app:
  name: xyz
db:
  host: db.internal
  port: 3306
  pass: sUp3$ecr3T!
```

If your config parameter matches the config interface, you can also leave the entire token to point to the `db` key

```yaml
app:
  name: xyz
db: AWSSECRETS#/appxyz/service1-db-config
```

result:

```yaml
app:
  name: xyz
db: {
  "host": "db.internal",
  "port": 3306,
  "pass": "sUp3$ecr3T!",
}
```

## Go API Examples

One of the sample use cases includes implementation in a K8s controller.

E.g. your Custom CRD stores some values in plain text that should really be secrets/nonpublic config parameters - something like this can be invoked from inside the controller code using the generator pkg API.

See [examples](./examples/examples.go) for more examples and tests for sample input/usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/DevLabFoundry/configmanager/v3/pkg/generator"
	"github.com/DevLabFoundry/configmanager/v3"
)

func main() {
	cm := &configmanager.ConfigManager{}
	cnf := generator.NewConfig()
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
	pm, err := cm.RetrieveWithInputReplaced(exampleK8sCrdMarshalled, *cnf)

	if err != nil {
		panic(err)
	}
	fmt.Println(pm)
}
```

Above example would ensure that you can safely store config/secret values on a CRD in plain text.

Or using go1.19+ [generics example](https://github.com/dnitsch/reststrategy/blob/d14ccec2b29bff646678ab9cf1775c0e93308569/controller/controller.go#L353).

> Beware logging out the CRD after tokens have been replaced.

Samlpe call to retrieve from inside an app/serverless function to only grab the relevant values from config.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/DevLabFoundry/configmanager/v3"
	"github.com/DevLabFoundry/configmanager/v3/pkg/generator"
)

var (
	DB_CONNECTION_STRING    string = "someuser:%v@tcp(%s:3306)/someschema"
	DB_PASSWORD_SECRET_PATH string = os.Getenv("DB_PASSWORD_TOKEN")
	DB_HOST_URL             string = os.Getenv("DB_URL_TOKEN")
)

func main() {
	connString, err := credentialString(context.TODO, DB_PASSWORD_SECRET_PATH, DB_HOST_URL)
	if err != nil {
		log.Fatal(err)
	}

}

func credentialString(ctx context.Context, pwdToken, hostToken string) (string, error) {

	cnf := generator.NewConfig()

	pm, err := configmanager.Retrieve([]string{pwdToken, hostToken}, *cnf)

	if err != nil {
		return "", err
	}
	if pwd, ok := pm[pwdToken]; ok {
		if host, ok := pm[hostToken]; ok {
			return fmt.Sprintf(DB_CONNECTION_STRING, pwd, host), nil
		}
	}

	return "", fmt.Errorf("unable to find value via token")
}
```
