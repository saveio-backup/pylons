/**
 * Description:
 * Author: Yihen.Liu
 * Create: 2018-11-23
 */
package test

import (
	"fmt"
	"testing"

	"github.com/oniio/oniChannel/network/rpc"
)

func TestScanSpecialBlockByTag(t *testing.T) {
	fmt.Println("hello world")
	rpcClient := rpc.NewRpcClient("http://127.0.0.1:20336")
	height, err := rpcClient.GetCurrentBlockHeight()
	if err != nil {
		fmt.Printf("%s\n", err)
	} else {
		fmt.Printf("height:%s\n", height)
	}
	_height := make(chan uint32)
	rpcClient.ScanSpecialBlockByTag("hello", _height)
}
