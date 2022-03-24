package rpczmq

import (
	"context"
	"github.com/superisaac/jsonz"
)

type MQItem struct {
	ID      string `json:"id"`
	Brief   string `json:"brief"`
	Kind    string `json:"kind"`
	MsgData []byte `json:"msgdata"`
}

type MQChunk struct {
	Items  []MQItem `json:"items"`
	NextID string   `json:"nextID"`
}

type MQClient interface {
	// append an item to MQ
	Add(ctx context.Context, section string, ntf *jsonz.NotifyMessage) (string, error)

	// Get a trunk given prevID
	Chunk(ctx context.Context, section string, prevID string, count int64) (MQChunk, error)

	// Get the tail chunk of queue
	Tail(ctx context.Context, section string, count int64) (MQChunk, error)

	// Subscribe to change of queue
	Subscribe(rootctx context.Context, section string, callback func(item MQItem)) error
}
