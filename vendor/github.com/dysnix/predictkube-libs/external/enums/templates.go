package enums

import "github.com/dysnix/predictkube-proto/external/proto/enums"

//go:generate go-enum -type=CompressionType -transform=lower
// CompressionType is an enumeration of GRPC traffic compression type values
type CompressionType int

const (
	None CompressionType = iota // default compression type (if not use compression)
	Gzip                        // gzip compression type
	Zstd                        // zstd compression type
)

//go:generate go-enum -type=SSLMode -transform=lower
// SSLMode is type of sslmode postgresql connection
type SSLMode int

const (
	Enable  SSLMode = iota // SSLMode postgres connection string sslmode Enable
	Disable                // SSLMode postgres connection string sslmode Disable
)

//go:generate go-enum -type=DeletionType -transform=lower
// DeletionType is type of gorm delete action
type DeletionType int

const (
	Hard DeletionType = iota // Hard type of gorm model delete action (anyway)
	Soft                     // Soft type of gorm model delete action (change deleted_at field only)
)

var (
	_deletionTypeToProto = map[DeletionType]enums.DeleteType{
		Hard: enums.DeleteType_Hard,
		Soft: enums.DeleteType_Soft,
	}

	_deletionTypeFromProto = map[enums.DeleteType]DeletionType{
		enums.DeleteType_Hard: Hard,
		enums.DeleteType_Soft: Soft,
	}
)

func (i DeletionType) AdaptToProto() enums.DeleteType {
	return _deletionTypeToProto[i]
}

func (i *DeletionType) AdaptFromProto(in enums.DeleteType) (out DeletionType) {
	*i, _ = _deletionTypeFromProto[in]
	return *i
}

//go:generate go-enum -type=KindType
// KindType is kind of kubernetes apps/v1 resource
type KindType int

const (
	Deployment  KindType = iota // Deployment kind
	StatefulSet                 // StatefulSet kind
	DaemonSet                   // DaemonSet kind
)

var (
	_kindTypeToProto = map[KindType]enums.ResourceType{
		Deployment:  enums.ResourceType_Deployment,
		StatefulSet: enums.ResourceType_StatefulSet,
		DaemonSet:   enums.ResourceType_DaemonSet,
	}

	_kindTypeFromProto = map[enums.ResourceType]KindType{
		enums.ResourceType_Deployment:  Deployment,
		enums.ResourceType_StatefulSet: StatefulSet,
		enums.ResourceType_DaemonSet:   DaemonSet,
	}
)

func (i KindType) AdaptToProto() enums.ResourceType {
	return _kindTypeToProto[i]
}

func (i *KindType) AdaptFromProto(in enums.ResourceType) (out KindType) {
	*i, _ = _kindTypeFromProto[in]
	return *i
}

//go:generate go-enum -type=TransportType
// TransportType is a type of HTTP transport lib usage
type TransportType int

const (
	NetHTTP  TransportType = iota // NetHTTP transport from net/http package
	FastHTTP                      // FastHTTP transport from github.com/valyala/fasthttp package
)
