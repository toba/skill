package brew

import (
	"cmp"
	"fmt"

	"github.com/toba/jig-go/v4/internal/companion"
)

// WorkflowParams holds the inputs needed to generate the update-homebrew CI job.
type WorkflowParams struct {
	Tool    string // binary name, e.g. "todo"
	Org     string // GitHub org, e.g. "toba"
	Tap     string // tap repo, e.g. "toba/homebrew-tap" (empty = Org/homebrew-Tool)
	Desc    string // one-line description
	License string // e.g. "Apache-2.0"
	Asset   string // e.g. "todo_darwin_arm64.tar.gz"
	Needs   string // job this depends on, e.g. "release"
}

// GenerateWorkflowJob produces the YAML block for an update-homebrew job.
func GenerateWorkflowJob(p WorkflowParams) string {
	className := formulaClassName(p.Tool)
	needs := cmp.Or(p.Needs, "release")
	tapRepo := p.Tap
	return fmt.Sprintf(`
  update-homebrew:
    needs: %s
    runs-on: ubuntu-latest
    steps:
      - name: Update Homebrew formula
        env:
          GH_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
        run: |
          TAG="${GITHUB_REF#refs/tags/}"
          VERSION="${TAG#v}"

          ASSET="%s"
          SHA=$(gh release download "$TAG" --repo "$GITHUB_REPOSITORY" --pattern checksums.txt -O - \
            | grep "$ASSET" | awk '{print $1}')

          if [ -z "$SHA" ]; then
            echo "ERROR: Could not extract SHA256 for $ASSET"
            exit 1
          fi

          git clone "https://x-access-token:${GH_TOKEN}@github.com/%s.git" tap
          cd tap

          cat > Formula/%s.rb << FORMULA
          class %s < Formula
            desc "%s"
            homepage "https://github.com/%s/%s"
            url "https://github.com/%s/%s/releases/download/${TAG}/%s"
            version "${VERSION}"
            sha256 "${SHA}"
            license "%s"

            depends_on :macos
            depends_on arch: :arm64

            def install
              bin.install "%s"
            end

            test do
              assert_match "%s", shell_output("#{bin}/%s version")
            end
          end
          FORMULA

          sed -i 's/^          //' Formula/%s.rb

          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add Formula/%s.rb
          git commit -m "bump %s to ${VERSION}"
          git push
`, needs,
		p.Asset,
		tapRepo,
		p.Tool,
		className, p.Desc, p.Org, p.Tool,
		p.Org, p.Tool, p.Asset,
		p.License,
		p.Tool,
		p.Tool, p.Tool,
		p.Tool,
		p.Tool,
		p.Tool)
}

// InjectWorkflowJob appends the update-homebrew job to an existing workflow file.
// It returns the modified content or an error if the job already exists.
func InjectWorkflowJob(content string, p WorkflowParams) (string, error) {
	return companion.InjectJob(content, "update-homebrew:", &p.Needs, func() string {
		return GenerateWorkflowJob(p)
	})
}
