// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2014-2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package store

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/ubuntu-core/snappy/asserts"
	"github.com/ubuntu-core/snappy/dirs"
	"github.com/ubuntu-core/snappy/osutil"
	"github.com/ubuntu-core/snappy/progress"
	"github.com/ubuntu-core/snappy/release"
	"github.com/ubuntu-core/snappy/snap"
)

type remoteRepoTestSuite struct {
	store *SnapUbuntuStoreRepository

	origDownloadFunc func(string, io.Writer, *http.Request, progress.Meter) error
}

func TestStore(t *testing.T) { TestingT(t) }

var _ = Suite(&remoteRepoTestSuite{})

func (t *remoteRepoTestSuite) SetUpTest(c *C) {
	t.store = NewUbuntuStoreSnapRepository(nil, "")
	t.origDownloadFunc = download
	dirs.SetRootDir(c.MkDir())
	c.Assert(os.MkdirAll(dirs.SnapSnapsDir, 0755), IsNil)
}

func (t *remoteRepoTestSuite) TearDownTest(c *C) {
	download = t.origDownloadFunc
}

func (t *remoteRepoTestSuite) TestDownloadOK(c *C) {

	download = func(name string, w io.Writer, req *http.Request, pbar progress.Meter) error {
		c.Check(req.URL.String(), Equals, "anon-url")
		w.Write([]byte("I was downloaded"))
		return nil
	}

	snap := &snap.Info{
		Name:            "foo",
		AnonDownloadURL: "anon-url",
		DownloadURL:     "AUTH-URL",
	}
	path, err := t.store.Download(snap, nil)
	c.Assert(err, IsNil)
	defer os.Remove(path)

	content, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "I was downloaded")
}

func (t *remoteRepoTestSuite) TestAuthenticatedDownloadDoesNotUseAnonURL(c *C) {
	home := os.Getenv("HOME")
	os.Setenv("HOME", c.MkDir())
	defer os.Setenv("HOME", home)
	mockStoreToken := StoreToken{TokenName: "meep"}
	err := WriteStoreToken(mockStoreToken)
	c.Assert(err, IsNil)

	download = func(name string, w io.Writer, req *http.Request, pbar progress.Meter) error {
		c.Check(req.URL.String(), Equals, "AUTH-URL")
		w.Write([]byte("I was downloaded"))
		return nil
	}

	snap := &snap.Info{
		Name:            "foo",
		AnonDownloadURL: "anon-url",
		DownloadURL:     "AUTH-URL",
	}
	path, err := t.store.Download(snap, nil)
	c.Assert(err, IsNil)
	defer os.Remove(path)

	content, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "I was downloaded")
}

func (t *remoteRepoTestSuite) TestDownloadFails(c *C) {
	var tmpfile *os.File
	download = func(name string, w io.Writer, req *http.Request, pbar progress.Meter) error {
		tmpfile = w.(*os.File)
		return fmt.Errorf("uh, it failed")
	}

	snap := &snap.Info{
		Name:            "foo",
		AnonDownloadURL: "anon-url",
		DownloadURL:     "AUTH-URL",
	}
	// simulate a failed download
	path, err := t.store.Download(snap, nil)
	c.Assert(err, ErrorMatches, "uh, it failed")
	c.Assert(path, Equals, "")
	// ... and ensure that the tempfile is removed
	c.Assert(osutil.FileExists(tmpfile.Name()), Equals, false)
}

func (t *remoteRepoTestSuite) TestDownloadSyncFails(c *C) {
	var tmpfile *os.File
	download = func(name string, w io.Writer, req *http.Request, pbar progress.Meter) error {
		tmpfile = w.(*os.File)
		w.Write([]byte("sync will fail"))
		err := tmpfile.Close()
		c.Assert(err, IsNil)
		return nil
	}

	snap := &snap.Info{
		Name:            "foo",
		AnonDownloadURL: "anon-url",
		DownloadURL:     "AUTH-URL",
	}
	// simulate a failed sync
	path, err := t.store.Download(snap, nil)
	c.Assert(err, ErrorMatches, "fsync:.*")
	c.Assert(path, Equals, "")
	// ... and ensure that the tempfile is removed
	c.Assert(osutil.FileExists(tmpfile.Name()), Equals, false)
}

