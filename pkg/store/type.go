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
	UnknownDataType     DataType = "unknown"
	BeaconStateDataType DataType = "beacon_state"
	BlockTraceDataType  DataType = "execution_block_trace"
	BadBlockDataType    DataType = "execution_bad_block"
)
