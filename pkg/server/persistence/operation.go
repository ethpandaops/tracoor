package persistence

type Operation string

const (
	OperationInsertBeaconState Operation = "insert_beacon_state_metadata"
	OperationDeleteBeaconState Operation = "delete_beacon_state_metadata"
	OperationCountBeaconState  Operation = "count_beacon_state_metadata"
	OperationListBeaconState   Operation = "list_beacon_state_metadata"
	OperationUpdateBeaconState Operation = "update_beacon_state_metadata"
)
