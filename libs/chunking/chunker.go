package chunking

import (
	"github.com/gomantics/chunkx"
)

// Chunk represents a code chunk with metadata
type Chunk struct {
	Content   string
	StartLine int
	EndLine   int
	Index     int
	Language  string
}

// DefaultMaxSize is the default maximum chunk size in tokens
const DefaultMaxSize = 500

// Chunker wraps the chunkx chunker
type Chunker struct {
	chunker chunkx.Chunker
}

// NewChunker creates a new Chunker instance
func NewChunker() *Chunker {
	return &Chunker{
		chunker: chunkx.NewChunker(),
	}
}

// ChunkFile chunks a file by path, automatically detecting the language
func (c *Chunker) ChunkFile(filePath string) ([]Chunk, error) {
	chunks, err := c.chunker.ChunkFile(filePath, chunkx.WithMaxSize(DefaultMaxSize))
	if err != nil {
		return nil, err
	}

	return convertChunks(chunks), nil
}

// ChunkContent chunks content with an explicit language
func (c *Chunker) ChunkContent(content string, language string) ([]Chunk, error) {
	var opts []chunkx.Option
	opts = append(opts, chunkx.WithMaxSize(DefaultMaxSize))

	chunks, err := c.chunker.Chunk(content, opts...)
	if err != nil {
		return nil, err
	}

	return convertChunks(chunks), nil
}

// convertChunks converts chunkx chunks to our Chunk type
func convertChunks(chunks []chunkx.Chunk) []Chunk {
	result := make([]Chunk, len(chunks))
	for i, chunk := range chunks {
		result[i] = Chunk{
			Content:   chunk.Content,
			StartLine: chunk.StartLine,
			EndLine:   chunk.EndLine,
			Index:     i,
			Language:  string(chunk.Language),
		}
	}
	return result
}

