package main

import (
	"e2e/src/lib"
	"fmt"
	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
)

func TestUpdate(t *testing.T) {

	if Meta.Edition == lib.PlusCc {
		t.Skipf("%s edition is not capable of self-updating", Meta.Edition)
	}

	const obsoleteVersion = "0.0.1"

	var remote = lib.NewPluginRemote(Shell, Meta)

	var page playwright.Page
	var pluginsPage *tPluginsPage

	beforeEach := func(t *testing.T) {

		Restore()

		page = StartBrowser(t)
		must(page.Goto("/wp-admin/plugins.php"))

		pluginsPage = newPluginsPage(page, Meta)
	}

	makeObsoleteAndCheckForUpdates := func(t *testing.T) {
		remote.ReplaceVersion(obsoleteVersion)
		must(0, pluginsPage.CheckForUpdateLink.Click())
		require.NoError(t, Expect(pluginsPage.NewVersionIsAvailableMsg).ToBeVisible())
		require.NoError(t, Expect(pluginsPage.UpdateNowLink).ToBeVisible())
	}

	getActualPluginVersion := func() string {
		return lib.PluginMetaFromShell(Shell).Version
	}

	testLicenseError := func(t *testing.T, license *string) {

		remote.ReplaceLicense(license)

		makeObsoleteAndCheckForUpdates(t)

		pluginsPage.UpdateNow(page)

		require.NoError(t, Expect(pluginsPage.UpdateFailMsg).ToBeVisible())
		require.NoError(t, Expect(pluginsPage.UpdatedSuccessMsg).Not().ToBeVisible())

		require.Equal(t, getActualPluginVersion(), obsoleteVersion)
	}

	if Meta.Edition == lib.Free {
		t.Run("'update' to the version published at wp.org (catch fatal errors on downgrading)", func(t *testing.T) {
			beforeEach(t)

			// The free version does not show the 'Check for updates' link on the plugins page.

			// WordPress does not always refetch the plugin updates on the Plugins page.
			// But it does so for the Updates page.

			remote.ReplaceVersion(obsoleteVersion)

			url := page.URL()
			must(page.Goto("/wp-admin/update-core.php"))
			must(page.Goto(url))

			pluginsPage.UpdateNow(page)
			require.NoError(t, Expect(pluginsPage.UpdatedSuccessMsg).ToBeVisible())
			require.NotEqual(t, obsoleteVersion, getActualPluginVersion())
		})
	}

	if Meta.Edition != lib.PlusWbs {
		return
	}

	t.Run("update", func(t *testing.T) {

		t.Run("'update' to this very same version (make sure this version is able to update)", func(t *testing.T) {
			beforeEach(t)
			remote.ReplaceLicense(&licenses.ActiveLicense)
			defer lib.ActivateFakeUpdate(Shell, Meta.ID, Meta.Dir)()

			must(0, pluginsPage.CheckForUpdateLink.Click())
			require.NoError(t, Expect(pluginsPage.NewVersionIsAvailableMsg).ToBeVisible())

			pluginsPage.UpdateNow(page)
			require.NoError(t, Expect(pluginsPage.UpdatedSuccessMsg).ToBeVisible())
			require.Equal(t, "999", getActualPluginVersion())
		})

		t.Run("'update' to the previously released version (catch fatal errors on downgrading)", func(t *testing.T) {
			beforeEach(t)

			remote.ReplaceLicense(&licenses.ActiveLicense)
			makeObsoleteAndCheckForUpdates(t)

			pluginsPage.UpdateNow(page)
			require.NoError(t, Expect(pluginsPage.UpdatedSuccessMsg).ToBeVisible())
			require.NotEqual(t, obsoleteVersion, getActualPluginVersion())

			t.Run("update from the previously released version (ensure users can actually update the plugin)", func(t *testing.T) {
				defer lib.ActivateFakeUpdate(Shell, Meta.ID, Meta.Dir)()

				must(0, pluginsPage.CheckForUpdateLink.Click())
				require.NoError(t, Expect(pluginsPage.NewVersionIsAvailableMsg).ToBeVisible())

				pluginsPage.UpdateNow(page)
				require.NoError(t, Expect(pluginsPage.UpdatedSuccessMsg).ToBeVisible(playwright.LocatorAssertionsToBeVisibleOptions{Timeout: playwright.Float(15000)}))
				require.Equal(t, "999", getActualPluginVersion())
			})
		})
	})

	t.Run("update fail due to license issues", func(t *testing.T) {

		t.Run("no license", func(t *testing.T) {
			beforeEach(t)
			testLicenseError(t, nil)
			require.NoError(t, Expect(pluginsPage.UpdateFailMsg).ToContainText("No license information provided."))
		})

		t.Run("empty license", func(t *testing.T) {
			beforeEach(t)
			testLicenseError(t, &licenses.EmptyLicense)
			require.NoError(t, Expect(pluginsPage.UpdateFailMsg).ToContainText("No license information provided."))
		})

		t.Run("invalid license", func(t *testing.T) {
			beforeEach(t)
			testLicenseError(t, &licenses.InvalidLicense)
			require.NoError(t, Expect(pluginsPage.UpdateFailMsg).ToContainText("Invalid license information."))
		})

		t.Run("expired license", func(t *testing.T) {
			beforeEach(t)
			testLicenseError(t, &licenses.ExpiredLicense)
			hasText := Expect(pluginsPage.UpdateFailMsg).ToContainText
			require.NoError(t, hasText("Your license expired on"))
			require.NoError(t, hasText(fmt.Sprintf(`Renew: https://%s/renew?license=%s`, WebsiteDomain, licenses.ExpiredLicense)))
		})

		t.Run("revoked license", func(t *testing.T) {
			beforeEach(t)
			testLicenseError(t, &licenses.RevokedLicense)
			require.NoError(t, Expect(pluginsPage.UpdateFailMsg).ToContainText(regexp.MustCompile("Your license revoked on .* with reason")))
		})
	})
}

