package metis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/b-harvest/metisian/log"
	"github.com/gorilla/websocket"
	"github.com/machinebox/graphql"
	stakingtypes "github.com/metis-seq/themis/staking/types"
	themistypes "github.com/metis-seq/themis/types"
	"github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"net/url"
	"strings"
	"sync"
	"time"
)

// MetisianName param used for manager.
const MetisianName = "Metisian"

type MetisianClient struct {
	Ctx    context.Context
	Cancel context.CancelFunc

	// Target nodes which will connect to when fetch metis information.
	Nodes   []NodeInfo
	noNodes bool
	// NodeDownMin controls how long we wait before sending an alert that a node is not responding or has
	// fallen behind.
	NodeDownMin int `toml:"node_down_alert_minutes"`
	// NodeDownSeverity controls the Pagerduty severity when notifying if a node is down.
	NodeDownSeverity string `toml:"node_down_alert_severity"`

	client *MetisClient

	//updateChan chan *dash.ChainStatus
	//logChan    chan dash.LogMessage

	alertChan chan *alertMsg // channel used for outgoing notifications

	Sequencers map[string]*Sequencer

	minSignedPerWindow float64

	ChainId string

	seqClientMux sync.RWMutex

	// AlertIfNoServers: should an alert be sent if no servers are reachable?
	AlertIfNoServers bool `toml:"alert_if_no_servers"`

	lastBlockTime  time.Time
	lastBlockAlarm bool
	lastBlockNum   int64

	// How many minutes to wait before alerting that no new blocks have been seen
	Stalled int `toml:"stalled_minutes"`
	// Whether to alert when no new blocks are seen
	StalledAlerts bool `toml:"stalled_enabled"`
}

type NodeInfo struct {
	ApiURL      string `toml:"api_url"`
	RpcURL      string `toml:"rpc_url"`
	WsURL       string `toml:"ws_url"`
	AlertIfDown bool   `toml:"alert_if_down"`

	down      bool
	wasDown   bool
	syncing   bool
	lastMsg   string
	downSince time.Time
}

type alarmCache struct {
	SentPdAlarms   map[string]time.Time            `json:"sent_pd_alarms"`
	SentTgAlarms   map[string]time.Time            `json:"sent_tg_alarms"`
	SentDiAlarms   map[string]time.Time            `json:"sent_di_alarms"`
	SentSlkAlarms  map[string]time.Time            `json:"sent_slk_alarms"`
	AllAlarms      map[string]map[string]time.Time `json:"sent_all_alarms"`
	flappingAlarms map[string]map[string]time.Time
	notifyMux      sync.RWMutex
}

func NewClient(cfg *Config) (*MetisianClient, error) {
	var client MetisianClient

	client.alertChan = make(chan *alertMsg)
	client.Ctx, client.Cancel = context.WithCancel(context.Background())

	// configure sequencers
	client.Sequencers = map[string]*Sequencer{}
	for _, seqInfo := range cfg.Sequencers {
		if seqInfo.Alerts.UseParent {
			seqInfo.Alerts.Pagerduty = cfg.Pagerduty
			seqInfo.Alerts.Discord = cfg.Discord
			seqInfo.Alerts.Telegram = cfg.Telegram
			seqInfo.Alerts.Slack = cfg.Slack
		}

		seq := NewSequencer(seqInfo)
		client.Sequencers[seqInfo.Name] = &seq
	}
	manager := NewSequencer(
		SequencerInfo{
			Address: MetisianName,
			Name:    MetisianName,
			Alerts: AlertConfig{
				Pagerduty: cfg.Pagerduty,
				Discord:   cfg.Discord,
				Telegram:  cfg.Telegram,
				Slack:     cfg.Slack,
			},
		})
	client.Sequencers[MetisianName] = &manager

	// configure nodes
	client.Nodes = cfg.NodeInfos

	client.ChainId = cfg.ChainId

	return &client, nil
}

type ChainStatus struct {
	MsgType            string  `json:"msgType"`
	Name               string  `json:"Name"`
	ChainId            string  `json:"chain_id"`
	Moniker            string  `json:"moniker"`
	Bonded             bool    `json:"bonded"`
	Jailed             bool    `json:"jailed"`
	Tombstoned         bool    `json:"tombstoned"`
	Missed             int64   `json:"missed"`
	Window             int64   `json:"window"`
	MinSignedPerWindow float64 `json:"min_signed_per_window"`
	Nodes              int     `json:"nodes"`
	HealthyNodes       int     `json:"healthy_nodes"`
	ActiveAlerts       int     `json:"active_alerts"`
	Height             int64   `json:"height"`
	LastError          string  `json:"last_error"`

	Blocks []int `json:"blocks"`
}

type LogMessage struct {
	MsgType string `json:"msgType"`
	Ts      int64  `json:"ts"`
	Msg     string `json:"msg"`
}

