# Adding a provider

Add Token Prefix

```go
const (
	// AWS SecretsManager prefix
	SecretMgrPrefix ImplementationPrefix = "AWSSECRETS"
	// AWS Parameter Store prefix
	ParamStorePrefix ImplementationPrefix = "AWSPARAMSTR"
	// Azure Key Vault Secrets prefix
	AzKeyVaultSecretsPrefix ImplementationPrefix = "AZKVSECRET"
	// Hashicorp Vault prefix
	HashicorpVaultPrefix ImplementationPrefix = "VAULT"
	// GcpSecrets
	GcpSecretsPrefix ImplementationPrefix = "GCPSECRETS"
)
```

```go
var (
	// default varPrefix used by the replacer function
	// any token must beging with one of these else
	// it will be skipped as not a replaceable token
	VarPrefix = map[ImplementationPrefix]bool{SecretMgrPrefix: true, ParamStorePrefix: true, AzKeyVaultSecretsPrefix: true, GcpSecretsPrefix: true, HashicorpVaultPrefix: true} // <-- ADD here
)
```

ensure your implementation satisfy the `genVarsStrategy` interface

```go
type genVarsStrategy interface {
	tokenVal(rs *retrieveStrategy) (s string, e error)
	setTokenVal(s string)
}
```

Even if the native type is K/V return a marshalled version of the JSON as the rest of the flow will decide how to present it back to the final consumer.

Custom properties inside the GetValue request, you could specify your own Config struct for the provider, e.g. HashiVault implementation 

```go
// VaultConfig holds the parseable metadata struct
type VaultConfig struct {
	Version string `json:"version"`
	Role    string `json:"iam_role"`
}
```

You could then use it on the backingStore object 

```go
type VaultStore struct {
	svc    hashiVaultApi
	ctx    context.Context
	config *VaultConfig
	token  string
}
```

On initialize of the instance or in the setTokenVal method (see GCPSecrets or AWSSecrets/ParamStore examples). 

```go
storeConf := &VaultConfig{}
initialToken := ParseMetadata(token, storeConf)
imp := &VaultStore{
	ctx:    ctx,
	config: storeConf,
}
```

Where the initialToken is the original Token without the metadata in brackets and the `storeConf` pointer will have been filled with any of the parsed metadata and used in the actual provider implementation, see any of the providers for a sample implementation.
