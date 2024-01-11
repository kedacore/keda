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
	"github.com/Azure/azure-storage-blob-go/azblob"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

// goCheckpoint struct to adapt goSdk Checkpoint
type goCheckpoint struct {
	Checkpoint struct {
		SequenceNumber int64  `json:"sequenceNumber"`
		Offset         string `json:"offset"`
	} `json:"checkpoint"`
	PartitionID string `json:"partitionId"`
}

type baseCheckpoint struct {
	Epoch  int64  `json:"Epoch"`
	Offset string `json:"Offset"`
	Owner  string `json:"Owner"`
	Token  string `json:"Token"`
}

// Checkpoint in a common format
type Checkpoint struct {
	baseCheckpoint
	PartitionID    string `json:"PartitionId"`
	SequenceNumber int64  `json:"SequenceNumber"`
}

// Older python sdk stores the checkpoint differently
type pythonCheckpoint struct {
	baseCheckpoint
	PartitionID    string `json:"partition_id"`
	SequenceNumber int64  `json:"sequence_number"`
}

type checkpointer interface {
	resolvePath(info EventHubInfo) (*url.URL, error)
	extractCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error)
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

func NewCheckpoint(offset string, sequenceNumber int64) Checkpoint {
	return Checkpoint{baseCheckpoint: baseCheckpoint{Offset: offset}, SequenceNumber: sequenceNumber}
}

// GetCheckpointFromBlobStorage reads depending of the CheckpointStrategy the checkpoint from a azure storage
func GetCheckpointFromBlobStorage(ctx context.Context, httpClient util.HTTPDoer, info EventHubInfo, partitionID string) (Checkpoint, error) {
	checkpointer := newCheckpointer(info, partitionID)
	return getCheckpoint(ctx, httpClient, info, checkpointer)
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
func (checkpointer *azureFunctionCheckpointer) resolvePath(info EventHubInfo) (*url.URL, error) {
	eventHubNamespace, eventHubName, err := getHubAndNamespace(info)
	if err != nil {
		return nil, err
	}

	path, err := url.Parse(fmt.Sprintf("/%s/%s/%s/%s/%s", checkpointer.containerName, eventHubNamespace, eventHubName, info.EventHubConsumerGroup, checkpointer.partitionID))
	if err != nil {
		return nil, err
	}

	return path, nil
}

// extract checkpoint for AzureFunctionCheckpointer
func (checkpointer *azureFunctionCheckpointer) extractCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error) {
	var checkpoint Checkpoint
	err := readToCheckpointFromBody(get, &checkpoint)
	if err != nil {
		return Checkpoint{}, err
	}

	return checkpoint, nil
}

// resolve path for blobMetadataCheckpointer
func (checkpointer *blobMetadataCheckpointer) resolvePath(info EventHubInfo) (*url.URL, error) {
	eventHubNamespace, eventHubName, err := getHubAndNamespace(info)
	if err != nil {
		return nil, err
	}

	path, err := url.Parse(fmt.Sprintf("/%s/%s/%s/%s/checkpoint/%s", checkpointer.containerName, eventHubNamespace, eventHubName, strings.ToLower(info.EventHubConsumerGroup), checkpointer.partitionID))
	if err != nil {
		return nil, err
	}

	return path, nil
}

// extract checkpoint for blobMetadataCheckpointer
func (checkpointer *blobMetadataCheckpointer) extractCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error) {
	return getCheckpointFromStorageMetadata(get, checkpointer.partitionID)
}

// resolve path for goSdkCheckpointer
func (checkpointer *goSdkCheckpointer) resolvePath(info EventHubInfo) (*url.URL, error) {
	path, err := url.Parse(fmt.Sprintf("/%s/%s", info.BlobContainer, checkpointer.partitionID))
	if err != nil {
		return nil, err
	}

	return path, nil
}

// resolve path for daprCheckpointer
func (checkpointer *daprCheckpointer) resolvePath(info EventHubInfo) (*url.URL, error) {
	_, eventHubName, err := getHubAndNamespace(info)
	if err != nil {
		return nil, err
	}

	path, err := url.Parse(fmt.Sprintf("/%s/dapr-%s-%s-%s", info.BlobContainer, eventHubName, info.EventHubConsumerGroup, checkpointer.partitionID))
	if err != nil {
		return nil, err
	}

	return path, nil
}

// extract checkpoint for DaprCheckpointer
func (checkpointer *daprCheckpointer) extractCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error) {
	return newGoSdkCheckpoint(get)
}

