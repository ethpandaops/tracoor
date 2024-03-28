package store

type Type string

const (
	UnknownStore Type = "unknown"
	S3StoreType  Type = "s3"
)

func IsValidStoreType(st Type) bool {
	switch st {
	case S3StoreType:
		return true
	default:
		return false
	}
}

type DataType string

const (
	UnknownDataType        DataType = "unknown"
	BeaconStateDataType    DataType = "beacon_state"
	BeaconBlockDataType    DataType = "beacon_block"
	BeaconBadBlockDataType DataType = "beacon_bad_block"
	BeaconBadBlobDataType  DataType = "beacon_bad_blob"
	BlockTraceDataType     DataType = "execution_block_trace"
	BadBlockDataType       DataType = "execution_bad_block"
)
