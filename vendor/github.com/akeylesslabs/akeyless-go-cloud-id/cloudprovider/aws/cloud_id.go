package aws

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

func GetCloudId() (string, error) {
	// Endpoint https://sts.amazonaws.com is available only in single region: us-east-1.
	// So, caller identity request can be only us-east-1. Default call brings region where caller is
	region := "us-east-1"

	sess, err := session.NewSession()
	if err != nil {
		return "", err
	}

	svc := sts.New(sess, aws.NewConfig().WithRegion(region))
	input := &sts.GetCallerIdentityInput{}
	req, _ := svc.GetCallerIdentityRequest(input)

	if err := req.Sign(); err != nil {
		return "", err
	}

	headersJson, err := json.Marshal(req.HTTPRequest.Header)
	if err != nil {
		return "", err
	}
	requestBody, err := ioutil.ReadAll(req.HTTPRequest.Body)
	if err != nil {
		return "", err
	}

	awsData := make(map[string]string)
	awsData["sts_request_method"] = req.HTTPRequest.Method
	awsData["sts_request_url"] = base64.StdEncoding.EncodeToString([]byte(req.HTTPRequest.URL.String()))
	awsData["sts_request_body"] = base64.StdEncoding.EncodeToString(requestBody)
	awsData["sts_request_headers"] = base64.StdEncoding.EncodeToString(headersJson)
	awsDataDump, err := json.Marshal(awsData)

	if err != nil {
		return "", err
	}

	cloudId := base64.StdEncoding.EncodeToString(awsDataDump)
	return cloudId, nil
}
