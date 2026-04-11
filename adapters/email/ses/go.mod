module github.com/marcusPrado02/go-commons/adapters/email/ses

go 1.25

replace github.com/marcusPrado02/go-commons => ../../..

require (
	github.com/aws/aws-sdk-go-v2 v1.41.5
	github.com/aws/aws-sdk-go-v2/service/ses v1.34.22
	github.com/marcusPrado02/go-commons v0.0.0-00010101000000-000000000000
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
)
