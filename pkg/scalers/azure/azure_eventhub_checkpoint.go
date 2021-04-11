package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/imdario/mergo"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

// goCheckpoint struct to adapt GoSdk Checkpoint
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

type azureWebjobCheckpointer struct {
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

type defaultCheckpointer struct {
	partitionID   string
	containerName string
}

// GetCheckpointFromBlobStorage reads depending of the CheckpointType the checkpoint from a azure storage
func GetCheckpointFromBlobStorage(ctx context.Context, httpClient util.HTTPDoer, info EventHubInfo, partitionID string) (Checkpoint, error) {
	checkpointer := newCheckpointer(info, partitionID)
	return getCheckpoint(ctx, httpClient, info, checkpointer)
}

func newCheckpointer(info EventHubInfo, partitionID string) checkpointer {
	if info.CheckpointType == "GoSdk" {
		return &goSdkCheckpointer{
			containerName: info.BlobContainer,
			partitionID:   partitionID,
		}
	} else if info.CheckpointType == "BlobMetadata" {
		return &blobMetadataCheckpointer{
			containerName: info.BlobContainer,
			partitionID:   partitionID,
		}
	} else if info.CheckpointType == "AzureWebJob" || info.BlobContainer == "" {
		return &azureWebjobCheckpointer{
			containerName: "azure-webjobs-eventhub",
			partitionID:   partitionID,
		}
	} else {
		return &defaultCheckpointer{
			containerName: info.BlobContainer,
			partitionID:   partitionID,
		}
	}
}

// resolve path for AzureWebJobCheckpointer
func (checkpointer *azureWebjobCheckpointer) resolvePath(info EventHubInfo) (*url.URL, error) {
	eventHubNamespace, eventHubName, err := getHubAndNamespace(info)
	if err != nil {
		return nil, err
	}

	path, _ := url.Parse(fmt.Sprintf("/%s/%s/%s/%s/%s", checkpointer.containerName, eventHubNamespace, eventHubName, info.EventHubConsumerGroup, checkpointer.partitionID))

	return path, nil
}

// extract checkpoint for AzureWebJobCheckpointer
func (checkpointer *azureWebjobCheckpointer) extractCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error) {
	var checkpoint Checkpoint
	err := readToCheckpointFromBody(get, &checkpoint)
	if err != nil {
		return Checkpoint{}, err
	}

	return checkpoint, nil
}

// resolve path for BlobMetadataCheckpointer
func (checkpointer *blobMetadataCheckpointer) resolvePath(info EventHubInfo) (*url.URL, error) {
	eventHubNamespace, eventHubName, err := getHubAndNamespace(info)
	if err != nil {
		return nil, err
	}

	path, _ := url.Parse(fmt.Sprintf("/%s/%s/%s/%s/checkpoint/%s", checkpointer.containerName, eventHubNamespace, eventHubName, info.EventHubConsumerGroup, checkpointer.partitionID))

	return path, nil
}

// extract checkpoint for BlobMetadataCheckpointer
func (checkpointer *blobMetadataCheckpointer) extractCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error) {
	return getCheckpointFromStorageMetadata(get, checkpointer.partitionID)
}

// resolve path for GoSdkCheckpointer
func (checkpointer *goSdkCheckpointer) resolvePath(info EventHubInfo) (*url.URL, error) {
	path, _ := url.Parse(fmt.Sprintf("/%s/%s", info.BlobContainer, checkpointer.partitionID))

	return path, nil
}

// extract checkpoint for GoSdkCheckpointer
func (checkpointer *goSdkCheckpointer) extractCheckpoint(get *azblob.DownloadResponse) (Checkpoint, error) {
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
		return Checkpoint{}, fmt.Errorf("failed to read blob data: %s", err)
	}
	defer reader.Close() // The client must close the response body when finished with it

	if err := json.Unmarshal(blobData.Bytes(), &checkpoint); err != nil {
		return Checkpoint{}, fmt.Errorf("failed to decode blob data: %s", err)
	}

	if err := json.Unmarshal(blobData.Bytes(), &pyCheckpoint); err != nil {
		return Checkpoint{}, fmt.Errorf("failed to decode blob data: %s", err)
	}

	err := mergo.Merge(&checkpoint, Checkpoint(pyCheckpoint))

	return checkpoint, err
}

func getCheckpoint(ctx context.Context, httpClient util.HTTPDoer, info EventHubInfo, checkpointer checkpointer) (Checkpoint, error) {
	blobCreds, storageEndpoint, err := ParseAzureStorageBlobConnection(httpClient, kedav1alpha1.PodIdentityProviderNone, info.StorageConnection, "")
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
		return fmt.Errorf("failed to read blob data: %s", err)
	}
	defer reader.Close() // The client must close the response body when finished with it

	if err := json.Unmarshal(blobData.Bytes(), &checkpoint); err != nil {
		return fmt.Errorf("failed to decode blob data: %s", err)
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
