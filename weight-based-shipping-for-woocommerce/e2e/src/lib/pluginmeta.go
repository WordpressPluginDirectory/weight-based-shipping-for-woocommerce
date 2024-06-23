package lib

import (
	"bytes"
	"e2e/src/lib/shell"
	"encoding/json"
	"fmt"
	urlpkg "net/url"
	"path"
	"regexp"
	"strings"
)

type PluginMeta struct {
	ID         string
	Title      string
	Version    string
	PluginFile string
	Dir        string
	Edition    Edition
}

type Edition string

const (
	Free    Edition = "free"
	PlusWbs Edition = "plus-wbs"
	PlusCc  Edition = "plus-cc"
)

var pluginSlugs = []string{"wc-weight-based-shipping", "weight-based-shipping-for-woocommerce"}

func PluginMetaFromShell(shell *shell.Shell) *PluginMeta {

	var found int
	var plugins pluginListJson
	var response *bytes.Buffer
	for _, slug := range pluginSlugs {

		res := shell.Cmd(nil, "wp", "plugin", "list", "--fields=file,version,title", "--name="+slug, "--format=json").Must()
		response = res.Stdout

		var l pluginListJson
		must(0, json.Unmarshal(res.Stdout.Bytes(), &l))
		if len(l) != 0 {
			plugins = l
			found++
		}
	}

	if found == 0 {
		panic("none of the specified slugs were found")
	}
	if found > 1 {
		panic("multiple of the specified slugs were found")
	}

	defer func() {
		if r := recover(); r != nil {
			s := r.(string)
			s = "wp plugin list: " + s + "; response: " + response.String()
			panic(s)
		}
	}()

	if len(plugins) != 1 {
		panic(fmt.Sprintf("requested one plugin, got %d", len(plugins)))
	}

	info := plugins[0]
	if info.File == "" {
		panic("file fiels is empty")
	}
	if info.Title == "" {
		panic("title field is empty")
	}
	if info.Version == "" {
		panic("version field is empty")
	}

	edition := detectEdition(shell, info.File)

	return NewPluginMeta(info.File, info.Title, info.Version, edition)
}

func NewPluginMeta(id, title, version string, edition Edition) *PluginMeta {

	id = strings.TrimSpace(id)
	if id == "" {
		panic("plugin meta: empty plugin id")
	}

	version = strings.TrimSpace(version)
	if version == "" {
		panic("plugin meta: empty plugin version")
	}

	pluginFile := pluginFileRelativeToWpRoot(id)
	dir := path.Dir(pluginFile)

	return &PluginMeta{
		ID:         id,
		Title:      title,
		Version:    version,
		PluginFile: pluginFile,
		Dir:        dir,
		Edition:    edition,
	}
}

type pluginListJson []struct {
	File    string
	Title   string
	Version string
}

func pluginFileRelativeToWpRoot(pluginFileId string) string {
	return fmt.Sprintf("wp-content/plugins/%s", pluginFileId)
}

var pluginURIHeader = regexp.MustCompile("Plugin URI: (.*)")

func detectEdition(shell *shell.Shell, pluginFile string) Edition {

	m := pluginURIHeader.FindSubmatch(shell.FS.Read(pluginFileRelativeToWpRoot(pluginFile)).Bytes())
	if m == nil || len(m[1]) == 0 {
		panic("Plugin URI not found")
	}

	_url := string(m[1])

	url, err := urlpkg.Parse(_url)
	if err != nil {
		panic("Plugin URI invalid: " + err.Error())
	}

	var edition Edition
	switch url.Host {
	case "wordpress.org":
		edition = Free
	case "weightbasedshipping.com":
		edition = PlusWbs
	case "codecanyon.net":
		edition = PlusCc
	default:
		panic("Plugin URI refers unknown domain: " + _url)
	}

	return edition
}
