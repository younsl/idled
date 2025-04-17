package utils

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// GetTagValue returns the value of a tag with the given key
func GetTagValue(tags []types.Tag, key string) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == key {
			if tag.Value != nil {
				return *tag.Value
			}
			return ""
		}
	}
	return ""
}

// GetName returns the value of the Name tag
func GetName(tags []types.Tag) string {
	return GetTagValue(tags, "Name")
}

// GetTagsMap converts a slice of tags to a map
func GetTagsMap(tags []types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}

// ConvertToEC2Tags converts a map of tags to a slice of EC2 tags
func ConvertToEC2Tags(tags map[string]string) []types.Tag {
	var result []types.Tag
	for k, v := range tags {
		key := k
		value := v
		result = append(result, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return result
}

// HasTag checks if a resource has a tag with the given key
func HasTag(tags []types.Tag, key string) bool {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == key {
			return true
		}
	}
	return false
}

// HasTagWithValue checks if a resource has a tag with the given key and value
func HasTagWithValue(tags []types.Tag, key, value string) bool {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == key && tag.Value != nil && *tag.Value == value {
			return true
		}
	}
	return false
}
