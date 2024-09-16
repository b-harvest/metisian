package metis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/b-harvest/metisian/log"
	"github.com/gorilla/websocket"
	"github.com/tendermint/tendermint/types"
	"strconv"
	"strings"
	"time"
)

const (
	QueryNewBlock  string = `tm.event='NewBlock'`
	QueryVote      string = `tm.event='Vote'`
	QueryTx        string = `tm.event='Tx'`
	QueryAndRespan string = ` AND re-propose-span.module='metis'`
)

// StatusType represents the various possible end states. Prevote and Precommit are special cases, where the node
// monitoring for misses did see them, but the proposer did not include in the block.
type StatusType int

const (
	Statusmissed StatusType = iota
	StatusPrevote
	StatusPrecommit
	StatusSigned
	StatusProposed
)

// StatusUpdate is passed over a channel from the websocket client indicating the current state, it is immediate in the
// case of prevotes etc, and the highest value seen is used in the final determination (which is how we tag
// prevote/precommit + missed blocks.
type StatusUpdate struct {
	Height int64
	Status StatusType
	Final  bool
}

// WsReply is a trimmed down version of the JSON sent from a tendermint websocket subscription.
type WsReply struct {
	Id     string `json:"id"`
	Result struct {
		Query string `json:"query"`
		Data  struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		} `json:"data"`
	} `json:"result"`
}

// Type is the abci message type
func (wsr WsReply) Type() string {
	return wsr.Result.Data.Type
}

// Value returns the JSON encoded raw bytes from the response. Unlike an ABCI RPC query, these are not protobuf.
func (wsr WsReply) Value() []byte {
	if wsr.Result.Data.Value == nil {
		return make([]byte, 0)
	}
	return wsr.Result.Data.Value
}