func (c *MetisianClient) Run() {
	go func() {
		for {
			select {
			case alert := <-c.alertChan:
				go func(msg *alertMsg) {
					var e error
					e = notifyPagerduty(msg)
					if e != nil {
						log.ErrorDynamicArgs(msg.sequencer, "error sending alert to pagerduty", e.Error())
					}
					e = notifyDiscord(msg)
					if e != nil {
						log.ErrorDynamicArgs(msg.sequencer, "error sending alert to discord", e.Error())
					}
					e = notifyTg(msg)
					if e != nil {
						log.ErrorDynamicArgs(msg.sequencer, "error sending alert to telegram", e.Error())
					}
					e = notifySlack(msg)
					if e != nil {
						log.ErrorDynamicArgs(msg.sequencer, "error sending alert to slack", e.Error())
					}
				}(alert)
			case <-c.Ctx.Done():
				return
			}
		}
	}()

	go c.watch()

	// node health checks:
	go func() {
		for {
			c.monitorHealth(c.Ctx)
		}
	}()

	go func() {
		for {
			c.monitorSequencerSet(c.Ctx)
		}
	}()

	// websocket subscription and occasional validator info refreshes
	for {
		e := c.newRpc()
		if e != nil {
			log.Error(e)
			time.Sleep(5 * time.Second)
			continue
		}

		e = c.GetSeqValInfos()
		if e != nil {
			log.ErrorDynamicArgs("🛑", e)
		}
		c.WsRun()
		c.client.wsConn.Close()
		log.Warn("🌀 websocket exited! Restarting monitoring")
		time.Sleep(5 * time.Second)

	}

}

const SEQUENCER_SET_URL = "https://sepolia-subgraph.metisdevops.link/subgraphs/name/metisio/sequencer-set"

type MetisClient struct {
	rpcUrl       string
	wsConn       *websocket.Conn
	seqSetClient graphql.Client
}

func NewMetisClient(nodeInfo NodeInfo) (*MetisClient, error) {
	var (
		mc    MetisClient
		wsUrl *url.URL
		err   error
	)
	if mc.rpcUrl = nodeInfo.RpcURL; mc.rpcUrl == "" {
		return nil, errors.New("rpc_url must entered")
	}
	if nodeInfo.WsURL == "" {
		wsUrl, err = NewWsUrl(nodeInfo.RpcURL)
		if err != nil {
			return nil, err
		}
	} else {
		wsUrl, err = url.Parse(nodeInfo.WsURL)
		if err != nil {
			return nil, err
		}
	}

	conn := &websocket.Conn{}
	conn, _, err = websocket.DefaultDialer.Dial(wsUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not dial ws client to %s: %s", wsUrl.String(), err.Error())
	}

	mc.wsConn = conn

	mc.seqSetClient = *graphql.NewClient(SEQUENCER_SET_URL)

	return &mc, nil
}

func NewWsUrl(remote string) (*url.URL, error) {
	// normalize the path, some public rpcs prefix with /rpc or similar.
	remote = strings.TrimRight(remote, "/")
	if !strings.HasSuffix(remote, "/websocket") {
		remote += "/websocket"
	}

	endpoint, err := url.Parse(remote)
	if err != nil {
		return nil, fmt.Errorf("parsing url in NewWsClient %s: %s", remote, err.Error())
	}

	// normalize scheme to ws or wss
	switch endpoint.Scheme {
	case "http", "tcp", "ws":
		endpoint.Scheme = "ws"
	case "https", "wss":
		endpoint.Scheme = "wss"
	default:
		return nil, fmt.Errorf("protocol %s is unknown, valid choices are http, https, tcp, unix, ws, and wss", endpoint.Scheme)
	}

	return endpoint, nil
}

func (mc *MetisClient) RPCStatus() (*ctypes.ResultStatus, error) {
	c := client.NewHTTP(mc.rpcUrl, "/websocket")
	return c.Status()
}

func (mc *MetisClient) GetValidatorSet() (*themistypes.ValidatorSet, error) {
	var vset themistypes.ValidatorSet

	c := client.NewHTTP(mc.rpcUrl, "/websocket")
	res, err := c.ABCIQuery(fmt.Sprintf("custom/%s/%s", stakingtypes.QuerierRoute, stakingtypes.QueryCurrentValidatorSet), nil)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(res.Response.Value, &vset)
	if err != nil {
		return nil, err
	}

	return &vset, nil
}

func (mc *MetisClient) WriteMessage(mt int, data []byte) error {
	return mc.wsConn.WriteMessage(mt, data)
}

func (c *MetisianClient) GetAnySequencer() *Sequencer {
	for _, s := range c.Sequencers {
		return s
	}
	panic("Cannot find any sequencer!!")
}
