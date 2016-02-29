// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2014-2015 Canonical Ltd
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

package helpers

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type HTestSuite struct{}

var _ = Suite(&HTestSuite{})

func (ts *HTestSuite) TestMakeMapFromEnvList(c *C) {
	envList := []string{
		"PATH=/usr/bin:/bin",
		"DBUS_SESSION_BUS_ADDRESS=unix:abstract=something1234",
	}
	envMap := MakeMapFromEnvList(envList)
	c.Assert(envMap, DeepEquals, map[string]string{
		"PATH": "/usr/bin:/bin",
		"DBUS_SESSION_BUS_ADDRESS": "unix:abstract=something1234",
	})
}

func (ts *HTestSuite) TestMakeMapFromEnvListInvalidInput(c *C) {
	envList := []string{
		"nonsesne",
	}
	envMap := MakeMapFromEnvList(envList)
	c.Assert(envMap, DeepEquals, map[string]string(nil))
}

func (ts *HTestSuite) TestMakeRandomString(c *C) {
	// for our tests
	rand.Seed(1)

	s1 := MakeRandomString(10)
	c.Assert(s1, Equals, "gmwJgSapGA")

	s2 := MakeRandomString(5)
	c.Assert(s2, Equals, "tLMod")
}

func (ts *HTestSuite) TestAtomicWriteFile(c *C) {
	tmpdir := c.MkDir()

	p := filepath.Join(tmpdir, "foo")
	err := AtomicWriteFile(p, []byte("canary"), 0644, 0)
	c.Assert(err, IsNil)

	content, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "canary")

	// no files left behind!
	d, err := ioutil.ReadDir(tmpdir)
	c.Assert(err, IsNil)
	c.Assert(len(d), Equals, 1)
}

func (ts *HTestSuite) TestAtomicWriteFilePermissions(c *C) {
	tmpdir := c.MkDir()

	p := filepath.Join(tmpdir, "foo")
	err := AtomicWriteFile(p, []byte(""), 0600, 0)
	c.Assert(err, IsNil)

	st, err := os.Stat(p)
	c.Assert(err, IsNil)
	c.Assert(st.Mode()&os.ModePerm, Equals, os.FileMode(0600))
}

func (ts *HTestSuite) TestAtomicWriteFileOverwrite(c *C) {
	tmpdir := c.MkDir()
	p := filepath.Join(tmpdir, "foo")
	c.Assert(ioutil.WriteFile(p, []byte("hello"), 0644), IsNil)
	c.Assert(AtomicWriteFile(p, []byte("hi"), 0600, 0), IsNil)

	content, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "hi")
}

