package smtc

import "fmt"

type Mock struct{}

func NewMock() *Mock {
	return &Mock{}
}

func (m *Mock) GetData() (SMTCData, error) {
	return SMTCData{
		Status:       "Active",
		Title:        "Demo Song",
		Artist:       "Demo Artist",
		AlbumTitle:   "Demo Album",
		PositionMs:   12340,
		DurationMs:   180000,
		HasSession:   true,
	}, nil
}

func init() {
	fmt.Println("[SMTC] Mock backend initialized")
}