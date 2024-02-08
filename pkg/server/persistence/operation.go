package persistence

type Operation string

const (
	OperationInsertBeaconState Operation = "insert_beacon_state_metadata"
	OperationDeleteBeaconState Operation = "delete_beacon_state_metadata"
	OperationCountBeaconState  Operation = "count_beacon_state_metadata"
	OperationListBeaconState   Operation = "list_beacon_state_metadata"
	OperationUpdateBeaconState Operation = "update_beacon_state_metadata"

	OperationDistinctValues Operation = "distinct_values"

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
