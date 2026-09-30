package common

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/bitcoinz-xyz/lightwalletd/walletrpc"
)

type txid string

// MempoolHeartbeatInterval: if nothing has been sent to a GetMempoolStream
// client for this long, send it an empty RawTransaction. Clients give up on a
// silent stream (the Z-Text wallet after 120s, Cloudflare's proxy after 125s)
// and BTCZ blocks are ~155s apart, so without this the stream is often cut
// before the new block that is supposed to close it. Clients skip the empty
// message because its data does not parse as a transaction. Zero disables it.
var MempoolHeartbeatInterval = 60 * time.Second

var (
	// Set of mempool txids that have been seen during the current block interval.
	// The zcashd RPC `getrawmempool` returns the entire mempool each time, so
	// this allows us to ignore the txids that we've already seen.
	g_txidSeen map[txid]struct{} = map[txid]struct{}{}

	// List of transactions during current block interval, in order received. Each
	// client thread can keep an index into this slice to record which transactions
	// it's sent back to the client (everything before that index). The g_txidSeen
	// map allows this list to not contain duplicates.
	g_txList []*walletrpc.RawTransaction

	// The most recent absolute time that we fetched the mempool and the latest
	// (tip) block hash (so we know when a new block has been mined).
	g_lastTime time.Time

	// The most recent zcashd getblockchaininfo reply, for height and best block
	// hash (tip) which is used to detect when a new block arrives.
	g_lastBlockChainInfo *BitcoinZdRpcReplyGetblockchaininfo = &BitcoinZdRpcReplyGetblockchaininfo{}

	// Mutex to protect the above variables.
	g_lock sync.Mutex
)

// InjectTransaction adds a locally-submitted transaction directly into the
// mempool tracker so that connected GetMempoolStream clients see it immediately,
// without waiting for the next getrawmempool poll cycle.
func InjectTransaction(txidStr string, rawtx *walletrpc.RawTransaction) {
	g_lock.Lock()
	defer g_lock.Unlock()
	if _, ok := g_txidSeen[txid(txidStr)]; ok {
		return // already tracked
	}
	g_txidSeen[txid(txidStr)] = struct{}{}
	g_txList = append(g_txList, rawtx)
	Log.Debugf("Injected tx %s into mempool tracker for immediate streaming\n", txidStr)
}

func GetMempool(ctx context.Context, sendToClient func(*walletrpc.RawTransaction) error) error {
	g_lock.Lock()
	index := 0
	lastSent := Time.Now()
	// Stay in this function until the tip block hash changes.
	stayHash := g_lastBlockChainInfo.BestBlockHash

	// Wait for more transactions to be added to the list
	for {
		// Don't fetch the mempool more often than every 2 seconds.
		now := Time.Now()
		if now.After(g_lastTime.Add(2 * time.Second)) {
			blockChainInfo, err := GetBlockChainInfo()
			if err != nil {
				g_lock.Unlock()
				return err
			}
			if g_lastBlockChainInfo.BestBlockHash != blockChainInfo.BestBlockHash {
				// A new block has arrived
				g_lastBlockChainInfo = blockChainInfo
				// We're the first thread to notice, clear cached state.
				g_txidSeen = map[txid]struct{}{}
				g_txList = []*walletrpc.RawTransaction{}
				g_lastTime = time.Time{}
				break
			}
			if err = refreshMempoolTxns(); err != nil {
				g_lock.Unlock()
				return err
			}
			g_lastTime = now
		}
		// Send transactions we haven't sent yet, best to not do so while
		// holding the mutex, since this call may get flow-controlled.
		toSend := g_txList[index:]
		index = len(g_txList)
		g_lock.Unlock()
		for _, tx := range toSend {
			if err := sendToClient(tx); err != nil {
				return err
			}
		}
		if len(toSend) > 0 {
			lastSent = Time.Now()
		} else if MempoolHeartbeatInterval > 0 && Time.Now().Sub(lastSent) >= MempoolHeartbeatInterval {
			if err := sendToClient(&walletrpc.RawTransaction{}); err != nil {
				return err
			}
			lastSent = Time.Now()
		}
		// The client went away: stop now instead of polling until the next block.
		if err := ctx.Err(); err != nil {
			return err
		}
		Time.Sleep(200 * time.Millisecond)
		g_lock.Lock()
		if g_lastBlockChainInfo.BestBlockHash != stayHash {
			break
		}
	}
	g_lock.Unlock()
	return nil
}

// RefreshMempoolTxns gets all new mempool txns and sends any new ones to waiting clients
func refreshMempoolTxns() error {
	params := []json.RawMessage{}
	result, rpcErr := RawRequest("getrawmempool", params)
	if rpcErr != nil {
		return rpcErr
	}
	var mempoolList []string
	err := json.Unmarshal(result, &mempoolList)
	if err != nil {
		return err
	}

	// Fetch all new mempool txns and add them into `newTxns`
	for _, txidstr := range mempoolList {
		if _, ok := g_txidSeen[txid(txidstr)]; ok {
			// We've already fetched this transaction
			continue
		}

		// We haven't fetched this transaction already.
		g_txidSeen[txid(txidstr)] = struct{}{}
		txidJSON, err := json.Marshal(txidstr)
		if err != nil {
			return err
		}

		params := []json.RawMessage{txidJSON, json.RawMessage("1")}
		result, rpcErr := RawRequest("getrawtransaction", params)
		if rpcErr != nil {
			// Not an error; mempool transactions can disappear
			continue
		}

		rawtx, err := ParseRawTransaction(result)
		if err != nil {
			return err
		}

		// Skip any transaction that has been mined since the list of txids
		// was retrieved.
		if (rawtx.Height != 0) {
			continue;
		}

		g_txList = append(g_txList, rawtx)
	}
	return nil
}
