package updates

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/safing/portmaster/base/utils"
)

const (
	localTestIndexFile   = "index.json"
	localTestFilterLists = "filterlists.zip"
)

func TestManualOnlyRejectsAllUpdatePaths(t *testing.T) {
	t.Parallel()

	u, err := New(&testInstance{}, "local", Config{
		Name: "local", Directory: t.TempDir(), DownloadDirectory: t.TempDir(),
		PurgeDirectory: t.TempDir(), IndexFile: localTestIndexFile, ManualOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	u.Configure(true, []string{"https://example.invalid/index.json"})
	if u.cfg.AutoCheck {
		t.Fatal("persisted config re-enabled app updates")
	}
	err = u.updateAndUpgrade(nil, nil, true, true)
	if !errors.Is(err, ErrManualUpdatesOnly) {
		t.Fatalf("forced update: %v", err)
	}
	err = u.upgrade(nil, true)
	if !errors.Is(err, ErrManualUpdatesOnly) {
		t.Fatalf("staged update: %v", err)
	}
}

func TestManualOnlyServesLocallyBuiltUI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// New applies production ACLs. Restore write access before testing.TempDir cleanup.
	t.Cleanup(func() { _ = utils.SetFilePermission(dir, utils.PublicWriteExecPermission) })
	err := os.WriteFile(filepath.Join(dir, "portmaster.zip"), []byte("local UI"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, localTestIndexFile), []byte("upstream index does not describe these local artifacts"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	u, err := New(&testInstance{}, "local", Config{
		Name: "local", Directory: dir,
		DownloadDirectory: t.TempDir(), PurgeDirectory: t.TempDir(), IndexFile: localTestIndexFile, ManualOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if u.corruptedInstallation != nil {
		t.Fatal(u.corruptedInstallation)
	}
	_, err = u.GetFile("portmaster.zip")
	if err != nil {
		t.Fatal(err)
	}
}

func TestExcludedResourcesKeepProtectionArtifacts(t *testing.T) {
	t.Parallel()

	cfg := Config{ExcludeArtifacts: []string{"news.yaml", "notifications.yaml"}}
	idx := &Index{Artifacts: []*Artifact{{Filename: "news.yaml"}, {Filename: "notifications.yaml"}, {Filename: "geoip.mmdb"}, {Filename: localTestFilterLists}}}
	cfg.filterIndex(idx)
	if len(idx.Artifacts) != 2 || idx.Artifacts[0].Filename != "geoip.mmdb" || idx.Artifacts[1].Filename != localTestFilterLists {
		t.Fatalf("unexpected artifacts: %#v", idx.Artifacts)
	}
}
