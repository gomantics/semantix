package chunking

import (
	"encoding/json"
	"flag"
	"os"
	"testing"

	approvals "github.com/approvals/go-approval-tests"
	"github.com/approvals/go-approval-tests/reporters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var acceptChanges bool

func TestMain(m *testing.M) {
	flag.BoolVar(&acceptChanges, "accept-changes", false, "auto-approve snapshot diffs")
	flag.Parse()

	approvals.UseFolder("testdata")
	if acceptChanges {
		approvals.UseFrontLoadedReporter(reporters.NewReporterThatAutomaticallyApproves())
	}

	os.Exit(m.Run())
}

const goSource = `package main

import "fmt"

type Server struct {
	host string
	port int
}

func NewServer(host string, port int) *Server {
	return &Server{host: host, port: port}
}

func (s *Server) Start() error {
	fmt.Printf("listening on %s:%d\n", s.host, s.port)
	return nil
}
`

const jsSource = `function greet(name) {
	return "Hello, " + name;
}

const add = (a, b) => a + b;
`

func TestChunkFile_Go(t *testing.T) {
	t.Parallel()
	chunks, err := ChunkFile("server.go", []byte(goSource))
	require.NoError(t, err)
	assert.NotEmpty(t, chunks, "expected at least one chunk for Go source")

	for _, c := range chunks {
		assert.Equal(t, "go", c.Language)
		assert.NotEmpty(t, c.Content)
		assert.Greater(t, c.EndLine, 0)
	}

	// All chunks should have a valid chunk type (block is the default for Go).
	for _, c := range chunks {
		assert.NotEmpty(t, c.ChunkType, "chunk type should never be empty")
	}
}

func TestChunkFile_JavaScript(t *testing.T) {
	t.Parallel()
	chunks, err := ChunkFile("script.js", []byte(jsSource))
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)

	for _, c := range chunks {
		assert.Equal(t, "javascript", c.Language)
	}
}

func TestChunkFile_UnknownExtension_FallsBackToGeneric(t *testing.T) {
	t.Parallel()
	content := []byte("some plain text content that has no recognized language")

	chunks, err := ChunkFile("data.xyz", []byte(content))
	require.NoError(t, err)
	// Generic chunker should still return something or empty without error.
	_ = chunks
}

func TestChunkFile_EmptyContent(t *testing.T) {
	t.Parallel()
	_, err := ChunkFile("empty.go", []byte(""))
	// Empty content may succeed (returning chunks or none) or return an error.
	// We just assert no panic occurs.
	_ = err
}

func TestClassifyNodeType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		nodeType string
		expected string
	}{
		{"function_declaration", "function"},
		{"function_definition", "function"},
		{"function_item", "function"},
		{"arrow_function", "function"},
		{"func_literal", "function"},
		{"method_declaration", "method"},
		{"method_definition", "method"},
		{"class_declaration", "class"},
		{"class_definition", "class"},
		{"class_specifier", "class"},
		{"type_declaration", "type"},
		{"type_spec", "type"},
		{"struct_type", "type"},
		{"interface_type", "type"},
		{"import_declaration", "import"},
		{"import_statement", "import"},
		{"import_spec_list", "import"},
		{"expression_statement", "block"},
		{"unknown_node", "block"},
		{"", "block"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			t.Parallel()
			got := classifyNodeType(tt.nodeType)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestChunkFile_GoApprovals snapshots the chunk metadata from the canonical Go
// source so regressions in chunking output are caught automatically.
// NOTE: approval tests use shared file-based state and must not run in parallel
// with other approval tests within the same package.
func TestChunkFile_GoApprovals(t *testing.T) {
	chunks, err := ChunkFile("server.go", []byte(goSource))
	require.NoError(t, err)

	type chunkSummary struct {
		FilePath   string `json:"filePath"`
		Language   string `json:"language"`
		ChunkType  string `json:"chunkType"`
		SymbolName string `json:"symbolName"`
		StartLine  int    `json:"startLine"`
		EndLine    int    `json:"endLine"`
	}

	summaries := make([]chunkSummary, len(chunks))
	for i, c := range chunks {
		summaries[i] = chunkSummary{
			FilePath:   c.FilePath,
			Language:   c.Language,
			ChunkType:  c.ChunkType,
			SymbolName: c.SymbolName,
			StartLine:  c.StartLine,
			EndLine:    c.EndLine,
		}
	}

	b, err := json.MarshalIndent(summaries, "", "  ")
	require.NoError(t, err)

	approvals.VerifyString(t, string(b))
}
