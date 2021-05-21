// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall_test

import (
	"fmt"
	"internal/testenv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestWin32finddata(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "long_name.and_extension")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create %v: %v", path, err)
	}
	f.Close()

	type X struct {
		fd  syscall.Win32finddata
		got byte
		pad [10]byte // to protect ourselves

	}
	var want byte = 2 // it is unlikely to have this character in the filename
	x := X{got: want}

	pathp, _ := syscall.UTF16PtrFromString(path)
	h, err := syscall.FindFirstFile(pathp, &(x.fd))
	if err != nil {
		t.Fatalf("FindFirstFile failed: %v", err)
	}
	err = syscall.FindClose(h)
	if err != nil {
		t.Fatalf("FindClose failed: %v", err)
	}

	if x.got != want {
		t.Fatalf("memory corruption: want=%d got=%d", want, x.got)
	}
}

func abort(funcname string, err error) {
	panic(funcname + " failed: " + err.Error())
}

func ExampleLoadLibrary() {
	h, err := syscall.LoadLibrary("kernel32.dll")
	if err != nil {
		abort("LoadLibrary", err)
	}
	defer syscall.FreeLibrary(h)
	proc, err := syscall.GetProcAddress(h, "GetVersion")
	if err != nil {
		abort("GetProcAddress", err)
	}
	r, _, _ := syscall.Syscall(uintptr(proc), 0, 0, 0, 0)
	major := byte(r)
	minor := uint8(r >> 8)
	build := uint16(r >> 16)
	print("windows version ", major, ".", minor, " (Build ", build, ")\n")
}

func TestTOKEN_ALL_ACCESS(t *testing.T) {
	if syscall.TOKEN_ALL_ACCESS != 0xF01FF {
		t.Errorf("TOKEN_ALL_ACCESS = %x, want 0xF01FF", syscall.TOKEN_ALL_ACCESS)
	}
}

func TestStdioAreInheritable(t *testing.T) {
	testenv.MustHaveGoBuild(t)
	testenv.MustHaveExecPath(t, "gcc")

	tmpdir := t.TempDir()

	// build go dll
	const dlltext = `
package main

import "C"
import (
	"fmt"
)

//export HelloWorld
func HelloWorld() {
	fmt.Println("Hello World")
}

func main() {}
`
	dllsrc := filepath.Join(tmpdir, "helloworld.go")
	err := os.WriteFile(dllsrc, []byte(dlltext), 0644)
	if err != nil {
		t.Fatal(err)
	}
	dll := filepath.Join(tmpdir, "helloworld.dll")
	cmd := exec.Command(testenv.GoToolPath(t), "build", "-o", dll, "-buildmode", "c-shared", dllsrc)
	out, err := testenv.CleanCmdEnv(cmd).CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build go library: %s\n%s", err, out)
	}

	// run powershell script
	psscript := fmt.Sprintf(`
hostname;
$signature = " [DllImport("%q")] public static extern void HelloWorld(); ";
Add-Type -MemberDefinition $signature -Name World -Namespace Hello;
[Hello.World]::HelloWorld();
hostname;
`, dll)
	psscript = strings.ReplaceAll(psscript, "\n", "")
	out, err = exec.Command("powershell", "-Command", psscript).CombinedOutput()
	if err != nil {
		t.Fatalf("Powershell command failed: %v: %v", err, string(out))
	}

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	have := strings.ReplaceAll(string(out), "\n", "")
	have = strings.ReplaceAll(have, "\r", "")
	want := fmt.Sprintf("%sHello World%s", hostname, hostname)
	if have != want {
		t.Fatalf("Powershell command output is wrong: got %q, want %q", have, want)
	}
}