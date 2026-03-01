package chunking

import (
	"fmt"

	"github.com/gomantics/chunkx"
	"github.com/gomantics/chunkx/languages"
)

const defaultMaxSize = 500

// Chunk represents a code chunk with its metadata.
type Chunk struct {
	Content    string
	FilePath   string
	StartLine  int
	EndLine    int
	Language   string
	ChunkType  string
	SymbolName string
}

// ChunkFile parses source code and returns semantically meaningful chunks.
func ChunkFile(filePath string, content []byte) ([]Chunk, error) {
	lang, detected := languages.DetectLanguage(filePath)
	if !detected {
		lang, _ = languages.GetLanguageConfig(languages.Generic)
	}

	chunker := chunkx.NewChunker()
	raw, err := chunker.Chunk(string(content),
		chunkx.WithLanguage(lang.Name),
		chunkx.WithMaxSize(defaultMaxSize),
	)
	if err != nil {
		return nil, fmt.Errorf("chunk %s: %w", filePath, err)
	}

	chunks := make([]Chunk, 0, len(raw))
	for _, r := range raw {
		chunkType := "block"
		symbolName := ""
		if len(r.NodeTypes) > 0 {
			chunkType = classifyNodeType(r.NodeTypes[0])
			symbolName = extractSymbolName(r.NodeTypes)
		}

		chunks = append(chunks, Chunk{
			Content:    r.Content,
			FilePath:   filePath,
			StartLine:  r.StartLine,
			EndLine:    r.EndLine,
			Language:   string(lang.Name),
			ChunkType:  chunkType,
			SymbolName: symbolName,
		})
	}

	return chunks, nil
}

func classifyNodeType(nodeType string) string {
	switch nodeType {
	case "function_declaration", "function_definition", "function_item",
		"arrow_function", "func_literal":
		return "function"
	case "method_declaration", "method_definition":
		return "method"
	case "class_declaration", "class_definition", "class_specifier":
		return "class"
	case "type_declaration", "type_spec", "struct_type", "interface_type":
		return "type"
	case "import_declaration", "import_statement", "import_spec_list":
		return "import"
	default:
		return "block"
	}
}

func extractSymbolName(nodeTypes []string) string {
	if len(nodeTypes) > 1 {
		return nodeTypes[1]
	}
	return ""
}