func (t *remoteRepoTestSuite) TestUbuntuStoreRepositoryHeaders(c *C) {
	req, err := http.NewRequest("GET", "http://example.com", nil)
	c.Assert(err, IsNil)

	t.store.configureStoreReq(req, "")

	c.Assert(req.Header.Get("X-Ubuntu-Release"), Equals, release.String())
	c.Check(req.Header.Get("Accept"), Equals, "application/hal+json")

	t.store.configureStoreReq(req, "application/json")

	c.Check(req.Header.Get("Accept"), Equals, "application/json")
}

const (
	funkyAppName      = "8nzc1x4iim2xj1g2ul64"
	funkyAppDeveloper = "chipaca"
)

/* acquired via
   curl -s -H "accept: application/hal+json" -H "X-Ubuntu-Release: 15.04-core" https://search.apps.ubuntu.com/api/v1/package/8nzc1x4iim2xj1g2ul64.chipaca | python -m json.tool
*/
const MockDetailsJSON = `{
    "_links": {
        "curies": [
            {
                "href": "https://wiki.ubuntu.com/AppStore/Interfaces/ClickPackageIndex#reltype_{rel}",
                "name": "clickindex",
                "templated": true
            }
        ],
        "self": {
            "href": "https://search.apps.ubuntu.com/api/v1/package/8nzc1x4iim2xj1g2ul64.chipaca"
        }
    },
    "alias": null,
    "allow_unauthenticated": true,
    "anon_download_url": "https://public.apps.ubuntu.com/anon/download/chipaca/8nzc1x4iim2xj1g2ul64.chipaca/8nzc1x4iim2xj1g2ul64.chipaca_42_all.snap",
    "architecture": [
        "all"
    ],
    "binary_filesize": 65375,
    "blacklist_country_codes": [
        "AX"
    ],
    "channel": "edge",
    "changelog": "",
    "click_framework": [],
    "click_version": "0.1",
    "company_name": "",
    "content": "application",
    "date_published": "2015-04-15T18:34:40.060874Z",
    "department": [
        "food-drink"
    ],
    "summary": "hello world example",
    "description": "Returns for store credit only.\nThis is a simple hello world example.",
    "developer_name": "John Lenton",
    "download_sha512": "5364253e4a988f4f5c04380086d542f410455b97d48cc6c69ca2a5877d8aef2a6b2b2f83ec4f688cae61ebc8a6bf2cdbd4dbd8f743f0522fc76540429b79df42",
    "download_url": "https://public.apps.ubuntu.com/download/chipaca/8nzc1x4iim2xj1g2ul64.chipaca/8nzc1x4iim2xj1g2ul64.chipaca_42_all.snap",
    "framework": [],
    "icon_url": "https://myapps.developer.ubuntu.com/site_media/appmedia/2015/04/hello.svg_Dlrd3L4.png",
    "icon_urls": {
        "256": "https://myapps.developer.ubuntu.com/site_media/appmedia/2015/04/hello.svg_Dlrd3L4.png"
    },
    "id": 2333,
    "keywords": [],
    "last_updated": "2015-04-15T18:30:16Z",
    "license": "Proprietary",
    "name": "8nzc1x4iim2xj1g2ul64.chipaca",
    "origin": "chipaca",
    "package_name": "8nzc1x4iim2xj1g2ul64",
    "price": 0.0,
    "prices": {},
    "publisher": "John Lenton",
    "ratings_average": 0.0,
    "release": [
        "15.04-core"
    ],
    "revision": 15,
    "screenshot_url": null,
    "screenshot_urls": [],
    "status": "Published",
    "stores": {
        "ubuntu": {
            "status": "Published"
        }
    },
    "support_url": "http://lmgtfy.com",
    "terms_of_service": "",
    "title": "Returns for store credit only.",
    "translations": {},
    "version": "42",
    "video_embedded_html_urls": [],
    "video_urls": [],
    "website": "",
    "whitelist_country_codes": []
}
`

func (t *remoteRepoTestSuite) TestUbuntuStoreRepositoryDetails(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// no store ID by default
		storeID := r.Header.Get("X-Ubuntu-Store")
		c.Check(storeID, Equals, "")

		c.Check(r.URL.Path, Equals, fmt.Sprintf("/details/%s.%s/edge", funkyAppName, funkyAppDeveloper))
		io.WriteString(w, MockDetailsJSON)
	}))

	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	var err error
	detailsURI, err := url.Parse(mockServer.URL + "/details/")
	c.Assert(err, IsNil)
	cfg := SnapUbuntuStoreConfig{
		DetailsURI: detailsURI,
	}
	repo := NewUbuntuStoreSnapRepository(&cfg, "")
	c.Assert(repo, NotNil)

	// the actual test
	result, err := repo.Snap(funkyAppName+"."+funkyAppDeveloper, "edge")
	c.Assert(err, IsNil)
	c.Check(result.Name, Equals, funkyAppName)
	c.Check(result.Developer, Equals, funkyAppDeveloper)
	c.Check(result.Version, Equals, "42")
	c.Check(result.Sha512, Equals, "5364253e4a988f4f5c04380086d542f410455b97d48cc6c69ca2a5877d8aef2a6b2b2f83ec4f688cae61ebc8a6bf2cdbd4dbd8f743f0522fc76540429b79df42")
	c.Check(result.Size, Equals, int64(65375))
	c.Check(result.Channel, Equals, "edge")
	c.Check(result.Description, Equals, "Returns for store credit only.\nThis is a simple hello world example.")
	c.Check(result.Summary, Equals, "hello world example")
}

