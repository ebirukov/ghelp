package linux

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var ErrMountExist = errors.New("mount point already exists")
var ErrMountNotExist = errors.New("mount point not exists")

type FS struct {
	source, target string
	fsType         string
	// Check if mount point exists
	check func(string) error
}

func (fs *FS) String() string {
	return fmt.Sprintf("FS[type=%s, target=%s]", fs.fsType, fs.target)
}

type FSError struct {
	op  string
	fs  *FS
	Err error
}

func (e *FSError) Error() string {
	return fmt.Sprintf("%s error; %s: %v\n", e.op, e.fs, e.Err)
}

func (e *FSError) Unwrap() error {
	return e.Err
}

func NewFS(source, target, fsType string, check func(string) error) *FS {
	return &FS{
		target: target,
		fsType: fsType,
		source: source,
		check:  check,
	}
}

func (fs *FS) Mount() error {
	return fs.MountWithFlags(0)
}

func (fs *FS) Unmount(flags int) error {
	return syscall.Unmount(fs.target, flags)
}

func (fs *FS) MountWithFlags(flags uintptr) error {
	if fs.check != nil {
		err := fs.check(fs.target)
		if err != nil {
			return &FSError{
				op:  "check mount point",
				fs:  fs,
				Err: err,
			}
		}
	}

	if err := os.MkdirAll(fs.target, 0755); err != nil {
		return &FSError{op: "mount", fs: fs, Err: err}
	}

	if err := syscall.Mount(fs.source, fs.target, fs.fsType, flags, ""); err != nil {
		return &FSError{op: "mount", fs: fs, Err: err}
	}

	return nil
}

func (fs *FS) ReadFile(path string) ([]byte, error) {
	if fs.check != nil {
		err := fs.check(fs.target)
		if err == nil || !errors.Is(err, ErrMountExist) {
			return nil, &FSError{op: "read file", fs: fs, Err: ErrMountNotExist}
		}
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(fs.target, path)
	}

	if !strings.HasPrefix(path, fs.target) {
		return nil, &FSError{op: "read file", fs: fs, Err: fmt.Errorf("path %s has another mount point", path)}
	}

	return os.ReadFile(path)
}

func (fs *FS) ReadLink(path string) (string, error) {
	if fs.check != nil {
		err := fs.check(fs.target)
		if err == nil || !errors.Is(err, ErrMountExist) {
			return "", &FSError{op: "read link", fs: fs, Err: ErrMountNotExist}
		}
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(fs.target, path)
	}

	if !strings.HasPrefix(path, fs.target) {
		return "", &FSError{op: "read link", fs: fs, Err: fmt.Errorf("path %s has another mount point", path)}
	}

	return os.Readlink(path)
}

func (fs *FS) WriteFile(path string, content string, perm os.FileMode) error {
	if fs.check != nil {
		err := fs.check(fs.target)
		if err == nil || !errors.Is(err, ErrMountExist) {
			return &FSError{op: "write file", fs: fs, Err: ErrMountNotExist}
		}
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(fs.target, path)
	}

	if !strings.HasPrefix(path, fs.target) {
		return &FSError{op: "write file", fs: fs, Err: fmt.Errorf("path %s has another mount point", path)}
	}

	return os.WriteFile(path, unsafe.Slice(unsafe.StringData(content), len(content)), perm)
}

func checkExist(path string) func(target string) error {
	return func(target string) error {
		if _, err := os.Stat(filepath.Join(target, path)); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		return ErrMountExist
	}
}

var ProcFS = NewFS("proc", "/proc", "proc", checkExist("self"))

var TraceFS = NewFS("tracefs", "/sys/kernel/tracing", "tracefs", checkExist("available_events"))

var DebugFS = NewFS("debugfs", "/sys/kernel/debug", "debugfs", checkExist("tracing"))

var CgroupFS = NewFS("cgroup2", "/sys/fs/cgroup", "cgroup2", checkExist("cgroup.controllers"))

var TempFS = NewFS("tmpfs", "/tmp", "tmpfs", nil)

var DevFS = NewFS("udev", "/dev", "devtmpfs", nil)
