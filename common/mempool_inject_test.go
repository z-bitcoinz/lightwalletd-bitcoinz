package common

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bitcoinz-xyz/lightwalletd/walletrpc"
)

// A transaction injected by SendTransaction must reach a quiet stream on the
// next poll, reset the heartbeat timer, and not be sent again when the same
// txid later shows up in getrawmempool.
func TestMempoolInjectHeartbeat(t *testing.T) {
	injected := false
	RawRequest = func(method string, params []json.RawMessage) (json.RawMessage, error) {
		switch method {
		case "getblockchaininfo":
			return json.Marshal(&BitcoinZdRpcReplyGetblockchaininfo{BestBlockHash: "quiet", Blocks: 300})
		case "getrawmempool":
			if injected {
				return json.Marshal([]string{"injectedtxid"})
			}
			return json.Marshal([]string{})
		}
		t.Fatalf("unexpected rpc %s", method)
		return nil, nil
	}
	sleepDuration = 5000 * time.Second
	start := nowStub()
	Time.Sleep = func(d time.Duration) {
		sleepStub(d)
		if !injected && nowStub().Sub(start) >= 90*time.Second {
			injected = true
			InjectTransaction("injectedtxid", &walletrpc.RawTransaction{Data: []byte{1, 2, 3}})
		}
	}
	Time.Now = nowStub
	g_lastBlockChainInfo = &BitcoinZdRpcReplyGetblockchaininfo{BestBlockHash: "quiet"}
	g_lastTime = time.Time{}
	g_txidSeen = map[txid]struct{}{}
	g_txList = nil

	type event struct {
		at   time.Duration
		data int
	}
	var events []event
	ctx, cancel := context.WithCancel(context.Background())
	err := GetMempool(ctx, func(tx *walletrpc.RawTransaction) error {
		events = append(events, event{nowStub().Sub(start), len(tx.Data)})
		if len(events) == 4 {
			cancel()
		}
		return nil
	})
	if err != context.Canceled {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	// heartbeat @60s, injected tx @~90s, heartbeats @~150s and @~210s
	want := []event{{60 * time.Second, 0}, {90 * time.Second, 3}, {150 * time.Second, 0}, {210 * time.Second, 0}}
	for i, w := range want {
		e := events[i]
		if e.data != w.data || e.at < w.at || e.at > w.at+time.Second {
			t.Fatalf("event %d = %+v, want ~%+v (all: %+v)", i, e, w, events)
		}
	}
	Time.Sleep = sleepStub
}
