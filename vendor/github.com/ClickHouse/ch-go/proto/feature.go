package proto

//go:generate go run github.com/dmarkham/enumer -type Feature -trimprefix Feature -output feature_enum.go

// Feature represents server side feature.
type Feature int

// src/Core/ProtocolDefines.h

// Possible features.
const (
	FeatureBlockInfo                   Feature = 51903
	FeatureTimezone                    Feature = 54058
	FeatureQuotaKeyInClientInfo        Feature = 54060
	FeatureDisplayName                 Feature = 54372
	FeatureVersionPatch                Feature = 54401
	FeatureTempTables                  Feature = 50264
	FeatureServerLogs                  Feature = 54406
	FeatureColumnDefaultsMetadata      Feature = 54410
	FeatureClientWriteInfo             Feature = 54420
	FeatureSettingsSerializedAsStrings Feature = 54429
	FeatureInterServerSecret           Feature = 54441
	FeatureOpenTelemetry               Feature = 54442
	FeatureXForwardedForInClientInfo   Feature = 54443
	FeatureRefererInClientInfo         Feature = 54447
	FeatureDistributedDepth            Feature = 54448
	FeatureQueryStartTime              Feature = 54449
	FeatureProfileEvents               Feature = 54451
	FeatureParallelReplicas            Feature = 54453
	FeatureCustomSerialization         Feature = 54454
	FeatureQuotaKey                    Feature = 54458
	FeatureAddendum                    Feature = 54458
	FeatureParameters                  Feature = 54459
	FeatureServerQueryTimeInProgress   Feature = 54460
	FeatureJSONStrings                 Feature = 54475
)

// Version reports protocol version when Feature was introduced.
func (f Feature) Version() int {
	return int(f)
}

// In reports whether feature is implemented in provided protocol version.
func (f Feature) In(v int) bool {
	return v >= f.Version()
}
