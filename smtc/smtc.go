package smtc

type SMTCData struct {
	Status       string `json:"status"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	AlbumTitle   string `json:"albumTitle,omitempty"`
	PositionMs   int64  `json:"positionMs"`
	DurationMs   int64  `json:"durationMs,omitempty"`
	HasSession   bool   `json:"hasSession"`
}

type SMTC interface {
	GetData() (SMTCData, error)
}
