package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	minio "github.com/minio/minio-go"
	"github.com/openfaas/openfaas-cloud/sdk"
)

const bucketName = "pipeline"

// Handle a serverless request
func Handle(req []byte) string {
	region := os.Getenv("s3_region")
	method := os.Getenv("Http_Method")

	minioClient, err := connectToMinio(region)
	if err != nil {
		return err.Error()
	}

	switch method {
	case http.MethodPost:
		pipelineLog := sdk.PipelineLog{}
		json.Unmarshal(req, &pipelineLog)

		minioClient.MakeBucket(bucketName, region)

		reader := bytes.NewReader([]byte(pipelineLog.Data))
		fullPath := getPath(bucketName, &pipelineLog)
		n, err := minioClient.PutObject(bucketName,
			fullPath,
			reader,
			int64(reader.Len()),
			minio.PutObjectOptions{})

		if err != nil {
			return err.Error()
		}
		return fmt.Sprintf("Wrote %d bytes to %s\n", n, fullPath)

	case http.MethodGet:
		queryRaw := os.Getenv("Http_Query")
		query, parseErr := url.ParseQuery(queryRaw)
		if parseErr != nil {
			return parseErr.Error()
		}

		p := sdk.PipelineLog{
			CommitSHA: query.Get("commitSHA"),
			Function:  query.Get("function"),
			RepoPath:  query.Get("repoPath"),
		}

		fullPath := getPath(bucketName, &p)
		log.Printf("Reading %s\n", fullPath)
		obj, err := minioClient.GetObject(bucketName, fullPath, minio.GetObjectOptions{})

		if err != nil {
			return err.Error()
		}

		logBytes, _ := ioutil.ReadAll(obj)

		return string(logBytes)
	}

	return fmt.Sprintf("pipeline-log, unknown request")
}

func connectToMinio(region string) (*minio.Client, error) {

	endpoint := os.Getenv("s3_url")

	secretKey, _ := sdk.ReadSecret("s3-secret-key")
	accessKey, _ := sdk.ReadSecret("s3-access-key")

	return minio.New(endpoint, accessKey, secretKey, false)
}

// getPath produces a string such as pipeline/alexellis/super-pancake-fn/commit-id/fn1/
func getPath(bucket string, p *sdk.PipelineLog) string {
	fileName := "build.log"
	return fmt.Sprintf("%s/%s/%s/%s/%s", bucket, p.RepoPath, p.CommitSHA, p.Function, fileName)
}
