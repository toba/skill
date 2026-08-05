package zed

import (
	"cmp"
	"fmt"

	"github.com/toba/jig-go/v4/internal/companion"
)

// WorkflowParams holds the inputs needed to generate the sync-extension CI job.
type WorkflowParams struct {
	Org   string // GitHub org, e.g. "toba"
	Ext   string // extension repo name, e.g. "gozer"
	Needs string // job this depends on, e.g. "release"
}

// GenerateSyncExtensionJob produces the YAML block for a sync-extension job.
func GenerateSyncExtensionJob(p WorkflowParams) string {
	needs := cmp.Or(p.Needs, "release")
	return fmt.Sprintf(`
  sync-extension:
    runs-on: ubuntu-latest
    needs: %s
    steps:
      - name: Dispatch version bump to %s/%s
        run: |
          gh api repos/%s/%s/dispatches \
            -f event_type=bump-version \
            -f 'client_payload[version]=${{ github.ref_name }}'
        env:
          GH_TOKEN: ${{ secrets.EXTENSION_PAT }}
`, needs, p.Org, p.Ext, p.Org, p.Ext)
}

// InjectSyncExtensionJob appends the sync-extension job to an existing workflow file.
// It returns the modified content or an error if the job already exists.
func InjectSyncExtensionJob(content string, p WorkflowParams) (string, error) {
	return companion.InjectJob(content, "sync-extension:", &p.Needs, func() string {
		return GenerateSyncExtensionJob(p)
	})
}
