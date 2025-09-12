// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3deployment"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/jsii-runtime-go"
)

// GetBucketRegion 通过bucket名称获取其所在区域
func GetBucketRegion(bucketName string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", err
	}

	s3Client := s3.NewFromConfig(cfg)
	
	result, err := s3Client.GetBucketLocation(context.TODO(), &s3.GetBucketLocationInput{
		Bucket: &bucketName,
	})
	if err != nil {
		return "", err
	}

	// AWS返回空字符串表示us-east-1
	if result.LocationConstraint == "" {
		return "us-east-1", nil
	}

	return string(result.LocationConstraint), nil
}

// CreateS3ObjectFromUrl 下载URL内容并使用 BucketDeployment 部署到S3
func CreateS3ObjectFromUrl(stack awscdk.Stack, id string, bucketName string, key string, url string) awss3deployment.BucketDeployment {
	// 下载内容
	resp, err := http.Get(url)
	if err != nil {
		panic(fmt.Sprintf("Failed to download %s: %v", url, err))
	}
	defer resp.Body.Close()
	
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read content from %s: %v", url, err))
	}
	
	return awss3deployment.NewBucketDeployment(stack, jsii.String(id), &awss3deployment.BucketDeploymentProps{
		Sources: &[]awss3deployment.ISource{
			awss3deployment.Source_Data(jsii.String(key), jsii.String(string(content)), nil),
		},
		DestinationBucket: awss3.Bucket_FromBucketName(stack, jsii.String(id+"-bucket"), jsii.String(bucketName)),
	})
}
