package common

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/bitcoinz-xyz/lightwalletd/walletrpc"
)

type txid string

// MempoolPollInterval controls how often mempool is refreshed.
// Set by cmd/root.go from CLI flags.
var MempoolPollInterval = 2 * time.Second

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

func GetMempool(sendToClient func(*walletrpc.RawTransaction) error) error {
	g_lock.Lock()
	index := 0
	// Stay in this function until the tip block hash changes.
	stayHash := g_lastBlockChainInfo.BestBlockHash

	// Wait for more transactions to be added to the list
	for {
		now := Time.Now()
		if now.After(g_lastTime.Add(MempoolPollInterval)) {
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
		Time.Sleep(200 * time.Millisecond)
		g_lock.Lock()
		if g_lastBlockChainInfo.BestBlockHash != stayHash {
			break
		}
	}
	g_lock.Unlock()
	return nil
}

// fetchResult holds the result of a parallel mempool transaction fetch.
type fetchResult struct {
	txidStr string
	rawtx   *walletrpc.RawTransaction
	err     error
}

// refreshMempoolTxns gets all new mempool txns and sends any new ones to waiting clients.
// Uses a small worker pool to fetch transactions in parallel.
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

	// Collect new txids to fetch
	var newTxids []string
	for _, txidstr := range mempoolList {
		if _, ok := g_txidSeen[txid(txidstr)]; ok {
			continue
		}
		g_txidSeen[txid(txidstr)] = struct{}{}
		newTxids = append(newTxids, txidstr)
	}

	if len(newTxids) == 0 {
		return nil
	}

	// Fetch new transactions in parallel with a small worker pool
	const mempoolWorkers = 3
	resultChan := make(chan fetchResult, len(newTxids))
	sem := make(chan struct{}, mempoolWorkers)
	var wg sync.WaitGroup

	for _, txidstr := range newTxids {
		wg.Add(1)
		sem <- struct{}{}
		go func(tid string) {
			defer wg.Done()
			defer func() { <-sem }()

			txidJSON, err := json.Marshal(tid)
			if err != nil {
				resultChan <- fetchResult{txidStr: tid, err: err}
				return
			}
			params := []json.RawMessage{txidJSON, json.RawMessage("1")}
			result, rpcErr := RawRequest("getrawtransaction", params)
			if rpcErr != nil {
				// Not an error; mempool transactions can disappear
				resultChan <- fetchResult{txidStr: tid}
				return
			}
			rawtx, err := ParseRawTransaction(result)
			if err != nil {
				resultChan <- fetchResult{txidStr: tid, err: err}
				return
			}
			resultChan <- fetchResult{txidStr: tid, rawtx: rawtx}
		}(txidstr)
	}
	wg.Wait()
	close(resultChan)

	// Process results in order they come in (mempool order doesn't matter much)
	for res := range resultChan {
		if res.err != nil {
			return res.err
		}
		if res.rawtx == nil {
			continue
		}
		// Skip any transaction that has been mined since the list of txids
		// was retrieved.
		if res.rawtx.Height != 0 {
			continue
		}
		g_txList = append(g_txList, res.rawtx)
	}
	return nil
}
