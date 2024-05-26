/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"dario.cat/mergo"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// goCheckpoint struct to adapt goSdk Checkpoint
type goCheckpoint struct {
	Checkpoint struct {
		SequenceNumber int64 `json:"sequenceNumber"`
	} `json:"checkpoint"`
	PartitionID string `json:"partitionId"`
}

// Checkpoint in a common format
type Checkpoint struct {
	PartitionID    string `json:"PartitionId"`
	SequenceNumber int64  `json:"SequenceNumber"`
}

// Older python sdk stores the checkpoint differently
type pythonCheckpoint struct {
	PartitionID    string `json:"partition_id"`
	SequenceNumber int64  `json:"sequence_number"`
}

type checkpointer interface {
	resolvePath(info EventHubInfo) (string, string, error)
	extractCheckpoint(get *azblob.DownloadStreamResponse) (Checkpoint, error)
}

type azureFunctionCheckpointer struct {
	partitionID   string
	containerName string
}

type blobMetadataCheckpointer struct {
	partitionID   string
	containerName string
}

type goSdkCheckpointer struct {
	partitionID   string
	containerName string
}

type daprCheckpointer struct {
	partitionID   string
	containerName string
}

type defaultCheckpointer struct {
	partitionID   string
	containerName string
}

func NewCheckpoint(sequenceNumber int64) Checkpoint {
	return Checkpoint{SequenceNumber: sequenceNumber}
}

// GetCheckpointFromBlobStorage reads depending of the CheckpointStrategy the checkpoint from a azure storage
func GetCheckpointFromBlobStorage(ctx context.Context, blobStorageClient *azblob.Client, info EventHubInfo, partitionID string) (Checkpoint, error) {
	checkpointer := newCheckpointer(info, partitionID)
	return getCheckpoint(ctx, blobStorageClient, info, checkpointer)
}

func newCheckpointer(info EventHubInfo, partitionID string) checkpointer {
	switch {
	case info.CheckpointStrategy == "goSdk":
		return &goSdkCheckpointer{
			containerName: info.BlobContainer,
			partitionID:   partitionID,
		}
	case info.CheckpointStrategy == "dapr":
		return &daprCheckpointer{
			containerName: info.BlobContainer,
			partitionID:   partitionID,
		}
	case info.CheckpointStrategy == "blobMetadata":
		return &blobMetadataCheckpointer{
			containerName: info.BlobContainer,
			partitionID:   partitionID,
		}
	case info.CheckpointStrategy == "azureFunction" || info.BlobContainer == "":
		return &azureFunctionCheckpointer{
			containerName: "azure-webjobs-eventhub",
			partitionID:   partitionID,
		}
	default:
		return &defaultCheckpointer{
			containerName: info.BlobContainer,
			partitionID:   partitionID,
		}
	}
}

// resolve path for AzureFunctionCheckpointer
func (checkpointer *azureFunctionCheckpointer) resolvePath(info EventHubInfo) (string, string, error) {
	eventHubNamespace, eventHubName, err := getHubAndNamespace(info)
	if err != nil {
		return "", "", err
	}

	path := fmt.Sprintf("%s/%s/%s/%s", eventHubNamespace, eventHubName, info.EventHubConsumerGroup, checkpointer.partitionID)
	if _, err := url.Parse(path); err != nil {
		return "", "", err
	}
	return checkpointer.containerName, path, nil
}

// extract checkpoint for AzureFunctionCheckpointer
func (checkpointer *azureFunctionCheckpointer) extractCheckpoint(get *azblob.DownloadStreamResponse) (Checkpoint, error) {
	var checkpoint Checkpoint
	err := readToCheckpointFromBody(get, &checkpoint)
	if err != nil {
		return Checkpoint{}, err
	}

	return checkpoint, nil
}

// resolve path for blobMetadataCheckpointer
func (checkpointer *blobMetadataCheckpointer) resolvePath(info EventHubInfo) (string, string, error) {
	eventHubNamespace, eventHubName, err := getHubAndNamespace(info)
	if err != nil {
		return "", "", err
	}

	path := fmt.Sprintf("%s/%s/%s/checkpoint/%s", eventHubNamespace, eventHubName, strings.ToLower(info.EventHubConsumerGroup), checkpointer.partitionID)
	if _, err := url.Parse(path); err != nil {
		return "", "", err
	}
	return checkpointer.containerName, path, nil
}

// extract checkpoint for blobMetadataCheckpointer
func (checkpointer *blobMetadataCheckpointer) extractCheckpoint(get *azblob.DownloadStreamResponse) (Checkpoint, error) {
	return getCheckpointFromStorageMetadata(get, checkpointer.partitionID)
}

// resolve path for goSdkCheckpointer
func (checkpointer *goSdkCheckpointer) resolvePath(info EventHubInfo) (string, string, error) {
	return info.BlobContainer, checkpointer.partitionID, nil
}

