package qlsp

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/jhchabran/qlsp/internal/protocol"
	"github.com/sourcegraph/jsonrpc2"
)

type serverHandler struct {
	jsonrpc2.Handler
	server WIPServer
}

func newServerHandler(server WIPServer) jsonrpc2.Handler {
	return jsonrpc2.HandlerWithError((&serverHandler{server: server}).handle)
}

func (sh *serverHandler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
	LogClient(withConn(ctx, conn), req.Method)
	switch req.Method {
	case "initialize":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}

		var params protocol.ParamInitialize
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		return sh.server.Initialize(withConn(ctx, conn), &params)
	case "initialized":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}

		var params protocol.InitializedParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		return nil, sh.server.Initialized(withConn(ctx, conn), &params)
	case "textDocument/hover":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}

		var params protocol.HoverParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		return sh.server.Hover(withConn(ctx, conn), &params)
	case "textDocument/didOpen":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}

		var params protocol.DidOpenTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		return nil, sh.server.DidOpen(withConn(ctx, conn), &params)
	case "textDocument/didChange":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}

		var params protocol.DidChangeTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		return nil, sh.server.DidChange(withConn(ctx, conn), &params)

	default:
		LogClient(withConn(ctx, conn), req.Method)
		return nil, nil
	}
}

type connKey struct{}

func withConn(ctx context.Context, conn *jsonrpc2.Conn) context.Context {
	return context.WithValue(ctx, connKey{}, conn)
}

// LogClient sends a log message to the client.
func LogClient(ctx context.Context, message string) error {
	if conn, ok := ctx.Value(connKey{}).(*jsonrpc2.Conn); ok {
		err := conn.Notify(ctx, "window/logMessage", &protocol.LogMessageParams{
			Type:    protocol.Info,
			Message: message,
		})

		return err
	}

	return nil
}

type WIPServer interface {
	Initialize(context.Context, *protocol.ParamInitialize) (*protocol.InitializeResult, error)
	Initialized(context.Context, *protocol.InitializedParams) error
	Hover(context.Context, *protocol.HoverParams) (*protocol.Hover /*Hover | null*/, error)
	DidOpen(context.Context, *protocol.DidOpenTextDocumentParams) error
	DidChange(context.Context, *protocol.DidChangeTextDocumentParams) error
}

// BaseServer implements the protocol.Server interface but performs no operations that writing back to
// the client the name of the called method as a log message.
type BaseServer struct{}

func (s *BaseServer) Initialize(ctx context.Context, params *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	err := LogClient(ctx, "Initialize NOOP")
	return nil, err
}

func (s *BaseServer) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	return LogClient(ctx, "Initialized NOOP")
}

func (s *BaseServer) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover /*Hover | null*/, error) {
	err := LogClient(ctx, "Hover: NOOP")
	return nil, err
}

func (s *BaseServer) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	err := LogClient(ctx, "DidOpen NOOP")
	return err
}

func (s *BaseServer) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	err := LogClient(ctx, "DidChange NOOP")
	return err
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

func Serve(s WIPServer) error {
	ctx := context.Background()
	closer := ioutil.NopCloser(strings.NewReader(""))

	f, err := os.OpenFile("/tmp/qlsp.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	mylogger := log.New(f, "", log.LstdFlags)

	stream := jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{})

	<-jsonrpc2.NewConn(ctx, stream, newServerHandler(s), jsonrpc2.LogMessages(mylogger)).DisconnectNotify()

	err = closer.Close()
	return err
}
