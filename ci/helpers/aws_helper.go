package helpers

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/sts"
	uuid "github.com/satori/go.uuid"
)

func CreateSubnetGroup(prefix string, session *session.Session) (*string, error) {
	vpcID, err := getVPCID(session)
	if err != nil {
		return nil, err
	}
	ec2Service := ec2.New(session)
	subnets, err := ec2Service.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(vpcID)},
		}},
	})
	if err != nil {
		return nil, err
	}

	subnetIDs := make([]*string, len(subnets.Subnets))
	for i, subnet := range subnets.Subnets {
		subnetIDs[i] = subnet.SubnetId
	}

	service := elasticache.New(session)
	subnetGroup, err := service.CreateCacheSubnetGroup(&elasticache.CreateCacheSubnetGroupInput{
		CacheSubnetGroupName:        aws.String(fmt.Sprintf("%s-%s", prefix, uuid.NewV4().String())),
		CacheSubnetGroupDescription: aws.String(fmt.Sprintf("%s integration tests", prefix)),
		SubnetIds:                   subnetIDs,
	})
	if err != nil {
		return nil, err
	}

	return subnetGroup.CacheSubnetGroup.CacheSubnetGroupName, nil
}

func CreateSecurityGroup(prefix string, session *session.Session) (*string, error) {
	vpcID, err := getVPCID(session)
	if err != nil {
		return nil, err
	}

	localSubnet := os.Getenv("AWS_SUBNET_CIDR_BLOCK")
	if localSubnet == "" {
		localSubnet, err = getNetworkMetadata("subnet-ipv4-cidr-block", session)
		if err != nil {
			return nil, err
		}
	}

	ec2Service := ec2.New(session)
	securityGroup, err := ec2Service.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(fmt.Sprintf("%s-%s", prefix, uuid.NewV4().String())),
		Description: aws.String(fmt.Sprintf("%s integration tests", prefix)),
		VpcId:       aws.String(vpcID),
	})
	if err != nil {
		return nil, err
	}

	for _, port := range []int64{6379} {
		_, err = ec2Service.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: securityGroup.GroupId,
			IpPermissions: []*ec2.IpPermission{{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int64(port),
				ToPort:     aws.Int64(port),
				IpRanges:   []*ec2.IpRange{{CidrIp: aws.String(localSubnet)}},
			}},
		})
		if err != nil {
			return nil, err
		}
	}

	return securityGroup.GroupId, nil
}

func DestroySubnetGroup(name *string, session *session.Session) error {
	service := elasticache.New(session)
	_, err := service.DeleteCacheSubnetGroup(&elasticache.DeleteCacheSubnetGroupInput{
		CacheSubnetGroupName: name,
	})

	return err
}

func DestroyParameterGroup(name *string, session *session.Session) error {
	service := elasticache.New(session)
	_, err := service.DeleteCacheParameterGroup(&elasticache.DeleteCacheParameterGroupInput{
		CacheParameterGroupName: name,
	})

	return err
}

func DestroySecurityGroup(id *string, session *session.Session) error {
	ec2Service := ec2.New(session)
	_, err := ec2Service.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: id,
	})

	return err
}

func ReplicationGroupARN(session *session.Session, replicationGroupID string) (string, error) {
	stssvc := sts.New(session)
	getCallerIdentityOutput, err := stssvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	awsPartition := "aws"
	awsRegion := aws.StringValue(session.Config.Region)
	awsAccountID := aws.StringValue(getCallerIdentityOutput.Account)

	return fmt.Sprintf(
		"arn:%s:elasticache:%s:%s:cluster:%s-0001-001",
		awsPartition,
		awsRegion,
		awsAccountID,
		replicationGroupID,
	), nil
}

func getVPCID(session *session.Session) (string, error) {
	vpcID := os.Getenv("AWS_VPC_ID")
	if vpcID == "" {
		var err error
		vpcID, err = getNetworkMetadata("vpc-id", session)
		if err != nil {
			return "", err
		}
	}
	return vpcID, nil
}

func getNetworkMetadata(name string, session *session.Session) (string, error) {
	const prefix = "network/interfaces/macs"

	metaService := ec2metadata.New(session)
	// FIXME: What if there is more than one MAC?
	mac, err := metaService.GetMetadata(prefix)
	if err != nil {
		return "", err
	}

	return metaService.GetMetadata(strings.Join([]string{prefix, mac, name}, "/"))
}
