package aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/briandowns/spinner"
	"github.com/dustin/go-humanize"
	"github.com/younsl/idled/internal/models"
)

func getActualLastEventTimestamp(ctx context.Context, client *cloudwatchlogs.Client, logGroupName string) (int64, error) {
	filterInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(logGroupName),
		Limit:        aws.Int32(1),
		StartTime:    aws.Int64(0),
		EndTime:      aws.Int64(time.Now().UnixMilli()),
	}

	resp, err := client.FilterLogEvents(ctx, filterInput)
	if err != nil {
		var resourceNotFound *types.ResourceNotFoundException
		if errors.As(err, &resourceNotFound) {
			return 0, nil
		}
		return 0, fmt.Errorf("FilterLogEvents failed for %s: %w", logGroupName, err)
	}

	if len(resp.Events) > 0 && resp.Events[0].Timestamp != nil {
		return *resp.Events[0].Timestamp, nil
	}

	return 0, nil
}

func ScanLogGroups(cfg aws.Config, idleThresholdDays int) ([]models.LogGroupInfo, []error) {
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Scanning CloudWatch Log Groups ..."
	s.Start()

	client := cloudwatchlogs.NewFromConfig(cfg)
	var preliminaryGroups []types.LogGroup
	var fetchErrors []error
	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(client, &cloudwatchlogs.DescribeLogGroupsInput{})

	pageCount := 0
	for paginator.HasMorePages() {
		pageCount++
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			fetchErr := fmt.Errorf("error fetching log groups page %d: %w", pageCount, err)
			fetchErrors = append(fetchErrors, fetchErr)
			continue
		}
		preliminaryGroups = append(preliminaryGroups, output.LogGroups...)
	}

	var finalLogGroups []models.LogGroupInfo
	var checkErrors []error
	idleThresholdTime := time.Now().AddDate(0, 0, -idleThresholdDays).UnixMilli()

	for _, lg := range preliminaryGroups {
		retention := "Never expire"
		if lg.RetentionInDays != nil {
			retention = fmt.Sprintf("%d days", *lg.RetentionInDays)
		}

		creationTimestamp := int64(0)
		if lg.CreationTime != nil {
			creationTimestamp = *lg.CreationTime
		}

		actualLastEventTimestamp, err := getActualLastEventTimestamp(context.TODO(), client, aws.ToString(lg.LogGroupName))
		if err != nil {
			checkErrors = append(checkErrors, fmt.Errorf("failed check for %s: %w", aws.ToString(lg.LogGroupName), err))
		}

		var effectiveTimestamp int64
		var displayTimeStr string

		if actualLastEventTimestamp > 0 {
			effectiveTimestamp = actualLastEventTimestamp
			displayTimeStr = time.UnixMilli(effectiveTimestamp).Format("2006-01-02 15:04:05")
		} else if creationTimestamp > 0 {
			effectiveTimestamp = creationTimestamp
			displayTimeStr = fmt.Sprintf("N/A (Created: %s)", time.UnixMilli(creationTimestamp).Format("2006-01-02 15:04:05"))
		} else {
			effectiveTimestamp = 0
			displayTimeStr = "N/A"
		}

		if effectiveTimestamp > 0 && effectiveTimestamp < idleThresholdTime {
			info := models.LogGroupInfo{
				Name:            aws.ToString(lg.LogGroupName),
				RetentionDays:   retention,
				StoredBytes:     humanize.Bytes(uint64(aws.ToInt64(lg.StoredBytes))),
				LastEventTime:   displayTimeStr,
				ARN:             aws.ToString(lg.Arn),
				CreationTime:    time.UnixMilli(creationTimestamp),
				LastEventMillis: effectiveTimestamp,
			}
			finalLogGroups = append(finalLogGroups, info)
		}
	}

	allErrors := append(fetchErrors, checkErrors...)

	return finalLogGroups, allErrors
}