type tPluginsPage struct {
	CheckForUpdateLink       playwright.Locator
	NewVersionIsAvailableMsg playwright.Locator
	UpdateNowLink            playwright.Locator
	UpdatedSuccessMsg        playwright.Locator
	UpdateFailMsg            playwright.Locator
}

func newPluginsPage(page playwright.Page, meta *lib.PluginMeta) *tPluginsPage {
	rows := page.Locator(fmt.Sprintf(`[data-plugin="%s"]`, meta.ID))
	return &tPluginsPage{
		CheckForUpdateLink:       rows.GetByText("Check for updates"),
		NewVersionIsAvailableMsg: rows.GetByText(fmt.Sprintf(`new version of %s available`, meta.Title)),
		UpdateNowLink:            rows.GetByRole("link", playwright.LocatorGetByRoleOptions{Name: fmt.Sprintf(`Update %s now`, meta.Title)}),
		UpdatedSuccessMsg:        rows.GetByText("Updated!"), // aria-label: WooCommerce Tree Table Rate Shipping updated!
		UpdateFailMsg:            rows.GetByText("Update failed:"),
	}
}

func (pp *tPluginsPage) UpdateNow(page playwright.Page) {
	// load scripts handling the update-now link; otherwise it might get us to another page and break our assertions
	must(0, page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateLoad}))
	must(0, pp.UpdateNowLink.Click())
}

var licenses = struct {
	EmptyLicense   string
	InvalidLicense string
	ActiveLicense  string
	ExpiredLicense string
	RevokedLicense string
}{
	"",
	"test_invalid_license",
	"02bb4e7c1a924b1cbeb1ad7e54d1ec0e",
	"12bb4e7c1a924b1cbeb1ad7e54d1ec0e",
	"22bb4e7c1a924b1cbeb1ad7e54d1ec0e",
}
