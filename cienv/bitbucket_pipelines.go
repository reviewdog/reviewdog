package cienv

import "os"

// IsInBitbucketPipeline returns true if reviewdog is running in Bitbucket Pipelines.
func IsInBitbucketPipeline() bool {
	// https://support.atlassian.com/bitbucket-cloud/docs/variables-and-secrets/#Default-variables
	return os.Getenv("BITBUCKET_PIPELINE_UUID") != ""
}

// IsInBitbucketPipe returns true if reviewdog is running in a Bitbucket Pipe.
func IsInBitbucketPipe() bool {
	//
	return os.Getenv("BITBUCKET_PIPE_STORAGE_DIR") != ""
}
