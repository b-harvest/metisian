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

	NodeInfos []NodeInfo `toml:"node_infos"`
	ChainId   string     `toml:"chain_id"`

	// Pagerduty configuration values
	Pagerduty PDConfig `toml:"pagerduty"`
	// Discord webhook information
	Discord DiscordConfig `toml:"discord"`
	// Telegram api information
	Telegram TeleConfig `toml:"telegram"`
	// Slack webhook information
	Slack SlackConfig `toml:"slack"`

	Alerts AlertConfig `toml:"managerAlerts"`
}

type AlertConfig struct {
	// How many missed blocks are acceptable before alerting
	ConsecutiveMissed int `yaml:"consecutive_missed"`
	// Tag for pagerduty to set the alert priority
	ConsecutivePriority string `yaml:"consecutive_priority"`
	// Whether to alert on consecutive missed blocks
	ConsecutiveAlerts bool `yaml:"consecutive_enabled"`

	// Window is how many blocks missed as a percentage of the slashing window to trigger an alert
	Window int `yaml:"percentage_missed"`
	// PercentagePriority is a tag for pagerduty to route on priority
	PercentagePriority string `yaml:"percentage_priority"`
	// PercentageAlerts is whether to alert on percentage based misses
	PercentageAlerts bool `yaml:"percentage_enabled"`

	// AlertIfInactive decides if tenderduty send an alert if the validator is not in the active set?
	AlertIfInactive bool `yaml:"alert_if_inactive"`

	// PagerdutyAlerts: Should pagerduty alerts be sent for this sequencer? Both 'config.pagerduty.enabled: yes' and this must be set.
	//Deprecated: use Pagerduty.Enabled instead
	PagerdutyAlerts bool `yaml:"pagerduty_alerts"`
	// DiscordAlerts: Should discord alerts be sent for this sequencer? Both 'config.discord.enabled: yes' and this must be set.
	//Deprecated: use Discord.Enabled instead
	DiscordAlerts bool `yaml:"discord_alerts"`
	// TelegramAlerts: Should telegram alerts be sent for this sequencer? Both 'config.telegram.enabled: yes' and this must be set.
	//Deprecated: use Telegram.Enabled instead
	TelegramAlerts bool `yaml:"telegram_alerts"`

	// sequencer specific overrides for alert destinations.
	// Pagerduty configuration values
	Pagerduty PDConfig `yaml:"pagerduty"`
	// Discord webhook information
	Discord DiscordConfig `yaml:"discord"`
	// Telegram webhook information
	Telegram TeleConfig `yaml:"telegram"`
	// Slack webhook information
	Slack SlackConfig `yaml:"slack"`
}

// PDConfig is the information required to send alerts to PagerDuty
type PDConfig struct {
	Enabled         bool   `yaml:"enabled"`
	ApiKey          string `yaml:"api_key"`
	DefaultSeverity string `yaml:"default_severity"`
}

// DiscordConfig holds the information needed to publish to a Discord webhook for sending alerts
type DiscordConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Webhook  string   `yaml:"webhook"`
	Mentions []string `yaml:"mentions"`
}

// TeleConfig holds the information needed to publish to a Telegram webhook for sending alerts
type TeleConfig struct {
	Enabled  bool     `yaml:"enabled"`
	ApiKey   string   `yaml:"api_key"`
	Channel  string   `yaml:"channel"`
	Mentions []string `yaml:"mentions"`
}

// SlackConfig holds the information needed to publish to a Slack webhook for sending alerts
type SlackConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Webhook  string   `yaml:"webhook"`
	Mentions []string `yaml:"mentions"`
}

type SequencerInfo struct {
	// Sequencer's address you'll watch. (ex. 0x81fc9d26d6b234f9cc6a84bcfefc679cb64a227a)
	Address string `toml:"address"`
	Name    string `toml:"Name"`

	// Alerts defines the types of alerts to send for this sequencer.
	Alerts AlertConfig `yaml:"alerts"`
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
