// Package jsonrpc privides a function to start json rpc server
package jsonrpc

import (
	"net/http"
	"strconv"

	"fmt"

	"github.com/oniio/oniChannel/api/base/common"
	"github.com/oniio/oniChannel/api/base/rpc"
)

func StartRPCServer() error {
	http.HandleFunc("/", rpc.Handle)

	rpc.HandleFunc("test", rpc.Test)

	err := http.ListenAndServe(":"+strconv.Itoa(int(common.DEFAULT_RPC_PORT)), nil)
	if err != nil {
		return fmt.Errorf("ListenAndServe error:%s", err)
	}
	return nil
}
