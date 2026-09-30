package common

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bitcoinz-xyz/lightwalletd/walletrpc"
)

// A quiet mempool (no new block, no txs) must get an empty heartbeat every
// MempoolHeartbeatInterval, and the stream must end once the client is gone.
func TestMempoolHeartbeat(t *testing.T) {
	RawRequest = func(method string, params []json.RawMessage) (json.RawMessage, error) {
		switch method {
		case "getblockchaininfo":
			return json.Marshal(&BitcoinZdRpcReplyGetblockchaininfo{BestBlockHash: "quiet", Blocks: 300})
		case "getrawmempool":
			return json.Marshal([]string{})
		}
		t.Fatalf("unexpected rpc %s", method)
		return nil, nil
	}
	Time.Sleep = sleepStub
	Time.Now = nowStub
	sleepDuration = 5000 * time.Second
	g_lastBlockChainInfo = &BitcoinZdRpcReplyGetblockchaininfo{BestBlockHash: "quiet"}
	g_lastTime = time.Time{}
	g_txList = nil

	ctx, cancel := context.WithCancel(context.Background())
	start := nowStub()
	var beats []time.Duration
	err := GetMempool(ctx, func(tx *walletrpc.RawTransaction) error {
		if len(tx.Data) != 0 {
			t.Fatal("heartbeat carries data")
		}
		beats = append(beats, nowStub().Sub(start))
		if len(beats) == 3 {
			cancel()
		}
		return nil
	})
	if err != context.Canceled {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if len(beats) != 3 {
		t.Fatalf("want 3 heartbeats, got %d", len(beats))
	}
	for i, b := range beats {
		want := time.Duration(i+1) * MempoolHeartbeatInterval
		if b < want || b > want+time.Second {
			t.Fatalf("heartbeat %d at %v, want ~%v", i+1, b, want)
		}
	}
}
