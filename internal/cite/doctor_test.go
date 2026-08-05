package cite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/toba/jig-go/v4/internal/config"
	"github.com/toba/jig-go/v4/internal/github"
)

// mockClient implements github.Client for testing.
type mockClient struct {
	license    *github.LicenseInfo
	licenseErr error
}

func (m *mockClient) GetLicense(repo string) (*github.LicenseInfo, error) {
	if m.licenseErr != nil {
		return nil, m.licenseErr
	}
	return m.license, nil
}

func (m *mockClient) GetCommits(repo, branch string, perPage int) ([]github.Commit, error) {
	return nil, nil
}
func (m *mockClient) GetCommitsSince(repo, branch, since string, perPage int) ([]github.Commit, error) {
	return nil, nil
}
func (m *mockClient) Compare(repo, base, head string) (*github.CompareResponse, error) {
	return nil, nil
}
func (m *mockClient) GetCommitDetail(repo, sha string) (*github.Commit, error) { return nil, nil }
func (m *mockClient) GetHeadSHA(repo, branch string) (string, error)           { return "", nil }
func (m *mockClient) GetRepo(repo string) (*github.RepoInfo, error)            { return nil, nil }
func (m *mockClient) GetTree(repo, branch string) (*github.TreeResponse, error) {
	return nil, nil
}
func (m *mockClient) GetLatestRelease(repo string) (*github.Release, error) { return nil, nil }
func (m *mockClient) ListReleases(repo string, perPage int) ([]github.Release, error) {
	return nil, nil
}

func TestSplitRepo(t *testing.T) {
	tests := []struct {
		slug      string
		wantOwner string
		wantRepo  string
	}{
		{"toba/jig", "toba", "jig"},
		{"foo/bar-baz", "foo", "bar-baz"},
		{"single", "", "single"},
	}
	for _, tt := range tests {
		owner, repo := splitRepo(tt.slug)
		if owner != tt.wantOwner || repo != tt.wantRepo {
			t.Errorf("splitRepo(%q) = (%q, %q), want (%q, %q)", tt.slug, owner, repo, tt.wantOwner, tt.wantRepo)
		}
	}
}

func TestIsGitHubRepo(t *testing.T) {
	tests := []struct {
		repo string
		want bool
	}{
		{"toba/jig", true},
		{"foo/bar", true},
		{"https://gitlab.com/foo/bar", false},
		{"single", false},
	}
	for _, tt := range tests {
		if got := isGitHubRepo(tt.repo); got != tt.want {
			t.Errorf("isGitHubRepo(%q) = %v, want %v", tt.repo, got, tt.want)
		}
	}
}

func TestDiscoverLicenseFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.WriteFile("LICENSE", []byte("MIT License"), 0o644)
	os.WriteFile("NOTICE", []byte("Third party notices"), 0o644)

	found := discoverLicenseFiles()
	if len(found) != 2 {
		t.Fatalf("expected 2 license files, got %d: %v", len(found), found)
	}
}

func TestRunDoctor_NoSources(t *testing.T) {
	code := RunDoctor(DoctorOpts{Sources: nil})
	if code != 0 {
		t.Errorf("expected exit 0 for no sources, got %d", code)
	}
}

func TestRunDoctor_NoLicenseFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	code := RunDoctor(DoctorOpts{
		Sources: []config.Source{{Repo: "toba/jig"}},
	})
	// No license files → warn but exit 0.
	if code != 0 {
		t.Errorf("expected exit 0 when no license files, got %d", code)
	}
}

func TestRunDoctor_AttributionFound(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.WriteFile("LICENSE", []byte("MIT License\n\nPortions from toba/jig are used under MIT."), 0o644)

	code := RunDoctor(DoctorOpts{
		Sources: []config.Source{{Repo: "toba/jig"}},
	})
	if code != 0 {
		t.Errorf("expected exit 0 for found attribution, got %d", code)
	}
}

func TestRunDoctor_AttributionMissing(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.WriteFile("LICENSE", []byte("MIT License\n\nCopyright 2024 My Project"), 0o644)

	code := RunDoctor(DoctorOpts{
		Sources: []config.Source{{Repo: "toba/jig"}},
	})
	if code != 1 {
		t.Errorf("expected exit 1 for missing attribution, got %d", code)
	}
}

func TestRunDoctor_PartialMatch(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Mention only the repo name, not the full slug.
	os.WriteFile("NOTICE", []byte("This project uses jig for issue tracking."), 0o644)

	code := RunDoctor(DoctorOpts{
		Sources: []config.Source{{Repo: "toba/jig"}},
	})
	if code != 0 {
		t.Errorf("expected exit 0 for partial repo name match, got %d", code)
	}
}

func TestRunDoctor_MultipleSourcesMixed(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.WriteFile("LICENSE", []byte("Uses toba/jig under MIT."), 0o644)

	code := RunDoctor(DoctorOpts{
		Sources: []config.Source{
			{Repo: "toba/jig"},
			{Repo: "other/missing"},
		},
	})
	if code != 1 {
		t.Errorf("expected exit 1 when one source missing attribution, got %d", code)
	}
}

func TestRunDoctor_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.WriteFile("LICENSE", []byte("Includes code from TOBA/JIG"), 0o644)

	code := RunDoctor(DoctorOpts{
		Sources: []config.Source{{Repo: "toba/jig"}},
	})
	if code != 0 {
		t.Errorf("expected exit 0 for case-insensitive match, got %d", code)
	}
}

func TestRunDoctor_NoticeFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Not in LICENSE but in NOTICE.
	os.WriteFile("LICENSE", []byte("MIT License"), 0o644)
	os.WriteFile("NOTICE", []byte("toba/jig - MIT License"), 0o644)

	code := RunDoctor(DoctorOpts{
		Sources: []config.Source{{Repo: "toba/jig"}},
	})
	if code != 0 {
		t.Errorf("expected exit 0 when attribution in NOTICE, got %d", code)
	}
}

func TestRunDoctor_WithGitHubLicense(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	os.WriteFile("LICENSE", []byte("Uses toba/jig"), 0o644)

	info := &github.LicenseInfo{}
	info.License.Key = "mit"
	info.License.Name = "MIT License"
	info.License.SPDXID = "MIT"

	code := RunDoctor(DoctorOpts{
		Sources: []config.Source{{Repo: "toba/jig"}},
		Client:  &mockClient{license: info},
	})
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestFileNames(t *testing.T) {
	paths := []string{"/foo/bar/LICENSE", "/baz/NOTICE.md"}
	got := fileNames(paths)
	if got[0] != "LICENSE" || got[1] != "NOTICE.md" {
		t.Errorf("fileNames = %v, want [LICENSE NOTICE.md]", got)
	}
	_ = filepath.Base // suppress unused import if needed
}
