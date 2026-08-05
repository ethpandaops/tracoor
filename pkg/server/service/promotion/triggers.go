package promotion

import "github.com/attestantio/go-eth2-client/spec"

// Trigger names recorded in manifests. Selection may be sloppy-generous;
// these labels may not be, so a trigger is only ever recorded when its
// condition was positively observed.
const (
	TriggerSlashing             = "slashing"
	TriggerVoluntaryExit        = "voluntary_exit"
	TriggerBLSToExecutionChange = "bls_to_execution_change"
	TriggerDeposit              = "deposit"
	TriggerExecutionRequest     = "execution_request"
	TriggerGap                  = "gap"
	TriggerReorg                = "reorg"
	TriggerLowParticipation     = "low_participation"
	TriggerForkBoundary         = "fork_boundary"
	TriggerUndecodableFork      = "undecodable_fork"
	TriggerBaseline             = "baseline"
)

// decodedTriggers evaluates the triggers that need a typed decode. Accessor
// errors mean the field does not exist on this fork (or the library does not
// know where it lives); an unknown location degrades to fewer triggers, never
// to an error.
func (p *Promoter) decodedTriggers(block *spec.VersionedSignedBeaconBlock) []string {
	triggers := make([]string, 0, 4)
	cfg := &p.config.Triggers

	if triggerEnabled(cfg.Slashing) {
		attester, _ := block.AttesterSlashings()
		proposer, _ := block.ProposerSlashings()

		if len(attester) > 0 || len(proposer) > 0 {
			triggers = append(triggers, TriggerSlashing)
		}
	}

	if triggerEnabled(cfg.VoluntaryExit) {
		if exits, err := block.VoluntaryExits(); err == nil && len(exits) > 0 {
			triggers = append(triggers, TriggerVoluntaryExit)
		}
	}

	if triggerEnabled(cfg.BLSToExecutionChange) {
		if changes, err := block.BLSToExecutionChanges(); err == nil && len(changes) > 0 {
			triggers = append(triggers, TriggerBLSToExecutionChange)
		}
	}

	if triggerEnabled(cfg.Deposit) {
		if deposits, err := block.Deposits(); err == nil && len(deposits) > 0 {
			triggers = append(triggers, TriggerDeposit)
		}
	}

	if triggerEnabled(cfg.ExecutionRequest) {
		if requests, err := block.ExecutionRequests(); err == nil && requests != nil {
			if len(requests.Deposits)+len(requests.Withdrawals)+len(requests.Consolidations) > 0 {
				triggers = append(triggers, TriggerExecutionRequest)
			}
		}
	}

	if triggerEnabled(cfg.LowParticipation) {
		if aggregate, err := block.SyncAggregate(); err == nil && aggregate != nil {
			bits := aggregate.SyncCommitteeBits
			if total := bits.Len(); total > 0 {
				if float64(bits.Count())/float64(total) < p.config.SyncParticipationFloor {
					triggers = append(triggers, TriggerLowParticipation)
				}
			}
		}
	}

	return triggers
}
