package rpcmapmq

import (
	"context"
	"github.com/superisaac/jsonz"
)

type MQItem struct {
	Offset  string `json:"offset"`
	Brief   string `json:"brief"`
	Kind    string `json:"kind"`
	MsgData []byte `json:"msgdata"`
}

type MQChunk struct {
	Items      []MQItem `json:"items"`
	LastOffset string   `json:"lastoffset"`
}

type MQClient interface {
	// append an item to MQ
	Add(ctx context.Context, section string, ntf *jsonz.NotifyMessage) (string, error)

	// Get a trunk given last offset
	Chunk(ctx context.Context, section string, lastOffset string, count int64) (MQChunk, error)

	// Get the tail chunk of queue, aka queue[-count:]
	Tail(ctx context.Context, section string, count int64) (MQChunk, error)

	// Subscribe to change of queue
	Subscribe(rootctx context.Context, section string, output chan MQItem) error
}
