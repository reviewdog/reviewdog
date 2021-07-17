package cienv

import "os"

// IsInBitbucketPipeline returns true if reviewdog is running in Bitbucket Pipelines.
func IsInBitbucketPipeline() bool {
	// https://support.atlassian.com/bitbucket-cloud/docs/variables-and-secrets/#Default-variables
	return os.Getenv("BITBUCKET_PIPELINE_UUID") != ""
}

// IsInBitbucketPipe returns true if reviewdog is running in a Bitbucket Pipe.
func IsInBitbucketPipe() bool {
	// https://support.atlassian.com/bitbucket-cloud/docs/write-a-pipe-for-bitbucket-pipelines/
	// this env variables are not really documented, but they are present in the build environment
	return os.Getenv("BITBUCKET_PIPE_STORAGE_DIR") != "" || os.Getenv("BITBUCKET_PIPE_SHARED_STORAGE_DIR") != ""
}
