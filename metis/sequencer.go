package metis

import (
	"errors"
	"github.com/metis-seq/themis/types"
)

type Sequencer struct {
	Address string

	name        string
	valInfo     *types.Validator // recent validator state, only refreshed every few minutes
	lastValInfo *types.Validator // use for detecting newly-jailed/tombstone

	blocksResults []int
	lastError     string

	activeAlerts int

	statTotalSigns      float64
	statTotalProps      float64
	statTotalMiss       float64
	statPrevoteMiss     float64
	statPrecommitMiss   float64
	statConsecutiveMiss float64

	statSeqData    *SeqData
	statNewSeqData *SeqData

	// Alerts defines the types of alerts to send for this sequencer.
	Alerts AlertConfig
}

func NewSequencer(i SequencerInfo) Sequencer {
	return Sequencer{
		Address: i.Address,
		name:    i.Name,
		Alerts:  i.Alerts,
	}
}

func (c *MetisianClient) GetSeqValInfos() (err error) {
	if c.client == nil {
		return errors.New("nil rpc client")
	}
	var vset = new(types.ValidatorSet)
	vset, err = c.client.GetValidatorSet()
	if err != nil {
		return err
	}

	for _, seq := range c.GetSequencers() {
		if seq.valInfo == nil {
			seq.valInfo = &types.Validator{}
		}

		for _, vali := range vset.Validators {
			if vali.Signer.String() == seq.Address {
				seq.valInfo = vali
				break
			}
		}
	}

	return
}
