package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
	"github.com/jhchabran/qlsp"
	"github.com/jhchabran/qlsp/internal/protocol"
)

type genjiServer struct {
	qlsp.BaseServer
	files map[string]string
}

func (s *genjiServer) withDB(f func(db *genji.DB) error) error {
	db, err := genji.Open(":memory:")
	if err != nil {
		return err
	}
	defer db.Close()
	return f(db)
}

func (s *genjiServer) Initialize(ctx context.Context, params *protocol.ParamInitialize) (*protocol.InitializeResult, error) {
	qlsp.LogClient(ctx, "Initializing")
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			HoverProvider:        true,
			TextDocumentSync:     protocol.Full,
			DocumentLinkProvider: protocol.DocumentLinkOptions{ResolveProvider: false},
		},
	}, nil
}

func (s *genjiServer) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	qlsp.LogClient(ctx, "Initialized")
	return nil
}

func (s *genjiServer) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover /*Hover | null*/, error) {
	qlsp.LogClient(ctx, "Hover")

	f := s.files[string(params.TextDocument.URI)]
	line := int(params.Position.Line)
	chunks := (strings.Split(f, "\n"))[0 : line+1]
	w := bytes.Buffer{}

	err := s.withDB(func(db *genji.DB) error {
		res, err := db.Query(strings.Join(chunks, "\n"))
		if err != nil {
			return err
		}

		err = document.IteratorToJSONArray(&w, res)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	md := fmt.Sprintf("```json\n%s\n```", w.String())
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: md,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: params.Position.Line, Character: 0},
			End:   protocol.Position{Line: params.Position.Line, Character: params.Position.Character},
		},
	}, nil
}

func (s *genjiServer) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	qlsp.LogClient(ctx, "DidOpen")
	qlsp.LogClient(ctx, string(params.TextDocument.URI))
	s.files[string(params.TextDocument.URI)] = params.TextDocument.Text
	return nil
}

func (s *genjiServer) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	qlsp.LogClient(ctx, "DidChange")
	// https://microsoft.github.io/language-server-protocol/specifications/specification-current/#textDocument_synchronization
	// We use TextDocumentSyncKind.Full, no need to apply the changes incrementally.
	s.files[string(params.TextDocument.URI)] = params.ContentChanges[0].Text
	return nil
}

func main() {
	gs := genjiServer{files: map[protocol.URI]string{}}
	err := qlsp.Serve(&gs)
	if err != nil {
		panic(err)
	}
}
