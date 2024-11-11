package apiaccess

type APIAccessKeyErrorResponse struct {
	// The message with the error cause.
	Message string `json:"message,omitempty"`
	// Type of error.
	Type               string                      `json:"type,omitempty"`
	UserKeyErrorType   APIAccessUserKeyErrorType   `json:"userErrorType,omitempty"`
	IngestKeyErrorType APIAccessIngestKeyErrorType `json:"ingestErrorType,omitempty"`
	ID                 string                      `json:"id,omitempty"`
}
