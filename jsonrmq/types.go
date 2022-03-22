package jsonrmq

type MQItem struct {
	ID      string `json:"id"`
	Brief   string `json:"brief"`
	Kind    string `json:"kind"`
	MsgData []byte `json:"msgdata"`
}

type MQRange struct {
	Items  []MQItem `json:"items"`
	NextID string   `json:"nextID"`
}
