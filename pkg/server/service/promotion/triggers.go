package promotion

import (
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
)

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

	if triggerEnabled(cfg.ExecutionRequest) && hasExecutionRequests(block) {
		triggers = append(triggers, TriggerExecutionRequest)
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

// hasExecutionRequests reports whether this block processes execution-layer
// triggered requests. Gloas (EIP-7732) moves the payload out of the block, so
// the requests a gloas block processes are its parent payload's and the
// versioned accessor declines them - read them off the body instead. EIP-8282
// builder requests count too: they are the same kind of rare, state-moving
// event the trigger exists to catch.
func hasExecutionRequests(block *spec.VersionedSignedBeaconBlock) bool {
	if body := gloasBody(block); body != nil {
		r := body.ParentExecutionRequests
		if r == nil {
			return false
		}

		return len(r.Deposits)+len(r.Withdrawals)+len(r.Consolidations)+
			len(r.BuilderDeposits)+len(r.BuilderExits) > 0
	}

	requests, err := block.ExecutionRequests()
	if err != nil || requests == nil {
		return false
	}

	deposits, _ := requests.Deposits()
	withdrawals, _ := requests.Withdrawals()
	consolidations, _ := requests.Consolidations()

	return len(deposits)+len(withdrawals)+len(consolidations) > 0
}

// gloasBody returns the block body for the gloas-shaped forks, or nil when the
// block is an earlier fork. Heze reuses the gloas containers.
func gloasBody(block *spec.VersionedSignedBeaconBlock) *gloas.BeaconBlockBody {
	var b *gloas.SignedBeaconBlock

	switch block.Version {
	case spec.DataVersionGloas:
		b = block.Gloas
	case spec.DataVersionHeze:
		b = block.Heze
	default:
		return nil
	}

	if b == nil || b.Message == nil {
		return nil
	}

	return b.Message.Body
}
