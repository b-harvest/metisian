package metis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/b-harvest/metisian/log"
	dash "github.com/b-harvest/metisian/metis/dashboard"
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
	ChainId string

	Ctx    context.Context
	Cancel context.CancelFunc

	Nodes   []NodeInfo
	noNodes bool

	NodeDownMin      int
	NodeDownSeverity string

	Stalled       int
	StalledAlerts bool

	AlertIfNoServers bool

	client *MetisClient

	updateChan chan *dash.SequencerStatus
	logChan    chan dash.LogMessage

	alertChan chan *alertMsg // channel used for outgoing notifications

	Sequencers map[string]*Sequencer

	seqClientMux sync.RWMutex

	lastBlockTime  time.Time
	lastBlockAlarm bool
	lastBlockNum   int64

	EnableDash bool
	Listen     string
	HideLogs   bool
}

func (mc *MetisianClient) GetSequencers() map[string]*Sequencer {
	var res = map[string]*Sequencer{}
	for _, seq := range mc.Sequencers {
		if seq.name != MetisianName {
			res[seq.name] = seq
		}
	}
	return res
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

const (
	showBlocks = 512
	staleHours = 24
)

func NewClient(cfg *Config) (*MetisianClient, error) {
	var (
		client MetisianClient
		err    error
	)

	// Check if configurations are valid
	if cfg.EnableDash {
		_, err = url.Parse(cfg.Listen)
		if err != nil || cfg.Listen == "" {
			log.Fatal(errors.New(fmt.Sprintf("error: The listen URL %s does not appear to be valid", cfg.Listen)))
		}
	}

	if cfg.NodeDownMin < 3 {
		log.Fatal(errors.New("warning: setting 'node_down_alert_minutes' to less than three minutes might result in false alarms"))
	}

	client.alertChan = make(chan *alertMsg)
	client.logChan = make(chan dash.LogMessage)
	client.updateChan = make(chan *dash.SequencerStatus, len(client.Sequencers)*2)
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

	client.Nodes = cfg.NodeInfos
	client.ChainId = cfg.ChainId
	client.NodeDownMin = cfg.NodeDownMin
	client.NodeDownSeverity = cfg.NodeDownSeverity
	client.Stalled = cfg.Stalled
	client.StalledAlerts = cfg.StalledAlerts
	client.AlertIfNoServers = cfg.AlertIfNoServers
	client.EnableDash = cfg.EnableDash
	client.Listen = cfg.Listen
	client.HideLogs = cfg.HideLogs

	return &client, nil
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

	if c.EnableDash {
		go dash.Serve(c.Listen, c.updateChan, c.logChan, c.HideLogs)
		log.Info("⚙️ starting dashboard on " + c.Listen)
	} else {
		go func() {
			for {
				<-c.updateChan
			}
		}()
	}

	for _, seq := range c.GetSequencers() {
		if c.EnableDash {
			if seq.blocksResults == nil {
				seq.blocksResults = make([]int, showBlocks)
				for i := range seq.blocksResults {
					seq.blocksResults[i] = -1
				}
			}

			c.updateChan <- &dash.SequencerStatus{
				MsgType:      "status",
				Name:         seq.name,
				Address:      seq.Address,
				Jailed:       false, // TODO replace `false` to seq.valInfo.jailed when save file logic has implemented.
				ActiveAlerts: 0,
				Blocks:       seq.blocksResults,
			}
		}
	}

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
	for _, s := range c.GetSequencers() {
		return s
	}
	panic("Cannot find any sequencer!!")
}
