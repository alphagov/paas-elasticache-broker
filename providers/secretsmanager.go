package providers

import "github.com/aws/aws-sdk-go/service/secretsmanager"

// SecretsManager is a partially extracted interface from the AWS Elasticache SDK
//
//go:generate counterfeiter -o mocks/secretsmanager.go . SecretsManager
type SecretsManager interface {
	CreateSecret(input *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error)
	GetSecretValue(input *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error)
	DeleteSecret(input *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error)
}
