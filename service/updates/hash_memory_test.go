package updates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalArtifactHashingUsesBoundedMemory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "core.bin")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Truncate(8 << 20)
	if err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	result := testing.Benchmark(func(b *testing.B) {
		b.Helper()
		for b.Loop() {
			_, err := GetSHA256(path, "")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	t.Logf("plain artifact hashing: %d bytes allocated per 8 MiB file", result.AllocedBytesPerOp())
	if result.AllocedBytesPerOp() > 256<<10 {
		t.Fatalf("hashing allocates %d bytes per file; local core indexing must stream large artifacts", result.AllocedBytesPerOp())
	}
}
