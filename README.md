# paas-elasticache-broker

A CloudFoundry service broker for AWS Elasticache services. Currently only Redis is supported.

## Installation for testing locally

- install the broker code
  ```
  go get github.com/alphagov/paas-elasticache-broker
  ```
- Create an ElastiCache subnet group from e.g. the AWS web console.
  This group must have at least one subnet for your Redis instances to use.
- Create an EC2 security group with TCP ingress for port 6379 with the
  CIDR of any subnet in the subnet group configured above.
- Create an IAM role or user with necessary permissions; see [example policy](#example-iam-policy)
- Copy the example config from the blackbox tests
  ```
  cd $GOPATH/src/github.com/alphagov/paas-elasticache-broker
  cp ci/blackbox/config.json my-config.json
  ```
- Edit the config's `cache_subnet_group_name` to be the name of the subnet
  group created earlier.
- Edit the config's `vpc_security_group_ids` to be a single-item list
  containing the security group ID created above.

## Running the broker locally:

```
# replace these env vars with those created during installation
AWS_VPC_ID=vpc-deadbeef AWS_SUBNET_CIDR_BLOCK=10.0.16.0/24 \
  go run main.go -config my-config.json
```

## Broker configuration

You have to pass in a configuration JSON file with the following format:

```
{
  "broker_name": "Broker name",
  "username": "Broker http auth username",
  "password": "Broker http auth password",
  "auth_token_seed": "common auth token seed (secret)",
  "region": "<%= p('elasticache-broker.region') %>",
  "cache_subnet_group_name": "AWS Elasticache cache subnet group name",
  "vpc_security_group_ids": "List of AWS security group ids",
  "catalog": <Broker catalog JSON>,
  "plan_configs": <Plan config JSON>,
  "log_level": "Logging level, valid values are: DEBUG, INFO, ERROR, FATAL",
  "kms_key_id": KMS key used for storing generated auth tokens in the AWS Secrets Manager service
}
```

Broker catalog example:

```
{
  "services": [
    {
      "id": "d235edcf-8790-444a-b6e1-35e3c91a82c0",
      "name": "redis",
      "description": "AWS ElastiCache Redis service",
      "plan_updateable": true,
      "bindable": true,
      "plans": [
        {
          "id": "94767b71-2b9c-4960-a4f8-77b81a96f7e0",
          "name": "micro",
          "description": "Micro plan",
          "free": false
        }
      ]
    }
  ]
}
```

Plan config example:

```
{
  "94767b71-2b9c-4960-a4f8-77b81a96f7e0": {
    "instance_type": "cache.t2.micro",
    "replicas_per_node_group": 0,
    "shard_count": 1,
    "snapshot_retention_limit": 0,
    "automatic_failover_enabled": false,
    "parameters": {
      "maxmemory-policy": "volatile-lru",
      "reserved-memory": "0"
    }
  }
}
```

The plan config keys should be the same as the service plan ids.

The relevant structs can be found in the [config.go](broker/config.go) file.
The broker catalog structs can be found in the [pivotal-cf/brokerapi](https://github.com/pivotal-cf/brokerapi/blob/master/catalog.go) project.

## Client credentials

When binding a service instance to an application the *Bind* call returns the following client credentials:

```
{
  "host": "redis-host",
  "port": 6379,
  "password": "pass",
  "uri": "rediss://x:pass@redis-host:6379",
  "tls_enabled": true
}
```

Using TLS is mandatory in the clients.

The password will be the same for all bindings as the ElastiCache Redis replication group has only one password which can't be changed after the instance is created. This also means we are not able to revoke the access when an application is unbound.

## Example IAM policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "elasticache:CreateReplicationGroup",
        "elasticache:DescribeReplicationGroups",
        "elasticache:DeleteReplicationGroup",
        "elasticache:CreateCacheParameterGroup",
        "elasticache:ModifyCacheParameterGroup",
        "elasticache:DeleteCacheParameterGroup"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
```
