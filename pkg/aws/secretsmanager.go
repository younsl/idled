package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	// smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"github.com/younsl/idled/internal/models"
)

const (
	secretsManagerIdleDays = 90
)

// SecretsManagerScanner contains the AWS client needed for scanning Secrets Manager resources
type SecretsManagerScanner struct {
	Client *secretsmanager.Client
	Region string
}

// NewSecretsManagerScanner creates a new SecretsManagerScanner for a given region
func NewSecretsManagerScanner(cfg aws.Config) *SecretsManagerScanner {
	return &SecretsManagerScanner{
		Client: secretsmanager.NewFromConfig(cfg),
		Region: cfg.Region,
	}
}

// GetIdleSecrets scans all secrets in the region and identifies idle ones.
func (s *SecretsManagerScanner) GetIdleSecrets(ctx context.Context) ([]models.SecretInfo, []error) {
	var idleSecrets []models.SecretInfo
	var scanErrs []error

	// Use a paginator to list all secrets
	paginator := secretsmanager.NewListSecretsPaginator(s.Client, &secretsmanager.ListSecretsInput{})

	now := time.Now()
	pageCount := 0

	for paginator.HasMorePages() {
		pageCount++
		output, err := paginator.NextPage(ctx)
		if err != nil {
			scanErrs = append(scanErrs, fmt.Errorf("error listing secrets page %d in region %s: %w", pageCount, s.Region, err))
			break // Stop processing this region on pagination error
		}

		if output != nil {
			for _, secret := range output.SecretList {
				// Check if LastAccessedDate is available
				if secret.LastAccessedDate != nil {
					lastAccessed := aws.ToTime(secret.LastAccessedDate)
					idleDuration := now.Sub(lastAccessed)
					idleDays := int(idleDuration.Hours() / 24)

					if idleDays > secretsManagerIdleDays {
						idleSecrets = append(idleSecrets, models.SecretInfo{
							ARN:              aws.ToString(secret.ARN),
							Name:             aws.ToString(secret.Name),
							Region:           s.Region,
							LastAccessedDate: lastAccessed,
							IdleDays:         idleDays,
						})
					}
				} else {
					// Secret has never been accessed, consider it idle based on creation date?
					// For now, we only consider secrets with a LastAccessedDate.
					// Alternatively, could check CreationDate if LastAccessedDate is nil.
				}
			}
		}
	}

	return idleSecrets, scanErrs
}

// TODO: Define models.SecretInfo in internal/models/aws.go if it doesn't exist.
// Example structure:
// type SecretInfo struct {
// 	 ARN            string
// 	 Name           string
// 	 Region         string
// 	 LastAccessedDate time.Time
// 	 IdleDays       int
// }
