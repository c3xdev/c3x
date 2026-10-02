package aws

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

// Identity is what the default AWS configuration resolves to.
type Identity struct {
	// Region is the default region, which may be empty: each resource
	// carries its own, and --region is only a fallback.
	Region string
	// CredentialSource says where the credentials come from (environment,
	// shared config, SSO, ...), never the credentials themselves.
	CredentialSource string
}

// Check resolves the default configuration and retrieves credentials, so
// `c3x doctor` can say whether usage sync could run. It calls no AWS
// service.
func Check(ctx context.Context) (Identity, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return Identity{}, fmt.Errorf("load AWS configuration: %w", err)
	}
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return Identity{}, fmt.Errorf("no AWS credentials: %w", err)
	}
	return Identity{Region: cfg.Region, CredentialSource: creds.Source}, nil
}