const MockNoDetailsJSON = `{"errors": ["No such package"], "result": "error"}`

func (t *remoteRepoTestSuite) TestUbuntuStoreRepositoryNoDetails(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, Equals, "/details/no-such-pkg/edge")
		w.WriteHeader(404)
		io.WriteString(w, MockNoDetailsJSON)
	}))

	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	var err error
	detailsURI, err := url.Parse(mockServer.URL + "/details/")
	c.Assert(err, IsNil)
	cfg := SnapUbuntuStoreConfig{
		DetailsURI: detailsURI,
	}
	repo := NewUbuntuStoreSnapRepository(&cfg, "")
	c.Assert(repo, NotNil)

	// the actual test
	result, err := repo.Snap("no-such-pkg", "edge")
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)
}

func (t *remoteRepoTestSuite) TestStructFields(c *C) {
	type s struct {
		Foo int `json:"hello"`
		Bar int `json:"potato,stuff"`
	}
	c.Assert(getStructFields(s{}), DeepEquals, []string{"hello", "potato"})
}

/* acquired via:
curl -s -H 'accept: application/hal+json' -H "X-Ubuntu-Release: 15.04-core" -H "X-Ubuntu-Architecture: amd64" "https://search.apps.ubuntu.com/api/v1/search?q=8nzc1x4iim2xj1g2ul64&fields=publisher,package_name,developer,title,icon_url,prices,content,ratings_average,version,anon_download_url,download_url,download_sha512,last_updated,binary_filesize,support_url,revision" | python -m json.tool
*/
const MockSearchJSON = `{
    "_embedded": {
        "clickindex:package": [
            {
                "_links": {
                    "self": {
                        "href": "https://search.apps.ubuntu.com/api/v1/package/8nzc1x4iim2xj1g2ul64.chipaca"
                    }
                },
                "anon_download_url": "https://public.apps.ubuntu.com/anon/download/chipaca/8nzc1x4iim2xj1g2ul64.chipaca/8nzc1x4iim2xj1g2ul64.chipaca_42_all.snap",
                "binary_filesize": 65375,
                "content": "application",
                "download_sha512": "5364253e4a988f4f5c04380086d542f410455b97d48cc6c69ca2a5877d8aef2a6b2b2f83ec4f688cae61ebc8a6bf2cdbd4dbd8f743f0522fc76540429b79df42",
                "download_url": "https://public.apps.ubuntu.com/download/chipaca/8nzc1x4iim2xj1g2ul64.chipaca/8nzc1x4iim2xj1g2ul64.chipaca_42_all.snap",
                "icon_url": "https://myapps.developer.ubuntu.com/site_media/appmedia/2015/04/hello.svg_Dlrd3L4.png",
                "last_updated": "2015-04-15T18:30:16Z",
                "origin": "chipaca",
                "package_name": "8nzc1x4iim2xj1g2ul64",
                "prices": {},
                "publisher": "John Lenton",
                "ratings_average": 0.0,
                "revision": 7,
                "support_url": "http://lmgtfy.com",
                "title": "Returns for store credit only.",
                "version": "42"
            }
        ]
    },
    "_links": {
        "curies": [
            {
                "href": "https://wiki.ubuntu.com/AppStore/Interfaces/ClickPackageIndex#reltype_{rel}",
                "name": "clickindex",
                "templated": true
            }
        ],
        "self": {
            "href": "https://search.apps.ubuntu.com/api/v1/search?q=8nzc1x4iim2xj1g2ul64&fields=publisher,package_name,developer,title,icon_url,prices,content,ratings_average,version,anon_download_url,download_url,download_sha512,last_updated,binary_filesize,support_url,revision"
        }
    }
}
`

