/*
 * Copyright (C) 2017 Canonical Ltd
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

#include "snap.h"
#include "snap.c"

#include <glib.h>

static void test_verify_security_tag(void)
{
	// First, test the names we know are good
	g_assert_true(verify_security_tag("snap.name.app", "name"));
	g_assert_true(verify_security_tag
		      ("snap.network-manager.NetworkManager",
		       "network-manager"));
	g_assert_true(verify_security_tag("snap.f00.bar-baz1", "f00"));
	g_assert_true(verify_security_tag("snap.foo.hook.bar", "foo"));
	g_assert_true(verify_security_tag("snap.foo.hook.bar-baz", "foo"));

	// Now, test the names we know are bad
	g_assert_false(verify_security_tag
		       ("pkg-foo.bar.0binary-bar+baz", "bar"));
	g_assert_false(verify_security_tag("pkg-foo_bar_1.1", ""));
	g_assert_false(verify_security_tag("appname/..", ""));
	g_assert_false(verify_security_tag("snap", ""));
	g_assert_false(verify_security_tag("snap.", ""));
	g_assert_false(verify_security_tag("snap.name", "name"));
	g_assert_false(verify_security_tag("snap.name.", "name"));
	g_assert_false(verify_security_tag("snap.name.app.", "name"));
	g_assert_false(verify_security_tag("snap.name.hook.", "name"));
	g_assert_false(verify_security_tag("snap!name.app", "!name"));
	g_assert_false(verify_security_tag("snap.-name.app", "-name"));
	g_assert_false(verify_security_tag("snap.name!app", "name!"));
	g_assert_false(verify_security_tag("snap.name.-app", "name"));
	g_assert_false(verify_security_tag("snap.name.app!hook.foo", "name"));
	g_assert_false(verify_security_tag("snap.name.app.hook!foo", "name"));
	g_assert_false(verify_security_tag("snap.name.app.hook.-foo", "name"));
	g_assert_false(verify_security_tag("snap.name.app.hook.f00", "name"));
	g_assert_false(verify_security_tag("sna.pname.app", "pname"));
	g_assert_false(verify_security_tag("snap.n@me.app", "n@me"));
	g_assert_false(verify_security_tag("SNAP.name.app", "name"));
	g_assert_false(verify_security_tag("snap.Name.app", "Name"));
	// This used to be false but it's now allowed.
	g_assert_true(verify_security_tag("snap.0name.app", "0name"));
	g_assert_false(verify_security_tag("snap.-name.app", "-name"));
	g_assert_false(verify_security_tag("snap.name.@app", "name"));
	g_assert_false(verify_security_tag(".name.app", "name"));
	g_assert_false(verify_security_tag("snap..name.app", ".name"));
	g_assert_false(verify_security_tag("snap.name..app", "name."));
	g_assert_false(verify_security_tag("snap.name.app..", "name"));

	// Test names that are both good, but snap name doesn't match security tag
	g_assert_false(verify_security_tag("snap.foo.hook.bar", "fo"));
	g_assert_false(verify_security_tag("snap.foo.hook.bar", "fooo"));
	g_assert_false(verify_security_tag("snap.foo.hook.bar", "snap"));
	g_assert_false(verify_security_tag("snap.foo.hook.bar", "bar"));

	// Regression test 12to8
	g_assert_true(verify_security_tag("snap.12to8.128to8", "12to8"));
	g_assert_true(verify_security_tag("snap.123test.123test", "123test"));
	g_assert_true(verify_security_tag
		      ("snap.123test.hook.configure", "123test"));

}

static void test_sc_snap_name_validate(void)
{
	struct sc_error *err = NULL;

	// Smoke test, a valid snap name
	sc_snap_name_validate("hello-world", &err);
	g_assert_null(err);

	// Smoke test: invalid character 
	sc_snap_name_validate("hello world", &err);
	g_assert_nonnull(err);
	g_assert_true(sc_error_match
		      (err, SC_SNAP_DOMAIN, SC_SNAP_INVALID_NAME));
	g_assert_cmpstr(sc_error_msg(err), ==,
			"snap name must use lower case letters, digits or dashes");
	sc_error_free(err);

	// Smoke test: no letters
	sc_snap_name_validate("", &err);
	g_assert_nonnull(err);
	g_assert_true(sc_error_match
		      (err, SC_SNAP_DOMAIN, SC_SNAP_INVALID_NAME));
	g_assert_cmpstr(sc_error_msg(err), ==,
			"snap name must contain at least one letter");
	sc_error_free(err);

	// Smoke test: leading dash
	sc_snap_name_validate("-foo", &err);
	g_assert_nonnull(err);
	g_assert_true(sc_error_match
		      (err, SC_SNAP_DOMAIN, SC_SNAP_INVALID_NAME));
	g_assert_cmpstr(sc_error_msg(err), ==,
			"snap name cannot start with a dash");
	sc_error_free(err);

	// Smoke test: trailing dash
	sc_snap_name_validate("foo-", &err);
	g_assert_nonnull(err);
	g_assert_true(sc_error_match
		      (err, SC_SNAP_DOMAIN, SC_SNAP_INVALID_NAME));
	g_assert_cmpstr(sc_error_msg(err), ==,
			"snap name cannot end with a dash");
	sc_error_free(err);

	// Smoke test: double dash
	sc_snap_name_validate("f--oo", &err);
	g_assert_nonnull(err);
	g_assert_true(sc_error_match
		      (err, SC_SNAP_DOMAIN, SC_SNAP_INVALID_NAME));
	g_assert_cmpstr(sc_error_msg(err), ==,
			"snap name cannot contain two consecutive dashes");
	sc_error_free(err);

	// Smoke test: NULL name is not valid
	sc_snap_name_validate(NULL, &err);
	g_assert_nonnull(err);
	g_assert_true(sc_error_match
		      (err, SC_SNAP_DOMAIN, SC_SNAP_INVALID_NAME));
	g_assert_cmpstr(sc_error_msg(err), ==, "snap name cannot be NULL");
	sc_error_free(err);

	const char *valid_names[] = {
		"a", "aa", "aaa", "aaaa",
		"a-a", "aa-a", "a-aa", "a-b-c",
		"a0", "a-0", "a-0a",
		"01game", "1-or-2"
	};
	for (size_t i = 0; i < sizeof valid_names / sizeof *valid_names; ++i) {
		g_test_message("checking valid snap name: %s", valid_names[i]);
		sc_snap_name_validate(valid_names[i], &err);
		g_assert_null(err);
	}
	const char *invalid_names[] = {
		// name cannot be empty
		"",
		// names cannot be too long
		"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		"xxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxx",
		"1111111111111111111111111111111111111111x",
		"x1111111111111111111111111111111111111111",
		"x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x-x",
		// dashes alone are not a name
		"-", "--",
		// double dashes in a name are not allowed
		"a--a",
		// name should not end with a dash
		"a-",
		// name cannot have any spaces in it
		"a ", " a", "a a",
		// a number alone is not a name
		"0", "123", "1-2-3",
		// identifier must be plain ASCII
		"日本語", "한글", "ру́сский язы́к",
	};
	for (size_t i = 0; i < sizeof invalid_names / sizeof *invalid_names;
	     ++i) {
		g_test_message("checking invalid snap name: >%s<",
			       invalid_names[i]);
		sc_snap_name_validate(invalid_names[i], &err);
		g_assert_nonnull(err);
		g_assert_true(sc_error_match
			      (err, SC_SNAP_DOMAIN, SC_SNAP_INVALID_NAME));
		sc_error_free(err);
	}
	// Regression test: 12to8 and 123test
	sc_snap_name_validate("12to8", &err);
	g_assert_null(err);
	sc_snap_name_validate("123test", &err);
	g_assert_null(err);

	// In case we switch to a regex, here's a test that could break things.
	const char good_bad_name[] = "u-94903713687486543234157734673284536758";
	char varname[sizeof good_bad_name] = { 0 };
	for (size_t i = 3; i <= sizeof varname - 1; i++) {
		g_assert_nonnull(memcpy(varname, good_bad_name, i));
		varname[i] = 0;
		g_test_message("checking valid snap name: >%s<", varname);
		sc_snap_name_validate(varname, &err);
		g_assert_null(err);
		sc_error_free(err);
	}
}

static void test_sc_snap_name_validate__respects_error_protocol(void)
{
	if (g_test_subprocess()) {
		sc_snap_name_validate("hello world", NULL);
		g_test_message("expected sc_snap_name_validate to return");
		g_test_fail();
		return;
	}
	g_test_trap_subprocess(NULL, 0, 0);
	g_test_trap_assert_failed();
	g_test_trap_assert_stderr
	    ("snap name must use lower case letters, digits or dashes\n");
}

static void test_sc_snap_drop_instance_name_no_dest(void)
{
	if (g_test_subprocess()) {
		sc_snap_drop_instance_name("foo_bar", NULL, 0);
		g_test_fail();
		return;
	}
	g_test_trap_subprocess(NULL, 0, 0);
	g_test_trap_assert_failed();

}

static void test_sc_snap_drop_instance_name_short_dest(void)
{
	if (g_test_subprocess()) {
		char dest[10] = { 0 };
		sc_snap_drop_instance_name("foo-foo-foo-foo-foo_bar", dest,
					   sizeof dest);
		g_test_fail();
		return;
	}
	g_test_trap_subprocess(NULL, 0, 0);
	g_test_trap_assert_failed();
}

static void test_sc_snap_drop_instance_name_short_dest2(void)
{
	if (g_test_subprocess()) {
		char dest[3] = { 0 };	// "foo" sans the nil byte
		sc_snap_drop_instance_name("foo", dest, sizeof dest);
		g_test_fail();
		return;
	}
	g_test_trap_subprocess(NULL, 0, 0);
	g_test_trap_assert_failed();
}

static void test_sc_snap_drop_instance_name_no_name(void)
{
	if (g_test_subprocess()) {
		char dest[10] = { 0 };
		sc_snap_drop_instance_name(NULL, dest, sizeof dest);
		g_test_fail();
		return;
	}
	g_test_trap_subprocess(NULL, 0, 0);
	g_test_trap_assert_failed();
}

static void test_sc_snap_drop_instance_name_basic(void)
{
	char name[41] = { 0xff };

	sc_snap_drop_instance_name("foo_bar", name, sizeof name);
	g_assert_cmpstr(name, ==, "foo");

	memset(name, 0xff, sizeof name);
	sc_snap_drop_instance_name("foo-bar_bar", name, sizeof name);
	g_assert_cmpstr(name, ==, "foo-bar");

	memset(name, 0xff, sizeof name);
	sc_snap_drop_instance_name("foo-bar", name, sizeof name);
	g_assert_cmpstr(name, ==, "foo-bar");

	memset(name, 0xff, sizeof name);
	sc_snap_drop_instance_name("_baz", name, sizeof name);
	g_assert_cmpstr(name, ==, "");

	memset(name, 0xff, sizeof name);
	sc_snap_drop_instance_name("foo", name, sizeof name);
	g_assert_cmpstr(name, ==, "foo");
}

static void __attribute__ ((constructor)) init(void)
{
	g_test_add_func("/snap/verify_security_tag", test_verify_security_tag);
	g_test_add_func("/snap/sc_snap_name_validate",
			test_sc_snap_name_validate);
	g_test_add_func("/snap/sc_snap_name_validate/respects_error_protocol",
			test_sc_snap_name_validate__respects_error_protocol);
	g_test_add_func("/snap/sc_snap_drop_instance_name/basic",
			test_sc_snap_drop_instance_name_basic);
	g_test_add_func("/snap/sc_snap_drop_instance_name/no_dest",
			test_sc_snap_drop_instance_name_no_dest);
	g_test_add_func("/snap/sc_snap_drop_instance_name/no_name",
			test_sc_snap_drop_instance_name_no_name);
	g_test_add_func("/snap/sc_snap_drop_instance_name/short_dest",
			test_sc_snap_drop_instance_name_short_dest);
	g_test_add_func("/snap/sc_snap_drop_instance_name/short_dest2",
			test_sc_snap_drop_instance_name_short_dest2);
}