// resolve path for daprCheckpointer
func (checkpointer *daprCheckpointer) resolvePath(info EventHubInfo) (string, string, error) {
	_, eventHubName, err := getHubAndNamespace(info)
	if err != nil {
		return "", "", err
	}

	path := fmt.Sprintf("dapr-%s-%s-%s", eventHubName, info.EventHubConsumerGroup, checkpointer.partitionID)
	if _, err := url.Parse(path); err != nil {
		return "", "", err
	}
	return info.BlobContainer, path, nil
}

// extract checkpoint for DaprCheckpointer
func (checkpointer *daprCheckpointer) extractCheckpoint(get *azblob.DownloadStreamResponse) (Checkpoint, error) {
	return newGoSdkCheckpoint(get)
}

// extract checkpoint for goSdkCheckpointer
func (checkpointer *goSdkCheckpointer) extractCheckpoint(get *azblob.DownloadStreamResponse) (Checkpoint, error) {
	return newGoSdkCheckpoint(get)
}

func newGoSdkCheckpoint(get *azblob.DownloadStreamResponse) (Checkpoint, error) {
	var checkpoint goCheckpoint
	err := readToCheckpointFromBody(get, &checkpoint)
	if err != nil {
		return Checkpoint{}, err
	}

	return Checkpoint{
		SequenceNumber: checkpoint.Checkpoint.SequenceNumber,
		PartitionID:    checkpoint.PartitionID,
	}, nil
}

// resolve path for DefaultCheckpointer
func (checkpointer *defaultCheckpointer) resolvePath(info EventHubInfo) (string, string, error) {
	path := fmt.Sprintf("%s/%s", info.EventHubConsumerGroup, checkpointer.partitionID)
	return info.BlobContainer, path, nil
}

// extract checkpoint with deprecated Python sdk checkpoint for backward compatibility
func (checkpointer *defaultCheckpointer) extractCheckpoint(get *azblob.DownloadStreamResponse) (Checkpoint, error) {
	var checkpoint Checkpoint
	var pyCheckpoint pythonCheckpoint
	blobData := &bytes.Buffer{}

	reader := get.Body
	if _, err := blobData.ReadFrom(reader); err != nil {
		return Checkpoint{}, fmt.Errorf("failed to read blob data: %w", err)
	}
	defer reader.Close() // The client must close the response body when finished with it

	if err := json.Unmarshal(blobData.Bytes(), &checkpoint); err != nil {
		return Checkpoint{}, fmt.Errorf("failed to decode blob data: %w", err)
	}

	if err := json.Unmarshal(blobData.Bytes(), &pyCheckpoint); err != nil {
		return Checkpoint{}, fmt.Errorf("failed to decode blob data: %w", err)
	}

	err := mergo.Merge(&checkpoint, Checkpoint(pyCheckpoint))

	return checkpoint, err
}

func getCheckpoint(ctx context.Context, blobStorageClient *azblob.Client, info EventHubInfo, checkpointer checkpointer) (Checkpoint, error) {
	container, path, err := checkpointer.resolvePath(info)
	if err != nil {
		return Checkpoint{}, err
	}

	get, err := blobStorageClient.DownloadStream(ctx, container, path, nil)
	if err != nil {
		return Checkpoint{}, err
	}

	return checkpointer.extractCheckpoint(&get)
}

func getCheckpointFromStorageMetadata(get *azblob.DownloadStreamResponse, partitionID string) (Checkpoint, error) {
	checkpoint := Checkpoint{
		PartitionID: partitionID,
	}

	metadata := get.Metadata

	var sequencenumber *string
	ok := false
	if sequencenumber, ok = metadata["sequencenumber"]; !ok {
		if sequencenumber, ok = metadata["Sequencenumber"]; !ok {
			return Checkpoint{}, fmt.Errorf("sequencenumber on blob not found")
		}
	}

	if sn, err := strconv.ParseInt(*sequencenumber, 10, 64); err == nil {
		checkpoint.SequenceNumber = sn
	} else {
		return Checkpoint{}, fmt.Errorf("sequencenumber is not a valid int64 value: %w", err)
	}
	return checkpoint, nil
}

func readToCheckpointFromBody(get *azblob.DownloadStreamResponse, checkpoint interface{}) error {
	blobData := &bytes.Buffer{}

	reader := get.Body
	if _, err := blobData.ReadFrom(reader); err != nil {
		return fmt.Errorf("failed to read blob data: %w", err)
	}
	defer reader.Close() // The client must close the response body when finished with it

	if err := json.Unmarshal(blobData.Bytes(), &checkpoint); err != nil {
		return fmt.Errorf("failed to decode blob data: %w", err)
	}

	return nil
}
