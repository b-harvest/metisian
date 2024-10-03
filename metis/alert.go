package metis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PagerDuty/go-pagerduty"
	log "github.com/b-harvest/metisian/log"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/r3labs/diff"
	"net/http"
	"strings"
	"sync"
	"time"
)

type alertMsg struct {
	pd   bool
	disc bool
	tg   bool
	slk  bool

	severity  string
	resolved  bool
	sequencer string
	message   string
	uniqueId  string
	key       string

	tgChannel  string
	tgKey      string
	tgMentions string

	discHook     string
	discMentions string

	slkHook     string
	slkMentions string
}

type notifyDest uint8

const (
	pd notifyDest = iota
	tg
	di
	slk
)

func (a *alarmCache) clearNoBlocks(seqeuncer string) {
	if a.AllAlarms == nil || a.AllAlarms[seqeuncer] == nil {
		return
	}
	a.notifyMux.Lock()
	defer a.notifyMux.Unlock()
	for clearAlarm := range a.AllAlarms[seqeuncer] {
		if strings.HasPrefix(clearAlarm, "stalled: have not seen a new block on") {
			delete(a.AllAlarms[seqeuncer], clearAlarm)
		}
	}
}

func (a *alarmCache) getCount(chain string) int {
	if a.AllAlarms == nil || a.AllAlarms[chain] == nil {
		return 0
	}
	a.notifyMux.RLock()
	defer a.notifyMux.RUnlock()
	return len(a.AllAlarms[chain])
}

func (a *alarmCache) clearAll(chain string) {
	if a.AllAlarms == nil || a.AllAlarms[chain] == nil {
		return
	}
	a.notifyMux.Lock()
	defer a.notifyMux.Unlock()
	a.AllAlarms[chain] = make(map[string]time.Time)
}

// alarms is used to prevent double notifications.
var alarms = &alarmCache{
	SentPdAlarms:   make(map[string]time.Time),
	SentTgAlarms:   make(map[string]time.Time),
	SentDiAlarms:   make(map[string]time.Time),
	SentSlkAlarms:  make(map[string]time.Time),
	AllAlarms:      make(map[string]map[string]time.Time),
	flappingAlarms: make(map[string]map[string]time.Time),
	notifyMux:      sync.RWMutex{},
}

func shouldNotify(msg *alertMsg, dest notifyDest) bool {
	alarms.notifyMux.Lock()
	defer alarms.notifyMux.Unlock()
	var whichMap map[string]time.Time
	var service string
	if alarms.AllAlarms[msg.sequencer] == nil {
		alarms.AllAlarms[msg.sequencer] = make(map[string]time.Time)
	}
	switch dest {
	case pd:
		whichMap = alarms.SentPdAlarms
		service = "PagerDuty"
	case tg:
		whichMap = alarms.SentTgAlarms
		service = "Telegram"
	case di:
		whichMap = alarms.SentDiAlarms
		service = "Discord"
	case slk:
		whichMap = alarms.SentSlkAlarms
		service = "Slack"
	}

	switch {
	case !whichMap[msg.sequencer+msg.message].IsZero() && !msg.resolved:
		// already sent this alert
		return false
	case !whichMap[msg.sequencer+msg.message].IsZero() && msg.resolved:
		// alarm is cleared
		delete(whichMap, msg.sequencer+msg.message)
		log.Info(fmt.Sprintf("💜 Resolved     alarm on %20s (%s) - notifying %s", msg.sequencer, msg.message, service))
		return true
	case msg.resolved:
		// it looks like we got a duplicate resolution or suppressed it. Note it and move on:
		log.Error(errors.New(fmt.Sprintf("😕 Not clearing alarm on %20s (%s) - no corresponding alert %s", msg.sequencer, msg.message, service)))
		return false
	}

	// check if the alarm is flapping, if we sent the same alert in the last five minutes, show a warning but don't alert
	if alarms.flappingAlarms[msg.sequencer] == nil {
		alarms.flappingAlarms[msg.sequencer] = make(map[string]time.Time)
	}

	// for pagerduty we perform some basic flap detection
	if dest == pd && msg.pd && alarms.flappingAlarms[msg.sequencer][msg.message].After(time.Now().Add(-5*time.Minute)) {
		log.ErrorDynamicArgs("🛑 flapping detected - suppressing pagerduty notification:", msg.sequencer, msg.message)
		return false
	} else if dest == pd && msg.pd {
		alarms.flappingAlarms[msg.sequencer][msg.message] = time.Now()
	}

	log.Info(fmt.Sprintf("new alarm on %20s (%s) - notifying %s", msg.sequencer, msg.message, service))
	whichMap[msg.sequencer+msg.message] = time.Now()
	return true
}