func (ts *HTestSuite) TestAtomicWriteFileSymlinkNoFollow(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	s := filepath.Join(tmpdir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink(s, p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	err := AtomicWriteFile(p, []byte("hi"), 0600, 0)
	c.Assert(err, NotNil)
}

func (ts *HTestSuite) TestAtomicWriteFileAbsoluteSymlinks(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	s := filepath.Join(tmpdir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink(s, p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	err := AtomicWriteFile(p, []byte("hi"), 0600, AtomicWriteFollow)
	c.Assert(err, IsNil)

	content, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "hi")
}

func (ts *HTestSuite) TestAtomicWriteFileOverwriteAbsoluteSymlink(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	s := filepath.Join(tmpdir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink(s, p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	c.Assert(ioutil.WriteFile(s, []byte("hello"), 0644), IsNil)
	c.Assert(AtomicWriteFile(p, []byte("hi"), 0600, AtomicWriteFollow), IsNil)

	content, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "hi")
}

func (ts *HTestSuite) TestAtomicWriteFileRelativeSymlinks(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink("../foo", p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	err := AtomicWriteFile(p, []byte("hi"), 0600, AtomicWriteFollow)
	c.Assert(err, IsNil)

	content, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "hi")
}

func (ts *HTestSuite) TestAtomicWriteFileOverwriteRelativeSymlink(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	s := filepath.Join(tmpdir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink("../foo", p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	c.Assert(ioutil.WriteFile(s, []byte("hello"), 0644), IsNil)
	c.Assert(AtomicWriteFile(p, []byte("hi"), 0600, AtomicWriteFollow), IsNil)

	content, err := ioutil.ReadFile(p)
	c.Assert(err, IsNil)
	c.Assert(string(content), Equals, "hi")
}

func (ts *HTestSuite) TestAtomicWriteFileNoOverwriteTmpExisting(c *C) {
	tmpdir := c.MkDir()
	realMakeRandomString := MakeRandomString
	defer func() { MakeRandomString = realMakeRandomString }()
	MakeRandomString = func(n int) string {
		// chosen by fair dice roll.
		// guranteed to be random.
		return "4"
	}

	p := filepath.Join(tmpdir, "foo")
	err := ioutil.WriteFile(p+".4", []byte(""), 0644)
	c.Assert(err, IsNil)

	err = AtomicWriteFile(p, []byte(""), 0600, 0)
	c.Assert(err, ErrorMatches, "open .*: file exists")
}

func (ts *HTestSuite) TestCurrentHomeDirHOMEenv(c *C) {
	tmpdir := c.MkDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)

	os.Setenv("HOME", tmpdir)
	home, err := CurrentHomeDir()
	c.Assert(err, IsNil)
	c.Assert(home, Equals, tmpdir)
}

func (ts *HTestSuite) TestCurrentHomeDirNoHomeEnv(c *C) {
	oldHome := os.Getenv("HOME")
	if oldHome == "/sbuild-nonexistent" {
		c.Skip("running in schroot this test won't work")
	}
	defer os.Setenv("HOME", oldHome)

	os.Setenv("HOME", "")
	home, err := CurrentHomeDir()
	c.Assert(err, IsNil)
	c.Assert(home, Equals, oldHome)
}

func skipOnMissingDevKmsg(c *C) {
	_, err := os.Stat("/dev/kmsg")
	if err != nil {
		c.Skip("Can not stat /dev/kmsg")
	}
}

func (ts *HTestSuite) TestGetattr(c *C) {
	T := struct {
		S string
		I int
	}{
		S: "foo",
		I: 42,
	}
	// works on values
	c.Assert(Getattr(T, "S").(string), Equals, "foo")
	c.Assert(Getattr(T, "I").(int), Equals, 42)
	// works for pointers too
	c.Assert(Getattr(&T, "S").(string), Equals, "foo")
	c.Assert(Getattr(&T, "I").(int), Equals, 42)
}

func makeTestFiles(c *C, srcDir, destDir string) {
	// a new file
	err := ioutil.WriteFile(filepath.Join(srcDir, "new"), []byte(nil), 0644)
	c.Assert(err, IsNil)

	// a existing file that needs update
	err = ioutil.WriteFile(filepath.Join(destDir, "existing-update"), []byte("old-content"), 0644)
	c.Assert(err, IsNil)
	err = ioutil.WriteFile(filepath.Join(srcDir, "existing-update"), []byte("some-new-content"), 0644)
	c.Assert(err, IsNil)

	// existing file that needs no update
	err = ioutil.WriteFile(filepath.Join(srcDir, "existing-unchanged"), []byte(nil), 0644)
	c.Assert(err, IsNil)
	err = exec.Command("cp", "-a", filepath.Join(srcDir, "existing-unchanged"), filepath.Join(destDir, "existing-unchanged")).Run()
	c.Assert(err, IsNil)

	// a file that needs removal
	err = ioutil.WriteFile(filepath.Join(destDir, "to-be-deleted"), []byte(nil), 0644)
	c.Assert(err, IsNil)
}

func compareDirs(c *C, srcDir, destDir string) {
	d1, err := exec.Command("ls", "-al", srcDir).CombinedOutput()
	c.Assert(err, IsNil)
	d2, err := exec.Command("ls", "-al", destDir).CombinedOutput()
	c.Assert(err, IsNil)
	c.Assert(string(d1), Equals, string(d2))
	// ensure content got updated
	c1, err := exec.Command("sh", "-c", fmt.Sprintf("find %s -type f |xargs cat", srcDir)).CombinedOutput()
	c.Assert(err, IsNil)
	c2, err := exec.Command("sh", "-c", fmt.Sprintf("find %s -type f |xargs cat", destDir)).CombinedOutput()
	c.Assert(err, IsNil)
	c.Assert(string(c1), Equals, string(c2))
}

func (ts *HTestSuite) TestSyncDirs(c *C) {

	for _, l := range [][2]string{
		[2]string{"src-short", "dst-loooooooooooong"},
		[2]string{"src-loooooooooooong", "dst-short"},
		[2]string{"src-eq", "dst-eq"},
	} {

		// ensure we have src, dest dirs with different length
		srcDir := filepath.Join(c.MkDir(), l[0])
		err := os.MkdirAll(srcDir, 0755)
		c.Assert(err, IsNil)
		destDir := filepath.Join(c.MkDir(), l[1])
		err = os.MkdirAll(destDir, 0755)
		c.Assert(err, IsNil)

		// add a src subdir
		subdir := filepath.Join(srcDir, "subdir")
		err = os.Mkdir(subdir, 0755)
		c.Assert(err, IsNil)
		makeTestFiles(c, subdir, destDir)

		// add a dst subdir that needs to get deleted
		subdir2 := filepath.Join(destDir, "to-be-deleted-subdir")
		err = os.Mkdir(subdir2, 0755)
		subdir3 := filepath.Join(subdir2, "to-be-deleted-sub-subdir")
		err = os.Mkdir(subdir3, 0755)

		// and a toplevel
		makeTestFiles(c, srcDir, destDir)

		// do it
		err = RSyncWithDelete(srcDir, destDir)
		c.Assert(err, IsNil)

		// ensure meta-data is identical
		compareDirs(c, srcDir, destDir)
		compareDirs(c, filepath.Join(srcDir, "subdir"), filepath.Join(destDir, "subdir"))
	}
}

func (ts *HTestSuite) TestSyncDirFails(c *C) {
	srcDir := c.MkDir()
	err := os.MkdirAll(srcDir, 0755)
	c.Assert(err, IsNil)

	destDir := c.MkDir()
	err = os.MkdirAll(destDir, 0755)
	c.Assert(err, IsNil)

	err = ioutil.WriteFile(filepath.Join(destDir, "meep"), []byte(nil), 0644)
	c.Assert(err, IsNil)

	// ensure remove fails
	err = os.Chmod(destDir, 0100)
	c.Assert(err, IsNil)
	// make tempdir cleanup work again
	defer os.Chmod(destDir, 0755)

	// do it
	err = RSyncWithDelete(srcDir, destDir)
	c.Check(err, NotNil)
	c.Check(err, ErrorMatches, ".*permission denied.*")
}

func (ts *HTestSuite) TestCopyIfDifferent(c *C) {
	srcDir := c.MkDir()
	dstDir := c.MkDir()

	// new file
	src := filepath.Join(srcDir, "bop")
	dst := filepath.Join(dstDir, "bob")
	err := ioutil.WriteFile(src, []byte(nil), 0644)
	c.Assert(err, IsNil)

	err = CopyIfDifferent(src, dst)
	c.Assert(err, IsNil)
	c.Check(FilesAreEqual(dst, src), Equals, true)

	// updated file
	src = filepath.Join(srcDir, "bip")
	dst = filepath.Join(dstDir, "bib")
	err = ioutil.WriteFile(src, []byte("123"), 0644)
	c.Assert(err, IsNil)
	err = ioutil.WriteFile(dst, []byte(nil), 0644)
	c.Assert(err, IsNil)

	err = CopyIfDifferent(src, dst)
	c.Assert(err, IsNil)
	c.Check(FilesAreEqual(dst, src), Equals, true)
}

func (ts *HTestSuite) TestCopyIfDifferentErrorsOnNoSrc(c *C) {
	srcDir := c.MkDir()
	dstDir := c.MkDir()

	src := filepath.Join(srcDir, "mop")
	dst := filepath.Join(dstDir, "mop")

	err := CopyIfDifferent(src, dst)
	c.Assert(err, NotNil)
}
