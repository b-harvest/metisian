package dash

type SequencerStatus struct {
	MsgType      string `json:"msgType"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	Jailed       bool   `json:"jailed"`
	ActiveAlerts int    `json:"active_alerts"`
	LastError    string `json:"last_error"`

	IsProducing bool `json:"is_producing"`

	Epochs []int64 `json:"epochs"`
	Blocks []int   `json:"blocks"`
}

type LogMessage struct {
	MsgType string `json:"msgType"`
	Ts      int64  `json:"ts"`
	Msg     string `json:"msg"`
}