func notifySlack(msg *alertMsg) (err error) {
	if !msg.slk {
		return
	}
	data, err := json.Marshal(buildSlackMessage(msg))
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", msg.slkHook, bytes.NewBuffer(data))
	if err != nil {
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("could not notify slack for %s got %d response", msg.sequencer, resp.StatusCode)
	}

	return
}

type SlackMessage struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Text      string `json:"text"`
	Color     string `json:"color"`
	Title     string `json:"title"`
	TitleLink string `json:"title_link"`
}

func buildSlackMessage(msg *alertMsg) *SlackMessage {
	prefix := ""
	color := "danger"
	if msg.resolved {
		msg.message = "OK: " + msg.message
		prefix = "💜 Resolved: "
		color = "good"
	}
	return &SlackMessage{
		Text: msg.message,
		Attachments: []Attachment{
			{
				Title: fmt.Sprintf("Metisian %s %s %s", prefix, msg.sequencer, msg.slkMentions),
				Color: color,
			},
		},
	}
}

func notifyDiscord(msg *alertMsg) (err error) {
	if !msg.disc {
		return nil
	}
	if !shouldNotify(msg, di) {
		return nil
	}
	discPost := buildDiscordMessage(msg)
	client := &http.Client{}
	data, err := json.MarshalIndent(discPost, "", "  ")
	if err != nil {
		log.ErrorDynamicArgs("⚠️ Could not notify discord!", err)
		return err
	}

	req, err := http.NewRequest("POST", msg.discHook, bytes.NewBuffer(data))
	if err != nil {
		log.ErrorDynamicArgs("⚠️ Could not notify discord!", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.ErrorDynamicArgs("⚠️ Could not notify discord!", err)
		return err
	}
	_ = resp.Body.Close()

	if resp.StatusCode != 204 {
		log.Info(fmt.Sprintf("%v", resp))
		log.ErrorDynamicArgs("⚠️ Could not notify discord! Returned", resp.StatusCode)
		return err
	}
	return nil
}

type DiscordMessage struct {
	Username  string         `json:"username,omitempty"`
	AvatarUrl string         `json:"avatar_url,omitempty"`
	Content   string         `json:"content"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string `json:"title,omitempty"`
	Url         string `json:"url,omitempty"`
	Description string `json:"description"`
	Color       uint   `json:"color"`
}

func buildDiscordMessage(msg *alertMsg) *DiscordMessage {
	prefix := ""
	if msg.resolved {
		prefix = "💜 Resolved: "
	}
	return &DiscordMessage{
		Username: "Metisian",
		Content:  prefix + msg.sequencer,
		Embeds: []DiscordEmbed{{
			Description: msg.message,
		}},
	}
}

func notifyTg(msg *alertMsg) (err error) {
	if !msg.tg {
		return nil
	}
	if !shouldNotify(msg, tg) {
		return nil
	}
	bot, err := tgbotapi.NewBotAPI(msg.tgKey)
	if err != nil {
		log.ErrorDynamicArgs("notify telegram:", err)
		return
	}

	prefix := ""
	if msg.resolved {
		prefix = "💜 Resolved: "
	}

	mc := tgbotapi.NewMessageToChannel(msg.tgChannel, fmt.Sprintf("%s: %s - %s", msg.sequencer, prefix, msg.message))
	_, err = bot.Send(mc)
	if err != nil {
		log.ErrorDynamicArgs("telegram send:", err)
	}
	return err
}

func notifyPagerduty(msg *alertMsg) (err error) {
	if !msg.pd {
		return nil
	}
	if !shouldNotify(msg, pd) {
		return nil
	}
	// key from the example, don't spam their api
	if msg.key == "aaaaaaaaaaaabbbbbbbbbbbbbcccccccccccc" {
		log.Error(errors.New("invalid pagerduty key"))
		return
	}
	action := "trigger"
	if msg.resolved {
		action = "resolve"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = pagerduty.ManageEventWithContext(ctx, pagerduty.V2Event{
		RoutingKey: msg.key,
		Action:     action,
		DedupKey:   msg.uniqueId,
		Payload: &pagerduty.V2Payload{
			Summary:  msg.message,
			Source:   msg.uniqueId,
			Severity: msg.severity,
		},
	})
	return
}

func getAlarms(chain string) string {
	alarms.notifyMux.RLock()
	defer alarms.notifyMux.RUnlock()
	// don't show this info if the logs are disabled on the dashboard, potentially sensitive info could be leaked.
	result := ""
	for k := range alarms.AllAlarms[chain] {
		result += "🚨 " + k + "\n"
	}
	return result
}

// alert creates a universal alert and pushes it to the alertChan to be delivered to appropriate services
func (c *MetisianClient) alert(seqName, message, severity string, resolved, notSend bool, id *string) {
	uniq := seqName
	if id != nil {
		uniq = *id
	}

	var seqAlert AlertConfig
	if c.Sequencers[seqName] == nil {
		msg := fmt.Sprintf("No sequencer found with Name: %s", seqName)
		log.Error(errors.New(msg))
		seqAlert = c.Sequencers[MetisianName].Alerts

		message = fmt.Sprintf("%s\ncontent: \n%s", msg, message)
	} else {
		seqAlert = c.Sequencers[seqName].Alerts
	}

	if !notSend {
		c.seqMux.RLock()
		a := &alertMsg{
			pd:           seqAlert.Pagerduty.Enabled,
			disc:         seqAlert.Discord.Enabled,
			tg:           seqAlert.Telegram.Enabled,
			slk:          seqAlert.Slack.Enabled,
			severity:     severity,
			resolved:     resolved,
			sequencer:    seqName,
			message:      message,
			uniqueId:     uniq,
			key:          seqAlert.Pagerduty.ApiKey,
			tgChannel:    seqAlert.Telegram.Channel,
			tgKey:        seqAlert.Telegram.ApiKey,
			tgMentions:   strings.Join(seqAlert.Telegram.Mentions, " "),
			discHook:     seqAlert.Discord.Webhook,
			discMentions: strings.Join(seqAlert.Discord.Mentions, " "),
			slkHook:      seqAlert.Slack.Webhook,
		}
		c.alertChan <- a
		c.seqMux.RUnlock()
	}
	alarms.notifyMux.Lock()
	defer alarms.notifyMux.Unlock()
	if alarms.AllAlarms[seqName] == nil {
		alarms.AllAlarms[seqName] = make(map[string]time.Time)
	}
	if resolved && !alarms.AllAlarms[seqName][message].IsZero() {
		delete(alarms.AllAlarms[seqName], message)
		return
	} else if resolved {
		return
	}
	alarms.AllAlarms[seqName][message] = time.Now()
}

// watch handles monitoring for missed blocks, stalled sequencer, node downtime
func (c *MetisianClient) watch() {
	var (
		noNodes        bool
		missedAlarm    = make(map[string]bool)
		noSequencerSet = make(map[string]bool)
	)

	// Alert if there are no endpoints available
	noNodesSec := 0 // delay a no-nodes alarm for 30 seconds, too noisy.
	for {
		var seq = c.GetAnySequencer()

		if seq.valInfo == nil {
			time.Sleep(time.Second)
			if c.AlertIfNoServers && !noNodes && c.noNodes && noNodesSec >= 60*c.NodeDownMin {
				noNodes = true
				c.alert(
					MetisianName,
					fmt.Sprintf("no RPC endpoints are working"),
					"critical",
					false,
					false,
					nil,
				)
			}
			noNodesSec += 1
			continue
		}
		noNodesSec = 0
		break
	}

	nodeAlarms := make(map[string]bool)

	for {
		time.Sleep(2 * time.Second)

		// alert if we can't monitor
		switch {
		case c.AlertIfNoServers && !noNodes && c.noNodes:
			noNodesSec += 2
			if noNodesSec <= 30*c.NodeDownMin {
				if noNodesSec%20 == 0 {
					log.ErrorDynamicArgs(fmt.Sprintf("no nodes available for %d seconds, deferring alarm", noNodesSec))
				}
				noNodes = false
			} else {
				noNodesSec = 0
				noNodes = true
				c.alert(
					MetisianName,
					fmt.Sprintf("no RPC endpoints are working"),
					"critical",
					false,
					false,
					nil,
				)
			}
		default:
			noNodesSec = 0
		}

		// stalled sequencer detection
		if c.StalledAlerts && !c.lastBlockAlarm && !c.lastBlockTime.IsZero() &&
			c.lastBlockTime.Before(time.Now().Add(time.Duration(-c.Stalled)*time.Minute)) {

			// sequencer is stalled send an alert!
			c.lastBlockAlarm = true
			c.alert(
				MetisianName,
				fmt.Sprintf("🚨 stalled: have not seen a new block in %d minutes", c.Stalled),
				"critical",
				false,
				false,
				nil,
			)
		} else if c.StalledAlerts && c.lastBlockAlarm && c.lastBlockTime.IsZero() {
			c.lastBlockAlarm = false
			c.alert(
				MetisianName,
				fmt.Sprintf("🚨 stalled: have not seen a new block in %d minutes", c.Stalled),
				"info",
				true,
				false,
				nil,
			)
			alarms.clearNoBlocks(MetisianName)
		}

		for _, seq := range c.GetSequencers() {

			// consecutive missed block alarms:
			if !missedAlarm[seq.name] && seq.Alerts.ConsecutiveAlerts && int(seq.statConsecutiveMiss) >= seq.Alerts.ConsecutiveMissed {
				// alert on missed block counter!
				missedAlarm[seq.name] = true
				id := seq.Address + "consecutive"
				c.alert(
					seq.name,
					fmt.Sprintf("🚨 sequencer has missed %d blocks", seq.Alerts.ConsecutiveMissed),
					seq.Alerts.ConsecutivePriority,
					false,
					false,
					&id,
				)
				seq.activeAlerts = alarms.getCount(seq.name)
			} else if missedAlarm[seq.name] && int(seq.statConsecutiveMiss) < seq.Alerts.ConsecutiveMissed {
				// clear the alert
				missedAlarm[seq.name] = false
				id := seq.Address + "consecutive"
				c.alert(
					seq.name,
					fmt.Sprintf("🚨 sequencer has missed %d blocks", seq.Alerts.ConsecutiveMissed),
					"info",
					true,
					false,
					&id,
				)
				seq.activeAlerts = alarms.getCount(seq.name)
			}

			// recommited sequencer alarms:
			if seq.statNewSeqData == nil || len(seq.statNewSeqData.Epoches) == 0 {
				if seq.statSeqData != nil {
					log.Debug(fmt.Sprintf("no epochs detected for this sequencer %20s (%s)", seq.name, seq.Address))
				} // skipping

			} else {
				if seq.statSeqData == nil || len(seq.statSeqData.Epoches) == 0 {
					seq.statSeqData = seq.statNewSeqData
				} else {
					changelog, err := diff.Diff(seq.statSeqData, seq.statNewSeqData)
					if err != nil {
						log.Warn(fmt.Sprintf("cannot diff sequencer-set : %v", err))
					}

					if len(changelog) > 0 {

						if !noSequencerSet[seq.name] && len(seq.statNewSeqData.Epoches) == 0 {
							noSequencerSet[seq.name] = true
							id := seq.Address + "sequencer-set"
							c.alert(
								seq.name,
								fmt.Sprintf("🚨 cannot fetch sequencer info : %20s (%s)", seq.name, seq.Address),
								"warn",
								false,
								false,
								&id)
							seq.activeAlerts = alarms.getCount(seq.name)
						} else if noSequencerSet[seq.name] && len(seq.statNewSeqData.Epoches) > 0 {
							noSequencerSet[seq.name] = false
							id := seq.Address + "sequencer-set"
							c.alert(
								seq.name,
								fmt.Sprintf("🚨 cannot fetch sequencer info : %20s (%s)", seq.name, seq.Address),
								"warn",
								true,
								false,
								&id)
							seq.activeAlerts = alarms.getCount(seq.name)
						}

						if !noSequencerSet[seq.name] {
							// check if sequencer data has removed
							if seq.statNewSeqData.find(seq.statSeqData.Epoches[0].ID) == nil {

								id := seq.Address + "respan"
								c.alert(
									seq.name,
									fmt.Sprintf("❌ sequencer has recommited!! please check your sequencer status"),
									"critical",
									false,
									false,
									&id)

								c.alert(
									seq.name,
									fmt.Sprintf("❌ sequencer has recommited!! please check your sequencer status"),
									"critical",
									true,
									true,
									&id)
								seq.activeAlerts = alarms.getCount(seq.name)
							} else if seq.statSeqData.find(seq.statNewSeqData.Epoches[0].ID) == nil {
								newTask := seq.statNewSeqData.Epoches[0]
								msg := fmt.Sprintf("💎 sequencer has new mining task\t\tspanId: %4v, startBlock: %8s, endBlock: %8s, recommited: %t", newTask.ID, newTask.StartBlock, newTask.EndBlock, newTask.Recommited)
								if seq.Alerts.NotifyMining {
									id := seq.Address + "mining"
									c.alert(
										seq.name,
										msg,
										"info",
										false,
										false,
										&id)

									c.alert(
										seq.name,
										msg,
										"info",
										true,
										true,
										&id)
									seq.activeAlerts = alarms.getCount(seq.name)
								}
							}

							seq.statSeqData = seq.statNewSeqData
						}
					}

				}
			}
		}

		// node down alarms
		for _, node := range c.Nodes {
			// window percentage missed block alarms
			if node.AlertIfDown && node.down && !node.wasDown && !node.downSince.IsZero() &&
				time.Since(node.downSince) > time.Duration(c.NodeDownMin)*time.Minute {
				// alert on dead node
				if nodeAlarms[node.RpcURL] {
					continue
				}
				nodeAlarms[node.RpcURL] = true // used to keep active alert count correct
				c.alert(
					MetisianName,
					fmt.Sprintf("Severity: %s\nRPC node %s has been down for > %d minutes", c.NodeDownSeverity, node.RpcURL, c.NodeDownMin),
					c.NodeDownSeverity,
					false,
					false,
					&node.RpcURL,
				)
			} else if node.AlertIfDown && !node.down && node.wasDown {
				// clear the alert
				nodeAlarms[node.RpcURL] = false
				node.wasDown = false
				c.alert(
					MetisianName,
					fmt.Sprintf("Severity: %s\nRPC node %s has been down for > %d minutes on %s", c.NodeDownSeverity, node.RpcURL, c.NodeDownMin),
					"info",
					true,
					false,
					&node.RpcURL,
				)
			}
		}
	}
}
