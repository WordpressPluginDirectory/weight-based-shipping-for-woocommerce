package main

import (
	"cmp"
	"e2e/src/lib"
	shellpkg "e2e/src/lib/shell"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/playwright-community/playwright-go"
	"log"
	"os"
	"testing"
	"time"
)

const WebsiteDomain = "weightbasedshipping.com"

var Env = struct {
	Headed      bool
	WpContainer string
	WpBaseUrl   string
}{
	Headed:      os.Getenv("headed") != "",
	WpContainer: os.Getenv("wp_container"),
	WpBaseUrl:   "http://" + cmp.Or(os.Getenv("wp_host"), "localhost"),
}

var Shell *shellpkg.Shell
var Meta *lib.PluginMeta
var Restore func()

func init() {

	if Env.WpContainer == "" {
		log.SetFlags(0)
		log.Print("wp container required")
		os.Exit(1)
	}

	Shell = shellpkg.NewShell(must(client.NewClientWithOpts(client.FromEnv)), Env.WpContainer, "www-data")
	Meta = lib.PluginMetaFromShell(Shell)
	Restore = lib.Backup(Shell, Meta.Dir)
}

const timeout = 20000

var Expect = playwright.NewPlaywrightAssertions(timeout).Locator

func StartBrowser(t *testing.T) playwright.Page {

	pw := must(playwright.Run())

	browser := must(pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(!Env.Headed),
	}))

	page := must(browser.NewPage(playwright.BrowserNewPageOptions{
		BaseURL: playwright.String(Env.WpBaseUrl),
		RecordVideo: &playwright.RecordVideo{
			Dir: "/tmp/video/",
		},
	}))
	page.SetDefaultTimeout(timeout)

	t.Cleanup(func() {
		_ = page.Close()
		video(t, page)
	})

	return page
}

func video(t *testing.T, page playwright.Page) {

	var op string
	done := make(chan struct{})
	if t.Failed() {
		op = "retrieval"
		go func() {
			path, _ := page.Video().Path()
			fmt.Println("video: ", path)
			close(done)
		}()
	} else {
		op = "deletion"
		go func() {
			_ = page.Video().Delete()
			close(done)
		}()
	}

	select {
	case <-time.After(5 * time.Second):
		fmt.Println(fmt.Sprintf("video %s timeout", op))
	case <-done:
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
