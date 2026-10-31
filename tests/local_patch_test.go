package crossseed

// Copied into internal/services/crossseed by the build workflow.
// Fails the build if the single-file isolation patch stops working.
// Mirrors upstream's TestLinkIntoNewFolder (hardlink_mode_test.go).

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/qui/internal/fsops"
	"github.com/autobrr/qui/internal/fsops/local"
	"github.com/autobrr/qui/pkg/hardlinktree"
)

func TestLocalPatch_LinkDestDir_SingleFile(t *testing.T) {
	t.Parallel()

	topFolder, destDir := linkDestDir(fsops.SlashPaths, "/downloads/AvistaZ", "Movie.2024",
		[]hardlinktree.TorrentFile{{Path: "Movie.2024.mkv"}})
	assert.Empty(t, topFolder, "a single bare file reserves no folder")
	assert.Equal(t, "/downloads/AvistaZ", destDir)
}

func TestLocalPatch_LinkIntoNewFolder(t *testing.T) {
	t.Parallel()

	single := []hardlinktree.TorrentFile{{Path: "Movie.2024.mkv"}}
	withRoot := []hardlinktree.TorrentFile{{Path: "Movie/movie.mkv"}, {Path: "Movie/subs.srt"}}
	rootless := []hardlinktree.TorrentFile{{Path: "movie.mkv"}, {Path: "subs.srt"}}

	tests := []struct {
		name         string
		files        []hardlinktree.TorrentFile
		existingFile string // file already in the preset dir
		existingDir  string // folder already in the preset dir
		wantDest     string // relative to the preset dir
		wantReserved string // folder the call reserves, "" for none
	}{
		// The patch: a single file goes straight into the tracker folder.
		{name: "single file, name free", files: single, wantDest: "."},
		// Name taken: isolated like upstream, never overwritten.
		{name: "single file, name taken", files: single, existingFile: "Movie.2024.mkv", wantDest: "Movie.2024", wantReserved: "Movie.2024"},
		{name: "single file, name and folder taken", files: single, existingFile: "Movie.2024.mkv", existingDir: "Movie.2024", wantDest: "Movie.2024--abcdef12"},
		// Unchanged upstream behavior.
		{name: "root folder is free", files: withRoot, wantDest: ".", wantReserved: "Movie"},
		{name: "rootless multi-file is isolated", files: rootless, wantDest: "Movie.2024", wantReserved: "Movie.2024"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			presetDir := filepath.Join(t.TempDir(), "AvistaZ")
			if tt.existingFile != "" {
				require.NoError(t, os.MkdirAll(presetDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(presetDir, tt.existingFile), []byte("x"), 0o600))
			}
			if tt.existingDir != "" {
				require.NoError(t, os.MkdirAll(filepath.Join(presetDir, tt.existingDir), 0o755))
			}

			var gotDest string
			created, err := LinkIntoNewFolder(t.Context(), local.NewBackend(), presetDir, "ABCDEF1234567890", "Movie.2024", tt.files,
				func(destDir string) (*fsops.TreeCreateResult, error) {
					gotDest = destDir
					return &fsops.TreeCreateResult{}, nil
				})
			require.NoError(t, err)
			require.Equal(t, filepath.Clean(filepath.Join(presetDir, tt.wantDest)), filepath.Clean(gotDest))

			if tt.wantReserved == "" {
				assert.Empty(t, created.Dirs)
				return
			}
			assert.Equal(t, []string{filepath.Join(presetDir, tt.wantReserved)}, created.Dirs)
			assert.DirExists(t, filepath.Join(presetDir, tt.wantReserved))
		})
	}
}
