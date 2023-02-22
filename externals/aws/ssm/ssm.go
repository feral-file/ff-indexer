package ssm

import (
	"context"
	"crypto/rsa"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/dgrijalva/jwt-go"
)

type SystemManager struct {
	client *ssm.Client
}

// New create new SystemManager
// config will load secret, region from aws configure
func New(ctx context.Context) (*SystemManager, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &SystemManager{
		client: ssm.NewFromConfig(cfg),
	}, nil
}

// FindParameter find parameter in AWS SSM parameter store
func (s *SystemManager) FindParameter(ctx context.Context, parameterName string) (*ssm.GetParameterOutput, error) {
	input := &ssm.GetParameterInput{
		Name: &parameterName,
	}

	parameter, err := s.client.GetParameter(ctx, input)
	if err != nil {
		return nil, err
	}

	return parameter, nil
}

// GetRSAPublishKeyFromParameterStore get RSA Publish Key from Parameter Store
func (s *SystemManager) GetRSAPublishKeyFromParameterStore(ctx context.Context, parameterName string) (*rsa.PublicKey, error) {
	parameter, err := s.FindParameter(ctx, parameterName)
	if err != nil {
		return nil, err
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(*parameter.Parameter.Value))
	if err != nil {
		return nil, err
	}

	return publicKey, nil
}
