package metis

import (
	"context"
	"fmt"
	"github.com/b-harvest/metisian/log"
	"github.com/machinebox/graphql"
	"strconv"
	"strings"
	"time"
)

type SequencerSetResponse struct {
	Data SeqData `json:"data"`
}

type SeqData struct {
	Epoches []*Epoch `json:"epoches"`
	IsNow   bool     `json:"is_now"`
}

type Epoch struct {
	StartBlock     string `json:"startBlock"`
	Recommited     bool   `json:"recommited"`
	Block          string `json:"block"`
	BlockTimestamp string `json:"blockTimestamp"`
	ID             string `json:"id"`
	Transaction    string `json:"transaction"`
	EndBlock       string `json:"endBlock"`
	Signer         string `json:"signer"`
}

type Block struct {
	Number    int64  `json:"number"`
	Hash      string `json:"hash"`
	Timestamp int64  `json:"timestamp"`
}

func (d *SeqData) find(epochId string) *Epoch {
	for _, epoch := range d.Epoches {
		if epoch.ID == epochId {
			return epoch
		}
	}
	return nil
}

func (c *MetisianClient) monitorSequencerSet(ctx context.Context) {
	tick := time.NewTicker(time.Second * 30)

	log.Info(fmt.Sprintf("⚙️ watching for Sequencer-set subgraph"))
	for {
		select {
		case <-ctx.Done():
			return

		case <-tick.C:
			var (
				err                error
				currentBlockNumber int64
			)
			currentBlockNumber, _ = c.GetEthBlockNumber()
			for _, seq := range c.GetSequencers() {
				go func(s *Sequencer) {
					req := graphql.NewRequest(`
query ($skip: Int, $first: Int, $address: String) {
    epoches(
        skip: $skip
        first: $first
        orderBy: block
        orderDirection: desc
        subgraphError: allow
        where: { signer: $address }
    ) {
        id
        startBlock
        endBlock
        signer
        transaction
        recommited
        block
        blockTimestamp
    }
    _meta {
        block {
            hash
            number
            timestamp
        }
        deployment
        hasIndexingErrors
    }
}
`)

					req.Var("address", s.Address)
					req.Var("first", 10)
					req.Var("skip", 0)

					req.Header.Set("Cache-Control", "no-cache")

					var respData SeqData
					if err = c.client.seqSetClient.Run(ctx, req, &respData); err != nil {
						log.Warn(fmt.Sprintf("%v", err))
						return
					}

					for _, epoch := range respData.Epoches {
						epochId := isDecimal(epoch.ID)

						// call ParseUint() function and pass the hexadecimal number as argument to it
						epochIdDec, _ := strconv.ParseUint(epochId, 16, 64)
						epoch.ID = fmt.Sprintf("%d", epochIdDec)
					}

					if len(respData.Epoches) > 0 {
						startBlockNumber, _ := strconv.ParseInt(respData.Epoches[0].StartBlock, 0, 64)
						endBlockNumber, _ := strconv.ParseInt(respData.Epoches[0].EndBlock, 0, 64)

						respData.IsNow = startBlockNumber < currentBlockNumber && currentBlockNumber < endBlockNumber
					}

					s.statNewSeqData = &respData
				}(seq)
			}

		}
	}
}

// function to get the hexadecimal number from string
func isDecimal(hexaString string) string {

	// replace 0x or 0X with empty String
	number := strings.Replace(hexaString, "0x", "", -1)
	number = strings.Replace(number, "0X", "", -1)

	// returns the hexadecimal number from a string
	return number
}
