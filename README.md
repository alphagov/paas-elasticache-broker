# paas-elasticache-broker

A CloudFoundry service broker for AWS Elasticache services. Currently only Redis is supported.

## Installation for testing locally

- install the broker code
  ```
  go get github.com/alphagov/paas-elasticache-broker
  ```
- Create an ElastiCache subnet group from e.g. the AWS web console.
  This group must have at least one subnet for your Redis instances to use.
- Create an EC2 security group with TCP ingress for ports 3306-5432 with the
  CIDR of any subnet in the subnet group configured above.
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
