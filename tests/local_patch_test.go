package crossseed

// Copied into internal/services/crossseed by the build workflow.
// Fails the build if the single-file isolation patch stops working.

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/qui/internal/fsops"
	"github.com/autobrr/qui/internal/models"
	"github.com/autobrr/qui/pkg/hardlinktree"
)

func TestLocalPatch_SingleFileTorrentHasNoIsolationFolder(t *testing.T) {
	s := &Service{
		trackerCustomizationStore: &mockTrackerCustomizationStore{
			customizations: []*models.TrackerCustomization{
				{DisplayName: "AvistaZ", Domains: []string{"tracker.example.com"}},
			},
		},
	}
	baseDir := filepath.FromSlash("/downloads")
	instance := &models.Instance{
		ID:                1,
		Name:              "qbt",
		HardlinkBaseDir:   baseDir,
		HardlinkDirPreset: "by-tracker",
	}
	candidate := CrossSeedCandidate{InstanceID: 1, InstanceName: "qbt"}
	trackerDir := filepath.Join(baseDir, "AvistaZ")

	dest := func(files []hardlinktree.TorrentFile) string {
		return s.buildHardlinkDestDir(context.Background(), fsops.HostPaths, instance, baseDir,
			"abcdef1234567890", "Some.Movie.2024.1080p.WEB-DL-GRP", candidate,
			"tracker.example.com", &CrossSeedRequest{}, files)
	}

	// The patched case: one bare file goes straight into the tracker folder.
	assert.Equal(t, trackerDir, dest([]hardlinktree.TorrentFile{
		{Path: "Some.Movie.2024.1080p.WEB-DL-GRP.mkv", Size: 1000},
	}))

	// Unchanged: a torrent with a root folder lands in the tracker folder.
	assert.Equal(t, trackerDir, dest([]hardlinktree.TorrentFile{
		{Path: "Show.S01/e01.mkv", Size: 1000},
		{Path: "Show.S01/e02.mkv", Size: 1000},
	}))

	// Unchanged: several rootless files still get their own folder.
	assert.Equal(t, filepath.Join(trackerDir, "Some.Movie.2024.1080p.WEB-DL-GRP--abcdef12"), dest([]hardlinktree.TorrentFile{
		{Path: "video.mkv", Size: 1000},
		{Path: "subs.srt", Size: 100},
	}))
}
