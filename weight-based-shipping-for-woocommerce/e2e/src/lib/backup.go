package lib

import (
	"e2e/src/lib/shell"
	"fmt"
	"strings"
)

func Backup(shell *shell.Shell, pluginDir string) func() {

	const dbBkpFile = "/tmp/dbbkp.sql"
	if !shell.FS.Exists(dbBkpFile) {
		shell.Cmd(nil, "wp", "db", "export", dbBkpFile).Must()
	}

	pluginBkpDir := fmt.Sprintf("%s-bkp", strings.TrimRight(pluginDir, "/"))
	if !shell.FS.Exists(pluginBkpDir) {
		shell.FS.Copy(pluginDir, pluginBkpDir)
	}

	return func() {
		shell.FS.Remove(pluginDir)
		shell.FS.Copy(pluginBkpDir, pluginDir)
		shell.Cmd(nil, "wp", "db", "import", dbBkpFile).Must()
	}
}
