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

// buildDest calls buildHardlinkDestDir with whichever signature this qui
// version has, so one test file works for releases and the dev branch.
// Taking the method value compiles for any signature; the type switch then
// picks the matching one. A signature this file does not know fails the test
// with the new signature in the message.
func buildDest(
	t *testing.T,
	s *Service,
	instance *models.Instance,
	baseDir, torrentHash, torrentName string,
	candidate CrossSeedCandidate,
	trackerDomain string,
	files []hardlinktree.TorrentFile,
) string {
	t.Helper()
	ctx := context.Background()
	req := &CrossSeedRequest{}

	switch f := any(s.buildHardlinkDestDir).(type) {
	// v1.31.1 and earlier
	case func(context.Context, *models.Instance, string, string, string, CrossSeedCandidate, string, *CrossSeedRequest, []hardlinktree.TorrentFile) string:
		return f(ctx, instance, baseDir, torrentHash, torrentName, candidate, trackerDomain, req, files)
	// Path dialect added after v1.31.1 (upstream PR #3140)
	case func(context.Context, fsops.PathDialect, *models.Instance, string, string, string, CrossSeedCandidate, string, *CrossSeedRequest, []hardlinktree.TorrentFile) string:
		return f(ctx, fsops.HostPaths, instance, baseDir, torrentHash, torrentName, candidate, trackerDomain, req, files)
	default:
		t.Fatalf("buildHardlinkDestDir has a new signature, update tests/local_patch_test.go: %T", f)
		return ""
	}
}

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
		return buildDest(t, s, instance, baseDir, "abcdef1234567890",
			"Some.Movie.2024.1080p.WEB-DL-GRP", candidate, "tracker.example.com", files)
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
