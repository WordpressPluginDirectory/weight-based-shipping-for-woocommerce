package lib

import (
	"bytes"
	"e2e/src/lib/shell"
	"encoding/json"
	"strings"
)

func ActivateFakeUpdate(shell *shell.Shell, pluginID, pluginDir string) func() {

	const zip = "fakeupdate.zip"
	const metajson = "updates" // the plugin's updater fetches <apiEndpoint>/updates

	tmp := strings.TrimSpace(shell.Cmd(nil, "mktemp", "-d").Must().Stdout.String())
	defer shell.FS.Remove(tmp)
	shell.FS.Copy(pluginDir, tmp)
	shell.Cmd(nil, "sed", "-i", "s/Version: .*/Version: 999/", tmp+"/"+pluginID).Must()
	shell.ShellCmd(nil, `(cd "$1" && zip -r - .) > "$2"`, tmp, zip).Must()

	shell.FS.Write(metajson, bytes.NewReader(must(json.Marshal(map[string]string{
		"name":         "WooCommerce Weight Based Shipping +",
		"version":      "999",
		"download_url": "http://localhost/" + zip,
	}))))

	shell.FS.Write(pluginDir+"/server/.config.php", strings.NewReader(`<?php return ['apiEndpoint' => 'http://localhost'];`))

	return func() {
		shell.FS.Remove(metajson, zip)
	}
}
