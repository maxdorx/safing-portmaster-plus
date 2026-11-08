package updates

import "testing"

func TestValidateArtifactFilename(t *testing.T) {
	t.Parallel()

	valid := []string{
		"filterlists.zip",
		"geoipv4.mmdb",
		"portmaster-core",
		"base.dsdl",
		"index.dsd",
		"file.with.dots.tar.gz",
		"file_with_underscores",
		"file-with-dashes",
	}

	invalid := []string{
		"",            // empty
		".",           // current dir (fs.ValidPath special-cases it as valid)
		"..",          // parent dir traversal
		"../evil",     // traversal with component
		"./evil",      // non-canonical: path.Clean changes it
		"a/../evil",   // non-canonical: path.Clean changes it
		"dir/file",    // path with component
		"dir\\file",   // backslash path (Windows-style)
		"//evil",      // non-canonical double slash
		"evil/",       // trailing slash
		"C:file",      // Windows drive-relative
		"file:stream", // NTFS alternate data stream
	}

	for _, name := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			t.Parallel()
			if err := validateArtifactFilename(name); err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
		})
	}

	for _, name := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			t.Parallel()
			if err := validateArtifactFilename(name); err == nil {
				t.Errorf("expected error for %q, got nil", name)
			}
		})
	}
}
