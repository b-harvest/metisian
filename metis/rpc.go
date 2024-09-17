package metis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/b-harvest/metisian/log"
	"io"
	"net/http"
	"net/url"
	"time"
)

// newRpc sets up the rpc client used for monitoring. It will try nodes in order until a working node is found.
// it will also get some initial info on the validator's status.
func (c *MetisianClient) newRpc() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var anyWorking bool // if healthchecks are running, we will skip to the first known good node.
	for _, endpoint := range c.Nodes {
		anyWorking = anyWorking || !endpoint.down
	}

	// grab the first working endpoint
	tryUrl := func(nodeInfo NodeInfo) (msg string, down, syncing bool) {
		_, err := url.Parse(nodeInfo.RpcURL)
		if err != nil {
			msg = fmt.Sprintf("❌ could not parse url: (%s) %s", nodeInfo.RpcURL, err)
			log.Warn(msg)
			down = true
			return
		}
		c.client, err = NewMetisClient(nodeInfo)
		if err != nil {
			msg = fmt.Sprintf("❌ could not connect client: (%s) %s", nodeInfo.RpcURL, err)
			log.Warn(msg)
			down = true
			return
		}
		var network string
		var catching_up bool
		status, err := c.client.RPCStatus()
		if err != nil {
			n, c, err := getStatusWithEndpoint(ctx, nodeInfo.RpcURL)
			if err != nil {
				msg = fmt.Sprintf("❌ could not get status: (%s) %s", nodeInfo.RpcURL, err)
				down = true
				log.Warn(msg)
				return
			}
			network, catching_up = n, c
		} else {
			network, catching_up = status.NodeInfo.Network, status.SyncInfo.CatchingUp
		}
		if network != c.ChainId {
			msg = fmt.Sprintf("networkId %s on %s does not match, expected %s, skipping", network, nodeInfo.RpcURL, c.ChainId)
			down = true
			log.Warn(msg)
			return
		}
		if catching_up {
			msg = fmt.Sprint("🐢 node is not synced, skipping ", nodeInfo.RpcURL)
			syncing = true
			down = true
			log.Warn(msg)
			return
		}
		c.noNodes = false
		return
	}
	down := func(endpoint NodeInfo, msg string) {
		if !endpoint.down {
			endpoint.down = true
			endpoint.downSince = time.Now()
		}
		endpoint.lastMsg = msg
	}
	for _, endpoint := range c.Nodes {
		if anyWorking && endpoint.down {
			continue
		}
		if msg, failed, syncing := tryUrl(endpoint); failed {
			endpoint.syncing = syncing
			down(endpoint, msg)
			continue
		}
		return nil
	}

	c.noNodes = true
	alarms.clearAll(MetisianName)
	c.Sequencers[MetisianName].lastError = "no usable RPC endpoints available"

	return errors.New("no usable endpoints available")
}

func (c *MetisianClient) monitorHealth(ctx context.Context) {
	tick := time.NewTicker(time.Minute)
	if c.client == nil {
		_ = c.newRpc()
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-tick.C:
			var err error
			for _, node := range c.Nodes {
				go func(node NodeInfo) {
					alert := func(msg string) {
						node.lastMsg = fmt.Sprintf("node %s is %s", node.RpcURL, msg)
						if !node.AlertIfDown {
							// even if we aren't alerting, we want to display the status in the dashboard.
							node.down = true
							return
						}
						if !node.down {
							node.down = true
							node.downSince = time.Now()
						}
						log.Warn("⚠️ " + node.lastMsg)
					}
					status, e := c.client.RPCStatus()
					if e != nil {
						alert("down")
						return
					}
					if status.NodeInfo.Network != c.ChainId {
						alert("on the wrong network")
						return
					}
					if status.SyncInfo.CatchingUp {
						alert("not synced")
						node.syncing = true
						return
					}

					// node's OK, clear the note
					if node.down {
						node.lastMsg = ""
						node.wasDown = true
					}
					node.down = false
					node.syncing = false
					node.downSince = time.Unix(0, 0)
					c.noNodes = false
					log.Info(fmt.Sprintf("🟢 node %s is healthy", node.RpcURL))
				}(node)
			}

			if c.client == nil {
				e := c.newRpc()
				if e != nil {
					log.Error(errors.New(fmt.Sprintf("💥 %s %v", c.ChainId, e)))
				}
			}

			for _, seq := range c.GetSequencers() {
				if seq.valInfo != nil {
					seq.lastValInfo = seq.valInfo.Copy()
				}
				err = c.GetSeqValInfos()
				if err != nil {
					log.ErrorDynamicArgs("❓ refreshing signing info for", err)
				}
			}
		}
	}
}

func getStatusWithEndpoint(ctx context.Context, u string) (string, bool, error) {
	// Parse the URL
	parsedURL, err := url.Parse(u)
	if err != nil {
		return "", false, err
	}

	// Check if the scheme is 'tcp' and modify to 'http'
	if parsedURL.Scheme == "tcp" {
		parsedURL.Scheme = "http"
	}

	queryPath := fmt.Sprintf("%s/status", parsedURL.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryPath, nil)
	if err != nil {
		return "", false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	type tendermintStatus struct {
		JsonRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			NodeInfo struct {
				Network string `json:"network"`
			} `json:"node_info"`
			SyncInfo struct {
				CatchingUp bool `json:"catching_up"`
			} `json:"sync_info"`
		} `json:"result"`
	}
	var status tendermintStatus
	if err := json.Unmarshal(b, &status); err != nil {
		return "", false, err
	}
	return status.Result.NodeInfo.Network, status.Result.SyncInfo.CatchingUp, nil
}
