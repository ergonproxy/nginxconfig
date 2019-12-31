package engine

type nginxResponse struct {
	Verstion      string `json:"version"`
	Build         string `json:"build"`
	Address       string `json:"address"`
	Generation    string `json:"generation"`
	LoadTimestamp string `json:"load_timestamp"`
	Timestamp     string `json:"timestamp"`
	PID           int    `json:"pid"`
	PPID          int    `json:"ppid"`
}

type processesResponse struct {
	Respawned int64 `json:"respawned"`
}