func (t *remoteRepoTestSuite) TestUbuntuStoreFind(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Check(r.URL.RawQuery, Equals, "q=name%3Afoo")
		io.WriteString(w, MockSearchJSON)
	}))
	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	var err error
	searchURI, err := url.Parse(mockServer.URL)
	c.Assert(err, IsNil)
	cfg := SnapUbuntuStoreConfig{
		SearchURI: searchURI,
	}
	repo := NewUbuntuStoreSnapRepository(&cfg, "")
	c.Assert(repo, NotNil)

	snaps, err := repo.FindSnaps("foo", "")
	c.Assert(err, IsNil)
	c.Assert(snaps, HasLen, 1)
	c.Check(snaps[0].Name, Equals, funkyAppName)
}

/* acquired via:
curl -s --data-binary '{"name":["8nzc1x4iim2xj1g2ul64.chipaca"]}'  -H 'content-type: application/json' https://search.apps.ubuntu.com/api/v1/click-metadata
*/
const MockUpdatesJSON = `[
    {
        "status": "Published",
        "name": "8nzc1x4iim2xj1g2ul64.chipaca",
        "package_name": "8nzc1x4iim2xj1g2ul64",
        "origin": "chipaca",
        "changelog": "",
        "icon_url": "https://myapps.developer.ubuntu.com/site_media/appmedia/2015/04/hello.svg_Dlrd3L4.png",
        "title": "Returns for store credit only.",
        "binary_filesize": 65375,
        "anon_download_url": "https://public.apps.ubuntu.com/anon/download/chipaca/8nzc1x4iim2xj1g2ul64.chipaca/8nzc1x4iim2xj1g2ul64.chipaca_42_all.snap",
        "allow_unauthenticated": true,
        "revision": 3,
        "version": "42",
        "download_url": "https://public.apps.ubuntu.com/download/chipaca/8nzc1x4iim2xj1g2ul64.chipaca/8nzc1x4iim2xj1g2ul64.chipaca_42_all.snap",
        "download_sha512": "5364253e4a988f4f5c04380086d542f410455b97d48cc6c69ca2a5877d8aef2a6b2b2f83ec4f688cae61ebc8a6bf2cdbd4dbd8f743f0522fc76540429b79df42"
    }
]`

func (t *remoteRepoTestSuite) TestUbuntuStoreRepositoryUpdates(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonReq, err := ioutil.ReadAll(r.Body)
		c.Assert(err, IsNil)
		c.Assert(string(jsonReq), Equals, `{"name":["`+funkyAppName+`"]}`)
		io.WriteString(w, MockUpdatesJSON)
	}))

	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	var err error
	bulkURI, err := url.Parse(mockServer.URL + "/updates/")
	c.Assert(err, IsNil)
	cfg := SnapUbuntuStoreConfig{
		BulkURI: bulkURI,
	}
	repo := NewUbuntuStoreSnapRepository(&cfg, "")
	c.Assert(repo, NotNil)

	results, err := repo.Updates([]string{funkyAppName})
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 1)
	c.Assert(results[0].Name, Equals, funkyAppName)
	c.Assert(results[0].Revision, Equals, 3)
	c.Assert(results[0].Version, Equals, "42")
}

func (t *remoteRepoTestSuite) TestStructFieldsSurvivesNoTag(c *C) {
	type s struct {
		Foo int `json:"hello"`
		Bar int
	}
	c.Assert(getStructFields(s{}), DeepEquals, []string{"hello"})
}

func (t *remoteRepoTestSuite) TestCpiURLDependsOnEnviron(c *C) {
	c.Assert(os.Setenv("SNAPPY_USE_STAGING_CPI", ""), IsNil)
	before := cpiURL()

	c.Assert(os.Setenv("SNAPPY_USE_STAGING_CPI", "1"), IsNil)
	defer os.Setenv("SNAPPY_USE_STAGING_CPI", "")
	after := cpiURL()

	c.Check(before, Not(Equals), after)
}

func (t *remoteRepoTestSuite) TestAuthURLDependsOnEnviron(c *C) {
	c.Assert(os.Setenv("SNAPPY_USE_STAGING_CPI", ""), IsNil)
	before := authURL()

	c.Assert(os.Setenv("SNAPPY_USE_STAGING_CPI", "1"), IsNil)
	defer os.Setenv("SNAPPY_USE_STAGING_CPI", "")
	after := authURL()

	c.Check(before, Not(Equals), after)
}

