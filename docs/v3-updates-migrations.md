# V3 Changes and updates

As part of the V3 update we are aiming to streamline and improve the following high level areas. 

- Network call optimisation
- Backing store plugin architecture

## Network Call optimisation

There are many cases when an input string/file or array of `--token` can point to the same underlying token. 

e.g. 
```yaml
db_user: AWSSECRETS:///app1/db|user
db_password: AWSSECRETS:///app1/db|password
db_port: AWSSECRETS:///app1/db|port
db_host: AWSSECRETS:///app1/db|host
```

Given the above input passed into the CLI i.e. `configmanager fromstr -i above-config.yaml`

This would result in 4 network calls to the underlying service, in this case the AWS Secrets Manager. 

The V3 update would fan in these 4 tokens into a single network call and then fan out back to a full map with the individual values for each of the look up keys.

> NB: any token using a metadata annotation on any token would guarantee a unique call to the underlying service

e.g.:

```yaml
db_user: AWSSECRETS:///app1/db|user
db_password: AWSSECRETS:///app1/db|password
db_port: AWSSECRETS:///app1/db|port
db_host: AWSSECRETS:///app1/db|host
db_host_2: AWSSECRETS:///app1/db|host[version=2]
```

Even though `AWSSECRETS:///app1/db|host[version=2]` and `AWSSECRETS:///app1/db|host` are technically the same AWS Secrets Manager item, specifying the version requires two separate network calls.

## Backing Store plugin architecture

The current implementation of the backing stores is defined entirely within the configmanager source code which becomes part of the final staticly linked binary. In order to avoid the bigger size binary and **more importantly** avoid security alerts for libraries that are nothing to do with a backing store provider which is not used!

Most probably, and most commonly, one would only use single or a combination of providers within the same Cloud for example.

### Plugin Architecture

There are a few options to choose from - terraform style provider plugins using gRPC.

