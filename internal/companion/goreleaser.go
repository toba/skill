package companion

import (
	"os"

	"github.com/toba/jig-go/v4/internal/constants"
)

// CheckGoreleaserExists looks for .goreleaser.yaml or .goreleaser.yml in the
// current directory. It returns the file content, the filename found, and
// whether either file exists.
func CheckGoreleaserExists() ([]byte, string, bool) {
	if data, err := os.ReadFile(constants.GoreleaserYAML); err == nil {
		return data, constants.GoreleaserYAML, true
	}
	if data, err := os.ReadFile(constants.GoreleaserYML); err == nil {
		return data, constants.GoreleaserYML, true
	}
	return nil, "", false
}
