package persistence

type Operation string

const (
	OperationDistinctValues Operation = "distinct_values"

	OperationInsertBeaconState Operation = "insert_beacon_state_metadata"
	OperationDeleteBeaconState Operation = "delete_beacon_state_metadata"
	OperationCountBeaconState  Operation = "count_beacon_state_metadata"
	OperationListBeaconState   Operation = "list_beacon_state_metadata"
	OperationUpdateBeaconState Operation = "update_beacon_state_metadata"

	OperationInsertBeaconBlock Operation = "insert_beacon_block_metadata"
	OperationDeleteBeaconBlock Operation = "delete_beacon_block_metadata"
	OperationCountBeaconBlock  Operation = "count_beacon_block_metadata"
	OperationListBeaconBlock   Operation = "list_beacon_block_metadata"
	OperationUpdateBeaconBlock Operation = "update_beacon_block_metadata"

	OperationInsertBeaconBadBlock Operation = "insert_beacon_bad_block_metadata"
	OperationDeleteBeaconBadBlock Operation = "delete_beacon_bad_block_metadata"
	OperationCountBeaconBadBlock  Operation = "count_beacon_bad_block_metadata"
	OperationListBeaconBadBlock   Operation = "list_beacon_bad_block_metadata"
	OperationUpdateBeaconBadBlock Operation = "update_beacon_bad_block_metadata"

	OperationInsertBeaconBadBlob Operation = "insert_beacon_bad_blob_metadata"
	OperationDeleteBeaconBadBlob Operation = "delete_beacon_bad_blob_metadata"
	OperationCountBeaconBadBlob  Operation = "count_beacon_bad_blob_metadata"
	OperationListBeaconBadBlob   Operation = "list_beacon_bad_blob_metadata"
	OperationUpdateBeaconBadBlob Operation = "update_beacon_bad_blob_metadata"

	OperationInsertExecutionBlockTrace Operation = "insert_execution_block_trace"
	OperationDeleteExecutionBlockTrace Operation = "delete_execution_block_trace"
	OperationCountExecutionBlockTrace  Operation = "count_execution_block_trace"
	OperationListExecutionBlockTrace   Operation = "list_execution_block_trace"
	OperationUpdateExecutionBlockTrace Operation = "update_execution_block_trace"

	OperationInsertExecutionBadBlock Operation = "insert_execution_bad_block"
	OperationDeleteExecutionBadBlock Operation = "delete_execution_bad_block"
	OperationCountExecutionBadBlock  Operation = "count_execution_bad_block"
	OperationListExecutionBadBlock   Operation = "list_execution_bad_block"
	OperationUpdateExecutionBadBlock Operation = "update_execution_bad_block"
)
