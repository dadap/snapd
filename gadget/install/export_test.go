// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2020 Canonical Ltd
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

package install

func MockDeployMountpoint(new string) (restore func()) {
	old := deployMountpoint
	deployMountpoint = new
	return func() {
		deployMountpoint = old
	}
}

func MockSysMount(f func(source, target, fstype string, flags uintptr, data string) error) (restore func()) {
	old := sysMount
	sysMount = f
	return func() {
		sysMount = old
	}
}

func MockSysUnmount(f func(target string, flags int) error) (restore func()) {
	old := sysUnmount
	sysUnmount = f
	return func() {
		sysUnmount = old
	}
}