// extract checkpoint for goSdkCheckpointer
func (checkpointer *goSdkCheckpointer) extractCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error) {
	return newGoSdkCheckpoint(get)
}

func newGoSdkCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error) {
	var checkpoint goCheckpoint
	err := readToCheckpointFromBody(get, &checkpoint)
	if err != nil {
		return Checkpoint{}, err
	}

	return Checkpoint{
		SequenceNumber: checkpoint.Checkpoint.SequenceNumber,
		baseCheckpoint: baseCheckpoint{
			Offset: checkpoint.Checkpoint.Offset,
		},
		PartitionID: checkpoint.PartitionID,
	}, nil
}

// resolve path for DefaultCheckpointer
func (checkpointer *defaultCheckpointer) resolvePath(info EventHubInfo) (*url.URL, error) {
	path, _ := url.Parse(fmt.Sprintf("/%s/%s/%s", info.BlobContainer, info.EventHubConsumerGroup, checkpointer.partitionID))

	return path, nil
}

// extract checkpoint with deprecated Python sdk checkpoint for backward compatibility
func (checkpointer *defaultCheckpointer) extractCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error) {
	var checkpoint Checkpoint
	var pyCheckpoint pythonCheckpoint
	blobData := &bytes.Buffer{}

	reader := get.Body(azblob.RetryReaderOptions{})
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

func getCheckpoint(ctx context.Context, httpClient util.HTTPDoer, info EventHubInfo, checkpointer checkpointer) (Checkpoint, error) {
	var podIdentity = info.PodIdentity

	// For back-compat, prefer a connection string over pod identity when present
	if len(info.StorageConnection) != 0 {
		podIdentity.Provider = kedav1alpha1.PodIdentityProviderNone
	}

	if podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzure || podIdentity.Provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
		if len(info.StorageAccountName) == 0 {
			return Checkpoint{}, fmt.Errorf("storageAccountName not supplied when PodIdentity authentication is enabled")
		}
	}

	blobCreds, storageEndpoint, err := ParseAzureStorageBlobConnection(ctx, httpClient,
		podIdentity, info.StorageConnection, info.StorageAccountName, info.BlobStorageEndpoint)

	if err != nil {
		return Checkpoint{}, err
	}

	path, err := checkpointer.resolvePath(info)
	if err != nil {
		return Checkpoint{}, err
	}

	baseURL := storageEndpoint.ResolveReference(path)

	get, err := downloadBlob(ctx, baseURL, blobCreds)
	if err != nil {
		return Checkpoint{}, err
	}

	return checkpointer.extractCheckpoint(get)
}

func getCheckpointFromStorageMetadata(get *azblob.DownloadResponse, partitionID string) (Checkpoint, error) {
	checkpoint := Checkpoint{
		PartitionID: partitionID,
	}

	metadata := get.NewMetadata()

	if sequencenumber, ok := metadata["sequencenumber"]; ok {
		if !ok {
			if sequencenumber, ok = metadata["Sequencenumber"]; !ok {
				return Checkpoint{}, fmt.Errorf("sequencenumber on blob not found")
			}
		}

		if sn, err := strconv.ParseInt(sequencenumber, 10, 64); err == nil {
			checkpoint.SequenceNumber = sn
		} else {
			return Checkpoint{}, fmt.Errorf("sequencenumber is not a valid int64 value: %w", err)
		}
	}

	if offset, ok := metadata["offset"]; ok {
		if !ok {
			if offset, ok = metadata["Offset"]; !ok {
				return Checkpoint{}, fmt.Errorf("offset on blob not found")
			}
		}
		checkpoint.Offset = offset
	}

	return checkpoint, nil
}

func readToCheckpointFromBody(get *azblob.DownloadResponse, checkpoint interface{}) error {
	blobData := &bytes.Buffer{}

	reader := get.Body(azblob.RetryReaderOptions{})
	if _, err := blobData.ReadFrom(reader); err != nil {
		return fmt.Errorf("failed to read blob data: %w", err)
	}
	defer reader.Close() // The client must close the response body when finished with it

	if err := json.Unmarshal(blobData.Bytes(), &checkpoint); err != nil {
		return fmt.Errorf("failed to decode blob data: %w", err)
	}

	return nil
}

func downloadBlob(ctx context.Context, baseURL *url.URL, blobCreds azblob.Credential) (*azblob.DownloadResponse, error) {
	blobURL := azblob.NewBlockBlobURL(*baseURL, azblob.NewPipeline(blobCreds, azblob.PipelineOptions{}))

	get, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to download file from blob storage: %w", err)
	}
	return get, nil
}
