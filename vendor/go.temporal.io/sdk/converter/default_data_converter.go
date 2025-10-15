package converter

var (
	defaultDataConverter = NewCompositeDataConverter(
		NewNilPayloadConverter(),
		NewByteSlicePayloadConverter(),

		// Order is important here. Both ProtoJsonPayload and ProtoPayload converters check for the same proto.Message
		// interface. The first match (ProtoJsonPayload in this case) will always be used for serialization.
		// Deserialization is controlled by metadata, therefore both converters can deserialize corresponding data format
		// (JSON or binary proto).
		NewProtoJSONPayloadConverter(),
		NewProtoPayloadConverter(),

		NewJSONPayloadConverter(),
	)
)

// GetDefaultDataConverter returns default data converter used by Temporal worker.
func GetDefaultDataConverter() DataConverter {
	return defaultDataConverter
}
