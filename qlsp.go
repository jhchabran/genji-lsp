package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/jhchabran/qlsp/internal/protocol"
	"github.com/sourcegraph/jsonrpc2"
)

type langHandler struct {
	jsonrpc2.Handler
}

func newHandler() jsonrpc2.Handler {
	return jsonrpc2.HandlerWithError((&langHandler{}).handle)
}

func (l *langHandler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
	switch req.Method {
	case "initialize":
		l.log(conn, "initialize")
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}

		var params protocol.InitializeParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		return protocol.InitializeResult{
			Capabilities: protocol.ServerCapabilities{
				HoverProvider: true,
			},
		}, nil
	case "initialized":
		l.log(conn, "initialized")
		return nil, nil
	default:
		l.log(conn, req.Method)
		return nil, nil
	}
}

func (l *langHandler) log(conn *jsonrpc2.Conn, msg string) {
	_ = conn.Notify(context.Background(), "window/logMessage", &protocol.LogMessageParams{
		Type:    3,
		Message: msg,
	})
}

type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}

func main() {
	ctx := context.Background()
	closer := ioutil.NopCloser(strings.NewReader(""))

	f, err := os.OpenFile("/tmp/qlsp.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	mylogger := log.New(f, "", log.LstdFlags)

	<-jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}), newHandler(), jsonrpc2.LogMessages(mylogger)).DisconnectNotify()
	err = closer.Close()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("connection closed")
}
