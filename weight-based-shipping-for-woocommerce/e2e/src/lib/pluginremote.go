package lib

import (
	"e2e/src/lib/shell"
	"fmt"
	"path"
	"strings"
)

type PluginRemote struct {
	shell *shell.Shell
	meta  *PluginMeta
}

func NewPluginRemote(shell *shell.Shell, meta *PluginMeta) *PluginRemote {
	return &PluginRemote{shell, meta}
}

func (p *PluginRemote) ReplaceLicense(license *string) {

	licfile := path.Join(p.meta.Dir, "license.key")

	if license == nil {
		p.shell.FS.Remove(licfile)
		return
	}

	p.shell.FS.Write(licfile, strings.NewReader(*license))
}

func (p *PluginRemote) ReplaceVersion(version string) {
	expr := fmt.Sprintf(`s/Version: .*/Version: %s/`, version)
	p.shell.Cmd(nil, "sed", "-i", expr, p.meta.PluginFile).Must()
}
