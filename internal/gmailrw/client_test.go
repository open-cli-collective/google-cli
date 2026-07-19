package gmailrw

import "testing"

func TestChunkIDs(t *testing.T) {
	ids := make([]string, 2500)
	for i := range ids {
		ids[i] = "id"
	}

	chunks := chunkIDs(ids, 1000)
	if len(chunks) != 3 {
		t.Fatalf("2500 ids @1000 => %d chunks, want 3", len(chunks))
	}
	if len(chunks[0]) != 1000 || len(chunks[1]) != 1000 || len(chunks[2]) != 500 {
		t.Fatalf("chunk sizes = %d/%d/%d, want 1000/1000/500", len(chunks[0]), len(chunks[1]), len(chunks[2]))
	}

	// Total is preserved.
	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	if total != len(ids) {
		t.Fatalf("chunked total = %d, want %d", total, len(ids))
	}

	// A non-positive size is a single passthrough chunk (never a panic).
	if got := chunkIDs(ids, 0); len(got) != 1 {
		t.Fatalf("size 0 => %d chunks, want 1 passthrough", len(got))
	}

	// Empty input yields no chunks.
	if got := chunkIDs(nil, 1000); len(got) != 0 {
		t.Fatalf("nil ids => %d chunks, want 0", len(got))
	}
}
