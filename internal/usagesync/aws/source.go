// Package aws is the AWS source for usage sync, and the only package that
// imports the AWS SDK. It makes read-only CloudWatch calls and turns the
// answers into the quantities the catalog reads. Credentials come only
// from the SDK's default chain (environment, shared config, SSO,
// instance or task role); nothing here handles a credential itself.
package aws

import (
	"context"
	"fmt"
	"sync"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"

	"github.com/c3xdev/c3x/internal/usagesync"
)

// CloudWatch is the one SDK call used. It is an interface so tests supply
// canned answers and CI needs no credentials.
type CloudWatch interface {
	GetMetricData(ctx context.Context, in *cloudwatch.GetMetricDataInput, opts ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}

// Source implements [usagesync.Source] for AWS.
type Source struct {
	clientFor func(region string) CloudWatch

	mu      sync.Mutex
	clients map[string]CloudWatch
}

// New builds a Source from the default AWS configuration. No call is made
// until a resource is collected.
func New(ctx context.Context) (*Source, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	return NewWithClients(func(region string) CloudWatch {
		return cloudwatch.NewFromConfig(cfg, func(o *cloudwatch.Options) { o.Region = region })
	}), nil
}

// NewWithClients builds a Source whose CloudWatch client for each region
// comes from clientFor. Used by tests.
func NewWithClients(clientFor func(region string) CloudWatch) *Source {
	return &Source{clientFor: clientFor, clients: map[string]CloudWatch{}}
}

// Provider implements [usagesync.Source].
func (s *Source) Provider() string { return "aws" }

// Emits implements [usagesync.Source].
func (s *Source) Emits() map[string][]string {
	return map[string][]string{
		kindS3Bucket: {keyStandardStorageGB},
	}
}

// Collect implements [usagesync.Source].
func (s *Source) Collect(ctx context.Context, t usagesync.Target, w usagesync.Window) (usagesync.Result, error) {
	switch t.Kind {
	case kindS3Bucket:
		return s.s3Storage(ctx, t, w)
	default:
		return usagesync.Result{}, fmt.Errorf("usage sync does not support %s", t.Kind)
	}
}

func (s *Source) client(region string) CloudWatch {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[region]
	if !ok {
		c = s.clientFor(region)
		s.clients[region] = c
	}
	return c
}
