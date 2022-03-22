package jsonrmq

type MQItem struct {
	ID string
	Brief  string
	Kind string
	Body []byte
}

type MQRange struct {
	Items []MQItem
	NextID string
}
