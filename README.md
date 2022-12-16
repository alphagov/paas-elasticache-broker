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
- Create an IAM role or user with necessary permissions; see [required IAM permissions](#required-iam-permissions) below
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

## Patching an existing bosh environment

If you want to patch an existing bosh environment you can run the following command:

```
make bosh_scp
```

This requires an existing bosh session to be established beforehand.

## Broker configuration

You have to pass in a configuration JSON file with the following format:

```
{
  "broker_name": "Broker name",
  "username": "Broker http auth username",
  "password": "Broker http auth password",
  "region": "<%= p('elasticache-broker.region') %>",
  "cache_subnet_group_name": "AWS Elasticache cache subnet group name",
  "vpc_security_group_ids": "List of AWS security group ids",
  "catalog": <Broker catalog JSON>,
  "plan_configs": <Plan config JSON>,
  "log_level": "Logging level, valid values are: DEBUG, INFO, ERROR, FATAL",
  "kms_key_id": "KMS key used for storing generated auth tokens in the AWS Secrets Manager service",
  "secrets_manager_path": "The path prefix used for secrets stored in AWS Secrets Manager service"
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
      "instances_retrievable": true,
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

When binding a service instance to an application the _Bind_ call returns the following client credentials:

```
{
  "host": "redis-host",
  "port": 6379,
  "password": "pass",
  "uri": "rediss://:pass@redis-host:6379",
  "tls_enabled": true
}
```

Using TLS is mandatory in the clients.

The password will be the same for all bindings as the ElastiCache Redis replication group has only one password which can't be changed after the instance is created. This also means we are not able to revoke the access when an application is unbound.

## Required IAM permissions

The broker needs a number of AWS permissions to operate:

### Elasticache

It needs full access to elasticache. The easiest way achieve this is to use the
[AWS-provided AmazonElastiCacheFullAccess policy][amazonelasticachefullaccess_policy]
(arn: `arn:aws:iam::aws:policy/AmazonElastiCacheFullAccess`) which also allows
creating the necessary service-linked role.

[amazonelasticachefullaccess_policy]: https://docs.aws.amazon.com/AmazonElastiCache/latest/mem-ug/IAM.IdentityBasedPolicies.html#IAM.IdentityBasedPolicies.PredefinedPolicies

### Secrets Manager

The broker uses [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/)
to store the generated passwords for instances. It therefore needs the following permissions:

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:CreateSecret",
        "secretsmanager:DescribeSecret",
        "secretsmanager:GetSecretValue",
        "secretsmanager:DeleteSecret",
        "secretsmanager:List*"
      ],
      "Resource": [
        "arn:aws:secretsmanager:*:*:secret:<SECRETS_MANAGER_PATH>/*"
      ]
    }
  ]
}
```

Where `<SECRETS_MANAGER_PATH>` matches the path given in the config file.

It also needs access to the KMS key used with Secrets Manager:

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "kms:GenerateDataKey",
        "kms:Encrypt",
        "kms:Decrypt"
      ],
      "Resource": [
        "<KMS_KEY_ARN>"
      ]
    }
  ]
}
```

Where `<KMS_KEY_ARN>` is the arn of the key provided in the config file.

## Generating a cache cluster name from a CF service instance GUID

Elasticache cluster names are generated by hashing the service GUID. To make life easier, use the cache cluster name
generator tool in `cache-cluster-name-generator/`

```
go run cache-cluster-name-generator/main.go "$(cf service your-service-name --guid)"
```

## Generating fakes for ginkgo testing

If providers/{elasticache,provider,secretsmanager}.go are updated, the fakes will more than likely need to be updated.
These can be regenerated via `make generate-fakes`.
