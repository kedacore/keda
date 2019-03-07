package scalers

import (
	"context"
	log "github.com/Sirupsen/logrus"
)

type AzureQueueScaler struct {
	ResolvedSecrets, Metadata map[string]string
}

// GetScaleDecision is a func
func (s *AzureQueueScaler) GetScaleDecision(ctx context.Context) (int32, error) {
	connectionString := getConnectionString(s)
	queueName := getQueueName(s)

	length, err := getQueueLength(ctx, connectionString, queueName)

	if err != nil {
		log.Errorf("error %s", err)
		return -1, err
	}

	if length > 0 {
		return 1, nil
	}

	return 0, nil
}

func getConnectionString(s *AzureQueueScaler) string {
	connectionSettingName := s.Metadata["connection"]
	if connectionSettingName == "" {
		connectionSettingName = "AzureWebJobsStorage"
	}

	return s.ResolvedSecrets[connectionSettingName]
}

func getQueueName(s *AzureQueueScaler) string {
	return s.Metadata["queueName"]
}
