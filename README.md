# Config Manager

[![Go Reference](https://pkg.go.dev/badge/github.com/DevLabFoundry/configmanager/v2.svg)](https://pkg.go.dev/github.com/DevLabFoundry/configmanager/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/DevLabFoundry/configmanager/v2)](https://goreportcard.com/report/github.com/DevLabFoundry/configmanager/v2)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=dnitsch_configmanager&metric=bugs)](https://sonarcloud.io/summary/new_code?id=dnitsch_configmanager)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=dnitsch_configmanager&metric=sqale_index)](https://sonarcloud.io/summary/new_code?id=dnitsch_configmanager)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=dnitsch_configmanager&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=dnitsch_configmanager)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=dnitsch_configmanager&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=dnitsch_configmanager)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=dnitsch_configmanager&metric=coverage)](https://sonarcloud.io/summary/new_code?id=dnitsch_configmanager)

Package used for retrieving application settings from various sources.

Currently supported variable and secrets implementations:
<!--
"AWSSECRETS"
	// AWS Parameter Store prefix
	ParamStorePrefix ImplementationPrefix = "AWSPARAMSTR"
	// Azure Key Vault Secrets prefix
	AzKeyVaultSecretsPrefix ImplementationPrefix = "AZKVSECRET"
	// Hashicorp Vault prefix
	HashicorpVaultPrefix ImplementationPrefix = "VAULT"
	// GcpSecrets
	GcpSecretsPrefix ImplementationPrefix = "GCPSECRETS" -->

- [AWS SecretsManager](https://aws.amazon.com/secrets-manager/)
	- Implementation Indicator: `AWSSECRETS`
- [AWS ParameterStore](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html)
	- Implementation Indicator: `AWSPARAMSTR`
- [AzureKeyvault Secrets](https://azure.microsoft.com/en-gb/products/key-vault/)
	- Implementation Indicator: `AZKVSECRET`
	- see [Special consideration for AZKVSECRET](#special-consideration-for-azkvsecret) around how to structure the token in this case.
- [Azure TableStorage](https://azure.microsoft.com/en-gb/products/storage/tables/)
	- Implementation Indicator: `AZTABLESTORE`
	- see [Special consideration for AZTABLESTORE](#special-consideration-for-aztablestore) around how to structure the token in this case.
- [Azure AppConfig](https://azure.microsoft.com/en-gb/products/app-configuration/)
	- Implementation Indicator: `AZAPPCONF`
- [GCP Secrets](https://cloud.google.com/secret-manager)
	- Implementation Indicator: `GCPSECRETS`
- [Hashicorp Vault](https://developer.hashicorp.com/vault/docs/secrets/kv)
	- Implementation Indicator: `VAULT`
	- using the KvV2 engine endpoint
	- see [special consideration hashivault](#special-consideration-for-hashicorpvault)

The main driver is to use component level configuration objects, if stored in a `"namespaced"` manner e.g. in AWS ParamStore as `/nonprod/component-service-a/configVar`, however this is not a requirement and the param name can be whatever. Though whilst using some sort of a organised manner it will be more straight forward to allow other services to consume certain secrets/params based on resource/access policies.

> Beware size limitation with certain config/vault implementations. In which case it's best to split certain items up e.g. TLS certs `/nonprod/component-service-a/pub-cert`, `/nonprod/component-service-a/private-cert`, `/nonprod/component-service-a/chain1-cert`, etc...

Where `configVar` can be either a parseable string `'som3#!S$CRet'` or a number `3306` or a parseable single level JSON object like `{host: ..., pass: ...., port: ...}` which can be returned whole or accessed via a key separator for a specific value.

## Use cases

- Go API

   This can be leveraged from any application written in Go - on start up or at runtime. Secrets/Config items can be retrieved in "bulk" and parsed into a provided type, [see here for examples](./examples/examples.go).

   > BREAKING CHANGE v2.x with the API (see [examples](./examples/examples.go))
  - `generator.NewConfig()` is no longer required.

    ```go
    // initialise new configmanager instance
    cm := configmanager.New(context.TODO())
    // add additional config to apply on your tokens
    cm.Config.WithTokenSeparator("://")
    pm, err := cm.Retrieve([]string{"IMPLEMENTATION://token1", "IMPLEMENTATION://token2", "ANOTHER_IMPL://token1"})
    ```

  - `RetrieveUnmarshalledFromYaml`|`RetrieveUnmarshalledFromJson`|`RetrieveMarshalledJson`|`RetrieveMarshalledYaml` methods are now on the ConfigManager struct, see `exampleRetrieveYamlMarshalled` or `exampleRetrieveYamlUnmarshalled` in [examples](./examples/examples.go)

- Kubernetes

   Avoid storing overly large configmaps and especially using secrets objects to store actual secrets e.g. DB passwords, 3rd party API creds, etc... By only storing a config file or a script containing only the tokens e.g. `AWSSECRETS:///$ENV/service/db-config` it can be git committed without writing numerous shell scripts, only storing either some interpolation vars like `$ENV` in a configmap or the entire configmanager token for smaller use cases.

- VMs

   VM deployments can function in a similar manner by passing in the contents or a path to the source config and the output path so that app at startup time can consume it.

## CLI

ConfigManager comes packaged as a CLI for all major platforms, to see [download/installation](./docs/installation.md)

For more detailed usage you can run -h with each subcommand and additional info can be found [here](./docs/commands.md)

## __Token Config__

The token is made up of the following parts:

_An example token would look like this_

#### `AWSSECRETS:///path/to/my/key|lookup.Inside.Object[meta=data]`

### Implementation indicator

The `AWSSECRETS` the strategy identifier to choose the correct provider at runtime. Multiple providers can be referenced in a single run via a CLI or with the API.

This is not overrideable and must be exactly as it is in the provided list of providers.

### __Token Separator__

The `#` symbol from the [example token](#awssecretspathtomykeylookupinsideobjectmetadata) - used for separating the implementation indicator and the look up value.

> The default is currently `#` - it will change to `://` to allow for a more natural reading of the "token". you can achieve this behaviour now by either specifying the `-s` to the CLI or ConfigManager Go API.

```go
cnf := generator.NewConfig().WithTokenSeparator("://")
```

### __Provider Secret/Config Path__

The `/path/to/my/key` part from the [example token](#awssecretspathtomykeylookupinsideobjectmetadata) is the actual path to the item in the backing store.

See the different special considerations per provider as it different providers will require different implementations.

### __Key Separator__

__THIS IS OPTIONAL__

The `|` symbol from the [example token](#awssecretspathtomykeylookupinsideobjectmetadata) is used to specify the key seperator.

If an item retrieved from a store is JSON parseable map it can be interrogated for further properties inside.

### __Look up key__

__THIS IS OPTIONAL__

The `lookup.Inside.Object` from the [example token](#awssecretspathtomykeylookupinsideobjectmetadata) is used to perform a lookup inside the retrieved item IF it is parseable into a `map[string]any` structure.

Given the below response from a backing store

```json
{
	"lookup": {
		"Inside": {
			"Object": {
				"host": "db.internal",
				"port": 3306,
				"pass": "sUp3$ecr3T!",
			}
		}
	}
}
```

The value returned for the [example token](#awssecretspathtomykeylookupinsideobjectmetadata)  would be:

```json
{
	"host": "db.internal",
	"port": 3306,
	"pass": "sUp3$ecr3T!",
}
```

See [examples of working with files](docs/examples.md#working-with-files) for more details.

### Token Metadata Config

The `[meta=data]` from the [example token](#awssecretspathtomykeylookupinsideobjectmetadata) - is the optional metadata about the target in the backing provider


> IT must have this format `[key=value]` - IT IS OPTIONAL
> IT must be specified last - either after a path lookup or if there is no key look up path specified then after the full path

The `key` and `value` would be provider specific. Meaning that different providers support different config, these values _CAN_ be safely omitted configmanager would just use the defaults where applicable or not specify the additional

- Hashicorp Vault (VAULT)
	- `iam_role` - would be the value of an IAM role ARN to use with AWSClient Authentication.
	- `version` - is the version of the secret/configitem to get (should be in an integer format)

	e.g. `VAULT://baz/bar/123|d88[role=arn:aws:iam::1111111:role/i-orchestration,version=1082313]`

- Azure AppConfig (AZAPPCONF)
	- `label` - the label to use whilst retrieving the item
	- `etag` - etag value

	e.g. `AZAPPCONF://baz/bar/123|d88[label=dev,etag=aaaaa1082313]`

- GCP secrets, AWS SEcrets, AZ KeyVault (`GCPSECRETS` , `AWSSECRETS`, `AZKVSECRET`)
	they all support the `version` metadata property

	e.g. `GCPSECRETS://baz/bar/123|d88[version=verUUID0000-1123zss]`

## Special considerations

This section outlines the special consideration in token construction on a per provider basis

### Special consideration for AZKVSECRET

For Azure KeyVault the first part of the token needs to be the name of the vault.

> Azure Go SDK (v2) requires the vault Uri on initializing the client

`AZKVSECRET#/test-vault//token/1` ==> will use KeyVault implementation to retrieve the `/token/1` from a `test-vault`.

`AZKVSECRET#/test-vault/no-slash-token-1` ==> will use KeyVault implementation to retrieve the `no-slash-token-1` from a `test-vault`.

> The preceeding slash to the vault name is optional - `AZKVSECRET#/test-vault/no-slash-token-1` and `AZKVSECRET#test-vault/no-slash-token-1` will both identify the vault of name `test-vault`

### Special consideration for AZTABLESTORE

The token itself must contain all of the following properties, so that it would look like this `AZTABLESTORE://STORAGE_ACCOUNT_NAME/TABLE_NAME/PARTITION_KEY/ROW_KEY`:

- Storage account name [`STORAGE_ACCOUNT_NAME`]
- Table Name [`TABLE_NAME`]
	- > It might make sense to make this table global to the domain or project
- Partition Key [`PARTITION_KEY`]
	- > This could correspond to the component/service name
- Row Key [`ROW_KEY`]
	- > This could correspond to the property itself or a group of properties
	- > e.g. `AZTABLESTORE://globalconfigstorageaccount/domainXyz/serviceXyz/db` => `{"value":{"host":"foo","port":1234,"enabled":true}}`
	- > It will continue to work the same way with additional keyseparators inside values.

> NOTE: if you store a more complex object inside a top level `value` property this will reduce the number of columns and normalize the table - **THE DATA INSIDE THE VALUE MUST BE JSON PARSEABLE**

All the usual token rules apply e.g. of `keySeparator`

`AZTABLESTORE://account/app1Config/db/config` => `{host: foo.bar, port: 8891}`

`AZTABLESTORE://account/app1Config/db/config|host` => `foo.bar`

### Special consideration for HashicorpVault

For HashicorpVault the first part of the token needs to be the name of the mountpath. In Dev Vaults this is `"secret"`,
 e.g.: `VAULT://secret___demo/configmanager|test`

or if the secrets are at another location: `VAULT://another/mount/path___config/app1/db`

The hardcoded separator cannot be modified and you must separate your `mountPath` with `___` (3x `_`) followed by the key to the secret.

#### AWS IAM auth to vault

when using Vault in AWS - you can set the value of the `VAULT_TOKEN=aws_iam` this will trigger the AWS Auth login as opposed to using the local token.

The Hashicorp Vault functions in the same exact way as the other implementations. It will retrieve the JSON object and can be looked up within it by using a key separator.

## [Go API](https://pkg.go.dev/github.com/DevLabFoundry/configmanager/v2)

## [Examples](docs/examples.md)

## Help

- More implementations should be easily added with a specific implementation under the strategy interface
    - see [add additional providers](docs/adding-provider.md)

- maybe run as cron in the background to perform a periodic sync in case values change?