func (t *remoteRepoTestSuite) TestAssertsURLDependsOnEnviron(c *C) {
	c.Assert(os.Setenv("SNAPPY_USE_STAGING_SAS", ""), IsNil)
	before := assertsURL()

	c.Assert(os.Setenv("SNAPPY_USE_STAGING_SAS", "1"), IsNil)
	defer os.Setenv("SNAPPY_USE_STAGING_SAS", "")
	after := assertsURL()

	c.Check(before, Not(Equals), after)
}

func (t *remoteRepoTestSuite) TestMyAppsURLDependsOnEnviron(c *C) {
	c.Assert(os.Setenv("SNAPPY_USE_STAGING_MYAPPS", ""), IsNil)
	before := myappsURL()

	c.Assert(os.Setenv("SNAPPY_USE_STAGING_MYAPPS", "1"), IsNil)
	defer os.Setenv("SNAPPY_USE_STAGING_MYAPPS", "")
	after := myappsURL()

	c.Check(before, Not(Equals), after)
}

func (t *remoteRepoTestSuite) TestDefaultConfig(c *C) {
	c.Check(strings.HasPrefix(defaultConfig.SearchURI.String(), "https://search.apps.ubuntu.com/api/v1/search?"), Equals, true)
	c.Check(defaultConfig.DetailsURI.String(), Equals, "https://search.apps.ubuntu.com/api/v1/package/")
	c.Check(strings.HasPrefix(defaultConfig.BulkURI.String(), "https://search.apps.ubuntu.com/api/v1/click-metadata?"), Equals, true)
	c.Check(defaultConfig.AssertionsURI.String(), Equals, "https://assertions.ubuntu.com/v1/assertions/")
}

var testAssertion = `type: snap-declaration
authority-id: super
series: 16
snap-id: snapidfoo
gates: 
publisher-id: devidbaz
snap-name: mysnap
timestamp: 2016-03-30T12:22:16Z

openpgp wsBcBAABCAAQBQJW+8VBCRDWhXkqAWcrfgAAQ9gIABZFgMPByJZeUE835FkX3/y2hORn
AzE3R1ktDkQEVe/nfVDMACAuaw1fKmUS4zQ7LIrx/AZYw5i0vKVmJszL42LBWVsqR0+p9Cxebzv9
U2VUSIajEsUUKkBwzD8wxFzagepFlScif1NvCGZx0vcGUOu0Ent0v+gqgAv21of4efKqEW7crlI1
T/A8LqZYmIzKRHGwCVucCyAUD8xnwt9nyWLgLB+LLPOVFNK8SR6YyNsX05Yz1BUSndBfaTN8j/k8
8isKGZE6P0O9ozBbNIAE8v8NMWQegJ4uWuil7D3psLkzQIrxSypk9TrQ2GlIG2hJdUovc5zBuroe
xS4u9rVT6UY=`

func (t *remoteRepoTestSuite) TestUbuntuStoreRepositoryAssertion(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Check(r.Header.Get("Accept"), Equals, "application/x.ubuntu.assertion")
		c.Check(r.URL.Path, Equals, "/assertions/snap-declaration/16/snapidfoo")
		io.WriteString(w, testAssertion)
	}))

	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	var err error
	assertionsURI, err := url.Parse(mockServer.URL + "/assertions/")
	c.Assert(err, IsNil)
	cfg := SnapUbuntuStoreConfig{
		AssertionsURI: assertionsURI,
	}
	repo := NewUbuntuStoreSnapRepository(&cfg, "")

	a, err := repo.Assertion(asserts.SnapDeclarationType, "16", "snapidfoo")
	c.Assert(err, IsNil)
	c.Check(a, NotNil)
	c.Check(a.Type(), Equals, asserts.SnapDeclarationType)
}

func (t *remoteRepoTestSuite) TestUbuntuStoreRepositoryNotFound(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Check(r.Header.Get("Accept"), Equals, "application/x.ubuntu.assertion")
		c.Check(r.URL.Path, Equals, "/assertions/snap-declaration/16/snapidfoo")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		io.WriteString(w, `{"status": 404,"title": "not found"}`)
	}))

	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	var err error
	assertionsURI, err := url.Parse(mockServer.URL + "/assertions/")
	c.Assert(err, IsNil)
	cfg := SnapUbuntuStoreConfig{
		AssertionsURI: assertionsURI,
	}
	repo := NewUbuntuStoreSnapRepository(&cfg, "")

	_, err = repo.Assertion(asserts.SnapDeclarationType, "16", "snapidfoo")
	c.Check(err, Equals, ErrAssertionNotFound)
}
