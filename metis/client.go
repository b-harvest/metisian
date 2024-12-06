package metis

import (
	"bytes"
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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// MetisianName param used for manager.
const MetisianName = "Metisian"

type MetisianClient struct {
	ChainId string

	SequencerSetUrl string
	L2RpcUrl        string

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

	seqMux sync.RWMutex

	lastBlockTime  time.Time
	lastBlockAlarm bool
	lastBlockNum   int64

	EnableDash bool
	Listen     string
	HideLogs   bool
}

func (c *MetisianClient) GetSequencers() map[string]*Sequencer {
	var res = map[string]*Sequencer{}
	for _, seq := range c.Sequencers {
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
	showBlocks       = 512
	staleHours       = 24
	SEPOLIA_CHAIN_ID = "sepolia-1"
	MAINNET_CHAIN_ID = "andromeda"
)

const SEPOLIA_SEQUENCER_SET_URL = "https://sepolia-subgraph.metisdevops.link/subgraphs/name/metisio/sequencer-set"
const SEPOLIA_L2_RPC_URL = "https://sepolia.metisdevops.link"

const MAINNET_SEQUENCER_SET_URL = "https://andromeda-subgraph.metisdevops.link/subgraphs/name/metisio/sequencer-set"
const MAINNET_L2_RPC_URL = "https://andromeda.metis.io"

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
			seqInfo.Alerts.Lark = cfg.Lark
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
				Lark:      cfg.Lark,
			},
		})
	client.Sequencers[MetisianName] = &manager

	client.Nodes = cfg.NodeInfos
	if cfg.ChainId == MAINNET_CHAIN_ID {
		client.ChainId = cfg.ChainId
		client.SequencerSetUrl = MAINNET_SEQUENCER_SET_URL
		client.L2RpcUrl = MAINNET_L2_RPC_URL
	} else if cfg.ChainId == SEPOLIA_CHAIN_ID {
		client.ChainId = cfg.ChainId
		client.SequencerSetUrl = SEPOLIA_SEQUENCER_SET_URL
		client.L2RpcUrl = SEPOLIA_L2_RPC_URL
	} else {
		return nil, errors.New(fmt.Sprintf("chain id doesn't matched. you should set either %s or %s", MAINNET_CHAIN_ID, SEPOLIA_CHAIN_ID))
	}

	client.NodeDownMin = cfg.NodeDownMin
	client.NodeDownSeverity = cfg.NodeDownSeverity
	client.Stalled = cfg.Stalled
	client.StalledAlerts = cfg.StalledAlerts
	client.AlertIfNoServers = cfg.AlertIfNoServers
	client.EnableDash = cfg.EnableDash
	client.Listen = cfg.Listen
	client.HideLogs = cfg.HideLogs

	sf, e := os.OpenFile(cfg.StateFile, os.O_RDONLY, 0600)
	if e != nil {
		log.Warn(e.Error())
	}
	b, e := io.ReadAll(sf)
	_ = sf.Close()
	if e != nil {
		log.Warn(e.Error())
	}
	saved := &savedState{}
	e = json.Unmarshal(b, saved)
	if e != nil {
		log.Warn(e.Error())
	}
	for seq, blocks := range saved.Blocks {
		if client.Sequencers[seq] != nil {
			client.Sequencers[seq].blocksResults = blocks
		}
	}

	for seq, seqData := range saved.Sequencers {
		if client.Sequencers[seq] != nil {
			client.Sequencers[seq].statSeqData = &seqData
		}
	}

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
					e = notifyLark(msg)
					if e != nil {
						log.ErrorDynamicArgs(msg.sequencer, "error sending alert to lark", e.Error())
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

type MetisClient struct {
	rpcUrl       string
	wsConn       *websocket.Conn
	seqSetClient graphql.Client
	l2RpcUrl     string
}

func NewMetisClient(nodeInfo NodeInfo, c *MetisianClient) (*MetisClient, error) {
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

	mc.seqSetClient = *graphql.NewClient(c.SequencerSetUrl, graphql.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
	mc.l2RpcUrl = c.L2RpcUrl

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

func (c *MetisianClient) GetEthBlockNumber() (int64, error) {
	jsonBody := map[string]interface{}{
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
		"jsonrpc": "2.0",
	}

	body, err := json.Marshal(jsonBody)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest("POST", c.L2RpcUrl, bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	var resBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &resBody); err != nil {
		return 0, err
	}

	resultHex, ok := resBody["result"].(string)
	if !ok {
		return 0, err
	}

	resultInt, err := strconv.ParseInt(resultHex[2:], 16, 64) // Strip "0x" and convert
	if err != nil {
		return 0, err
	}

	return resultInt, nil
}

// savedState is dumped to a JSON file at exit time, and is loaded at start. If successful it will prevent
// duplicate alerts, and will show old blocks in the dashboard.
type savedState struct {
	Alarms     *alarmCache          `json:"alarms"`
	Blocks     map[string][]int     `json:"blocks"`
	NodesDown  map[string]time.Time `json:"nodes_down"`
	Sequencers map[string]SeqData   `json:"sequencers"`
}

func (c *MetisianClient) SaveOnExit(stateFile string, saved chan interface{}) {
	quitting := make(chan os.Signal, 1)
	signal.Notify(quitting, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	saveState := func() {
		defer close(saved)
		log.Info("saving state...")
		//#nosec -- variable specified on command line
		f, e := os.OpenFile(stateFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if e != nil {
			log.Error(e)
			return
		}
		c.seqMux.Lock()
		defer c.seqMux.Unlock()
		blocks := make(map[string][]int)
		// only need to save counts if the dashboard exists
		if c.EnableDash {
			for k, v := range c.Sequencers {
				blocks[k] = v.blocksResults
			}
		}
		nodesDown := make(map[string]time.Time)
		for _, node := range c.Nodes {
			if node.down {
				if nodesDown == nil {
					nodesDown = make(map[string]time.Time)
				}
				nodesDown[node.RpcURL] = node.downSince
			}
		}

		sequencers := make(map[string]SeqData)
		for _, seq := range c.Sequencers {

			stat := seq.statSeqData
			if stat == nil {
				stat = seq.statNewSeqData
				if stat == nil {
					continue
				}
			}
			sequencers[seq.name] = *stat
		}

		b, e := json.Marshal(&savedState{
			Alarms:     alarms,
			Blocks:     blocks,
			NodesDown:  nodesDown,
			Sequencers: sequencers,
		})
		if e != nil {
			log.Error(e)
			return
		}
		_, _ = f.Write(b)
		_ = f.Close()
		log.Info("Metisian exiting.")
	}
	for {
		select {
		case <-c.Ctx.Done():
			saveState()
			return
		case <-quitting:
			saveState()
			c.Cancel()
			return
		}
	}
}
