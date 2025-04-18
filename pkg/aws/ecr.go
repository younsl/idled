package aws

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/younsl/idled/internal/models"
)

const (
	defaultECRIdleDays = 90
)

// ECRClient wraps the ECR API calls
type ECRClient struct {
	client *ecr.Client
	region string
}

// NewECRClient creates a new ECR client for the specified region
func NewECRClient(region string) (*ECRClient, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for region %s: %w", region, err)
	}
	return &ECRClient{
		client: ecr.NewFromConfig(cfg),
		region: region,
	}, nil
}

// GetIdleRepositories retrieves ECR repositories and identifies idle ones based on last push time
func (c *ECRClient) GetIdleRepositories() ([]models.RepositoryInfo, error) {
	var idleRepos []models.RepositoryInfo
	paginator := ecr.NewDescribeRepositoriesPaginator(c.client, &ecr.DescribeRepositoriesInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to describe ECR repositories in region %s: %w", c.region, err)
		}

		for _, repo := range output.Repositories {
			lastPush, imageCount, err := c.getLastPushTimeAndCount(repo.RepositoryName)
			if err != nil {
				// Log or handle error, maybe mark as potentially idle or skip
				fmt.Printf("Warning: Could not get image details for %s in %s: %v\n", *repo.RepositoryName, c.region, err)
			}

			idle := isECRRepositoryIdle(lastPush)

			// Optionally filter to only return idle ones, or return all with Idle flag
			// Currently returning all
			idleRepos = append(idleRepos, models.RepositoryInfo{
				Name:       aws.ToString(repo.RepositoryName),
				Region:     c.region,
				ARN:        aws.ToString(repo.RepositoryArn),
				URI:        aws.ToString(repo.RepositoryUri),
				LastPush:   lastPush,
				CreatedAt:  repo.CreatedAt,
				Idle:       idle,
				ImageCount: imageCount,
			})
		}
	}

	return idleRepos, nil
}

// getLastPushTimeAndCount finds the most recent image push time and total image count for a repository
func (c *ECRClient) getLastPushTimeAndCount(repoName *string) (*time.Time, int, error) {
	input := &ecr.DescribeImagesInput{
		RepositoryName: repoName,
	}
	imagePaginator := ecr.NewDescribeImagesPaginator(c.client, input)

	var latestPush *time.Time
	imageCount := 0

	for imagePaginator.HasMorePages() {
		page, err := imagePaginator.NextPage(context.TODO())
		if err != nil {
			// Handle errors, e.g., repository contains no images
			if _, ok := err.(*types.ImageNotFoundException); ok {
				return nil, 0, nil // No images found, so no last push time and count is 0
			} else if _, ok := err.(*types.RepositoryNotFoundException); ok {
				return nil, 0, fmt.Errorf("repository not found during image description: %w", err)
			}
			return nil, 0, fmt.Errorf("failed to describe images for repository %s: %w", *repoName, err)
		}

		imageCount += len(page.ImageDetails) // Add count from current page

		// Sort images by push time descending (only needed for last push time)
		sort.Slice(page.ImageDetails, func(i, j int) bool {
			if page.ImageDetails[i].ImagePushedAt == nil {
				return false
			}
			if page.ImageDetails[j].ImagePushedAt == nil {
				return true
			}
			return page.ImageDetails[i].ImagePushedAt.After(*page.ImageDetails[j].ImagePushedAt)
		})

		if len(page.ImageDetails) > 0 && page.ImageDetails[0].ImagePushedAt != nil {
			currentPageLatest := page.ImageDetails[0].ImagePushedAt
			if latestPush == nil || currentPageLatest.After(*latestPush) {
				latestPush = currentPageLatest
			}
		}
	}

	return latestPush, imageCount, nil
}

// isECRRepositoryIdle determines if a repository is idle based on the last push time
func isECRRepositoryIdle(lastPush *time.Time) bool {
	if lastPush == nil {
		// Consider repositories never pushed to as idle, or based on creation date?
		// For now, treating as idle if never pushed.
		return true
	}
	idleThreshold := time.Now().AddDate(0, 0, -defaultECRIdleDays)
	return lastPush.Before(idleThreshold)
}
