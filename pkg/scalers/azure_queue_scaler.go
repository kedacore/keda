package scalers

import (
	log "github.com/Sirupsen/logrus"
)

type azureQueueScaler struct {
	resolvedSecrets, metadata *map[string]string
}

// GetScaleDecision is a func
func (s *azureQueueScaler) GetScaleDecision() (int32, error) {
	connectionString := getConnectionString(s)
	queueName := getQueueName(s)

	length, err := getQueueLength(connectionString, queueName)

	if err != nil {
		log.Errorf("error %s", err)
		return -1, err
	}

	if length > 0 {
		return 1, nil
	}

	return 0, nil
}

func getConnectionString(s *azureQueueScaler) string {
	connectionSettingName := (*s.metadata)["connection"]
	if connectionSettingName == "" {
		connectionSettingName = "AzureWebJobsStorage"
	}

	return (*s.resolvedSecrets)[connectionSettingName]
}

func getQueueName(s *azureQueueScaler) string {
	return (*s.metadata)["queueName"]
}
