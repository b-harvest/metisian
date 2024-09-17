package metis

import (
	"errors"
	"fmt"
	"github.com/b-harvest/metisian/util"
	"github.com/pelletier/go-toml/v2"
	"os"
	"strings"
)

type Config struct {
	// What metisian watching
	Sequencers []SequencerInfo `toml:"sequencers"`

	// NodeDownMin controls how long we wait before sending an alert that a node is not responding or has
	// fallen behind.
	NodeDownMin int `toml:"node_down_alert_minutes"`
	// NodeDownSeverity controls the Pagerduty severity when notifying if a node is down.
	NodeDownSeverity string `toml:"node_down_alert_severity"`

	Stalled int `toml:"stalled_minutes"`

	StalledAlerts bool `toml:"stalled_enabled"`

	// AlertIfNoServers: should an alert be sent if no servers are reachable?
	AlertIfNoServers bool `toml:"alert_if_no_servers"`

	NodeInfos []NodeInfo `toml:"node_infos"`
	ChainId   string     `toml:"chain_id"`

	// sequencer specific overrides for alert destinations.
	// Pagerduty configuration values
	Pagerduty PDConfig `toml:"pagerduty"`
	// Discord webhook information
	Discord DiscordConfig `toml:"discord"`
	// Telegram webhook information
	Telegram TeleConfig `toml:"telegram"`
	// Slack webhook information
	Slack SlackConfig `toml:"slack"`

	// EnableDash enables the web dashboard
	EnableDash bool `toml:"enable_dashboard"`
	// Listen is the URL for the dashboard to listen on, must be a valid/parsable URL
	Listen string `toml:"listen_port"`
	// HideLogs controls whether logs are sent to the dashboard. It will also suppress many alarm details.
	// This is useful if the dashboard will be public.
	HideLogs bool `toml:"hide_logs"`
}

type AlertConfig struct {

	// How many missed blocks are acceptable before alerting
	ConsecutiveMissed int `toml:"consecutive_missed"`
	// Tag for pagerduty to set the alert priority
	ConsecutivePriority string `toml:"consecutive_priority"`
	// Whether to alert on consecutive missed blocks
	ConsecutiveAlerts bool `toml:"consecutive_enabled"`

	// If true, this sequencer will use parent alert configuration.
	//
	// e.g)
	// chain_id = "sepolia-1"
	//
	// [telegram]
	// enable = true
	// ...
	// [[sequencers]]
	// use_parent = true
	UseParent bool `toml:"use_parent"`

	NotifyMining bool `toml:"notify_mining"`

	// sequencer specific overrides for alert destinations.
	// Pagerduty configuration values
	Pagerduty PDConfig `toml:"pagerduty"`
	// Discord webhook information
	Discord DiscordConfig `toml:"discord"`
	// Telegram webhook information
	Telegram TeleConfig `toml:"telegram"`
	// Slack webhook information
	Slack SlackConfig `toml:"slack"`
}

// PDConfig is the information required to send alerts to PagerDuty
type PDConfig struct {
	Enabled         bool   `toml:"enabled"`
	ApiKey          string `toml:"api_key"`
	DefaultSeverity string `toml:"default_severity"`
}

// DiscordConfig holds the information needed to publish to a Discord webhook for sending alerts
type DiscordConfig struct {
	Enabled  bool     `toml:"enabled"`
	Webhook  string   `toml:"webhook"`
	Mentions []string `toml:"mentions"`
}

// TeleConfig holds the information needed to publish to a Telegram webhook for sending alerts
type TeleConfig struct {
	Enabled  bool     `toml:"enabled"`
	ApiKey   string   `toml:"api_key"`
	Channel  string   `toml:"channel"`
	Mentions []string `toml:"mentions"`
}

// SlackConfig holds the information needed to publish to a Slack webhook for sending alerts
type SlackConfig struct {
	Enabled  bool     `toml:"enabled"`
	Webhook  string   `toml:"webhook"`
	Mentions []string `toml:"mentions"`
}

type SequencerInfo struct {
	// Sequencer's address you'll watch. (ex. 0x81fc9d26d6b234f9cc6a84bcfefc679cb64a227a)
	Address string `toml:"address"`
	Name    string `toml:"Name"`

	// Alerts defines the types of alerts to send for this sequencer.
	Alerts AlertConfig `toml:"alerts"`
}

func LoadConfig(filePath, token string) (*Config, error) {

	cfg := &Config{}
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		contents, err := util.FetchRemoteFile(filePath, token)
		if err != nil || len(contents) < 1 {
			return nil, errors.New(fmt.Sprintf("Didn't find any files. check if there exists content - %v", err))
		}
		err = toml.Unmarshal([]byte(contents[0]), cfg)
		if err != nil {
			return nil, err
		}
	} else {
		f, e := os.OpenFile(filePath, os.O_RDONLY, 0600)
		if e != nil {
			return nil, e
		}
		i, e := f.Stat()
		if e != nil {
			_ = f.Close()
			return nil, e
		}
		b := make([]byte, int(i.Size()))
		_, e = f.Read(b)
		_ = f.Close()
		if e != nil {
			return nil, e
		}
		e = toml.Unmarshal(b, cfg)
		if e != nil {
			return nil, e
		}
	}

	return cfg, nil
}