// WsRun is our main entrypoint for the websocket listener. In the Run loop it will block, and if it exits force a
// renegotiation for a new client.
func (c *MetisianClient) WsRun() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var err error
	started := time.Now()
	for {
		// wait until our RPC client is connected and running. We will use the same URL for the websocket
		if c.client == nil || c.GetAnySequencer().valInfo == nil {
			if started.Before(time.Now().Add(-2 * time.Minute)) {
				log.ErrorDynamicArgs("websocket client timed out waiting for a working rpc endpoint, restarting")
				return
			}
			log.Info("⏰ waiting for a healthy client")
			time.Sleep(30 * time.Second)
			continue
		}
		break
	}

	err = c.client.wsConn.SetCompressionLevel(3)
	if err != nil {
		log.Warn(err.Error())
	}

	// This go func processes the results returned by the listeners. It has most of the logic on where data is sent,
	// like dashboards or prometheus.
	resultChan := make(chan map[string]StatusUpdate)
	go func() {
		var signState StatusType = -1
		for {
			select {
			case result := <-resultChan:
				for _, seq := range c.Sequencers {
					update := result[seq.Address]
					if update.Final && update.Height%20 == 0 {
						log.ErrorDynamicArgs(fmt.Sprintf("🧊 block %d", update.Height))
					}

					if update.Status > signState {
						signState = update.Status
					}
					if update.Final {
						c.lastBlockNum = update.Height
						c.lastBlockTime = time.Now()
						c.lastBlockAlarm = false
						info := getAlarms(seq.name)
						seq.blocksResults = append([]int{int(signState)}, seq.blocksResults[:len(seq.blocksResults)-1]...)
						if signState < 3 {
							warn := fmt.Sprintf("❌ warning      %20s (%s) missed block %d", seq.name, seq.Address, update.Height)
							info += warn + "\n"
							seq.lastError = time.Now().UTC().String() + " " + info
							log.Warn(warn)
						}

						switch signState {
						case Statusmissed:
							seq.statTotalMiss += 1
							seq.statConsecutiveMiss += 1
						case StatusPrecommit:
							seq.statPrecommitMiss += 1
							seq.statTotalMiss += 1
							seq.statConsecutiveMiss += 1
						case StatusPrevote:
							seq.statPrevoteMiss += 1
							seq.statTotalMiss += 1
							seq.statConsecutiveMiss += 1
						case StatusSigned:
							seq.statTotalSigns += 1
							seq.statConsecutiveMiss = 0
						case StatusProposed:
							seq.statTotalProps += 1
							seq.statTotalSigns += 1
							seq.statConsecutiveMiss = 0
						}
						signState = -1
						healthyNodes := 0
						for i := range c.Nodes {
							if !c.Nodes[i].down {
								healthyNodes += 1
							}
						}
						switch {
						case seq.valInfo.Jailed:
							info += "- validator is jailed\n"
						}

					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	voteChan := make(chan *WsReply)
	blockChan := make(chan *WsReply)

	var addresses []string
	for _, seq := range c.Sequencers {
		if seq.name == MetisianName {
			continue
		}
		addresses = append(addresses, strings.ToUpper(strings.TrimLeft(seq.Address, "0x")))
	}

	go handleVotes(ctx, voteChan, resultChan, addresses)
	go func() {
		e := handleBlocks(ctx, blockChan, resultChan, addresses)
		if e != nil {
			log.ErrorDynamicArgs("🛑", e)
			cancel()
		}
	}()

	//respanChan := make(chan *WsReply)

	// now that channel consumers are up, create our subscriptions and route data.
	go func() {
		var msg []byte
		var e error
		for {
			_, msg, e = c.client.wsConn.ReadMessage()
			if e != nil {
				log.Error(e)
				cancel()
				return
			}
			reply := &WsReply{}
			e = json.Unmarshal(msg, reply)
			if e != nil {
				continue
			}

			switch reply.Type() {
			case `tendermint/event/NewBlock`:
				blockChan <- reply
			case `tendermint/event/Vote`:
				voteChan <- reply
			//case `tendermint/event/Tx`:
			//	respanChan <- reply
			default:
				// fmt.Println("unknown response", reply.Type())
			}
		}
	}()

	for _, subscribe := range []string{QueryNewBlock, QueryVote} {
		q := fmt.Sprintf(`{"jsonrpc":"2.0","method":"subscribe","id":1,"params":{"query":"%s"}}`, subscribe)
		err = c.client.WriteMessage(websocket.TextMessage, []byte(q))
		if err != nil {
			log.Error(err)
			cancel()
			break
		}
	}
	log.Info(fmt.Sprintf("⚙️ watching for NewBlock and Vote events via %s", c.client.wsConn.RemoteAddr()))
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

type stringInt64 string

// helper to make the "everything is a string" issue less painful.
func (si stringInt64) val() int64 {
	i, _ := strconv.ParseInt(string(si), 10, 64)
	return i
}

type signature struct {
	ValidatorAddress string `json:"validator_address"`
}

// rawBlock is a trimmed down version of the block subscription result, it contains only what we need.
type rawBlock struct {
	Block struct {
		Header struct {
			Height          stringInt64 `json:"height"`
			ProposerAddress string      `json:"proposer_address"`
		} `json:"header"`
		LastCommit struct {
			Signatures []signature `json:"signatures"`
		} `json:"last_commit"`
	} `json:"block"`
}

// find determines if a validator's pre-commit was included in a finalized block.
func (rb rawBlock) find(val string) bool {
	if rb.Block.LastCommit.Signatures == nil {
		return false
	}
	for _, v := range rb.Block.LastCommit.Signatures {
		if v.ValidatorAddress == val {
			return true
		}
	}
	return false
}

// handleBlocks consumes the channel for new blocks and when it sees one sends a status update. It's also
// responsible for stalled sequencer detection and will shutdown the client if there are no blocks for a minute.
func handleBlocks(ctx context.Context, blocks chan *WsReply, results chan map[string]StatusUpdate, addresses []string) error {
	live := time.NewTicker(time.Minute)
	defer live.Stop()
	lastBlock := time.Now()
	for {
		select {
		case <-live.C:
			// no block for a full minute likely means we have either a dead client.
			if lastBlock.Before(time.Now().Add(-time.Minute)) {
				return errors.New("websocket idle for 1 minute, exiting")
			}
		case block := <-blocks:
			lastBlock = time.Now()
			b := &rawBlock{}
			err := json.Unmarshal(block.Value(), b)
			if err != nil {
				log.ErrorDynamicArgs("could not decode block", err)
				continue
			}
			for _, address := range addresses {
				upd := StatusUpdate{
					Height: b.Block.Header.Height.val(),
					Status: Statusmissed,
					Final:  true,
				}

				if b.Block.Header.ProposerAddress == address {
					upd.Status = StatusProposed
				} else if b.find(address) {
					upd.Status = StatusSigned
				}
				results <- map[string]StatusUpdate{address: upd}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// rawVote is a trimmed down version of the vote response.
type rawVote struct {
	Vote struct {
		Type             types.SignedMsgType `json:"type"`
		Height           stringInt64         `json:"height"`
		ValidatorAddress string              `json:"validator_address"`
	} `json:"Vote"`
}

// handleVotes consumes the channel for precommits and prevotes, tracking where in the process a validator is.
func handleVotes(ctx context.Context, votes chan *WsReply, results chan map[string]StatusUpdate, addresses []string) {
	for {
		select {
		case reply := <-votes:
			vote := &rawVote{}
			err := json.Unmarshal(reply.Value(), vote)
			if err != nil {
				log.Error(err)
				continue
			}
			for _, address := range addresses {
				if vote.Vote.ValidatorAddress == address {
					upd := StatusUpdate{Height: vote.Vote.Height.val()}
					switch vote.Vote.Type {
					case types.PrevoteType:
						upd.Status = StatusPrevote
					case types.PrecommitType:
						upd.Status = StatusPrecommit
					case types.ProposalType:
						upd.Status = StatusProposed
					default:
						continue
					}
					results <- map[string]StatusUpdate{address: upd}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
