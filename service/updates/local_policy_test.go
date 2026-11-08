package updates

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/safing/portmaster/base/utils"
)

func TestManualOnlyRejectsAllUpdatePaths(t *testing.T) {
	u, err := New(&testInstance{}, "local", Config{
		Name: "local", Directory: t.TempDir(), DownloadDirectory: t.TempDir(),
		PurgeDirectory: t.TempDir(), IndexFile: "index.json", ManualOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	u.Configure(true, []string{"https://example.invalid/index.json"})
	if u.cfg.AutoCheck {
		t.Fatal("persisted config re-enabled app updates")
	}
	if err := u.updateAndUpgrade(nil, nil, true, true); !errors.Is(err, ErrManualUpdatesOnly) {
		t.Fatalf("forced update: %v", err)
	}
	if err := u.upgrade(nil, true); !errors.Is(err, ErrManualUpdatesOnly) {
		t.Fatalf("staged update: %v", err)
	}
}

func TestManualOnlyServesLocallyBuiltUI(t *testing.T) {
	dir := t.TempDir()
	// New applies production ACLs. Restore write access before testing.TempDir cleanup.
	t.Cleanup(func() { _ = utils.SetFilePermission(dir, utils.PublicWriteExecPermission) })
	if err := os.WriteFile(filepath.Join(dir, "portmaster.zip"), []byte("local UI"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte("upstream index does not describe these local artifacts"), 0600); err != nil {
		t.Fatal(err)
	}
	u, err := New(&testInstance{}, "local", Config{Name: "local", Directory: dir,
		DownloadDirectory: t.TempDir(), PurgeDirectory: t.TempDir(), IndexFile: "index.json", ManualOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if u.corruptedInstallation != nil {
		t.Fatal(u.corruptedInstallation)
	}
	if _, err := u.GetFile("portmaster.zip"); err != nil {
		t.Fatal(err)
	}
}

func TestExcludedResourcesKeepProtectionArtifacts(t *testing.T) {
	cfg := Config{ExcludeArtifacts: []string{"news.yaml", "notifications.yaml"}}
	idx := &Index{Artifacts: []*Artifact{{Filename: "news.yaml"}, {Filename: "notifications.yaml"}, {Filename: "geoip.mmdb"}, {Filename: "filterlists.zip"}}}
	cfg.filterIndex(idx)
	if len(idx.Artifacts) != 2 || idx.Artifacts[0].Filename != "geoip.mmdb" || idx.Artifacts[1].Filename != "filterlists.zip" {
		t.Fatalf("unexpected artifacts: %#v", idx.Artifacts)
	}
}
