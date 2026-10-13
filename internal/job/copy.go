package job

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Limits on what one job may copy into its working folder.
const (
	maxCopyFiles = 5000
	maxCopyBytes = 100 << 20
)

// within reports whether path (already absolute and symlink-resolved) is root
// or inside it.
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// resolveRead turns a requested source folder into a real path inside one of
// the allowed roots, or refuses. Symlinks are resolved first, so a link inside
// an allowed folder cannot lead out of it.
func resolveRead(from string, roots []string) (string, error) {
	if !filepath.IsAbs(from) {
		return "", badRequestf("ref %q: the source must be an absolute path", from)
	}
	real, err := filepath.EvalSymlinks(filepath.Clean(from))
	if err != nil {
		return "", badRequestf("ref %q: not found", from)
	}
	for _, root := range roots {
		realRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		if within(realRoot, real) {
			return real, nil
		}
	}
	return "", badRequestf("ref %q is not in a place this job may read", from)
}

// destination turns a requested target into a path inside work, or refuses.
// Nothing may be written under .claude (the skills) or outside the folder.
func destination(work, to string) (string, error) {
	clean := filepath.Clean(to)
	if to == "" || filepath.IsAbs(clean) || clean == "." || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", badRequestf("ref target %q must be a relative path inside the working folder", to)
	}
	if clean == ".claude" || strings.HasPrefix(clean, ".claude"+string(filepath.Separator)) {
		return "", badRequestf("ref target %q may not be under .claude", to)
	}
	return filepath.Join(work, clean), nil
}

// copier copies folders while counting what it has copied, so the limits hold
// across everything one job brings in.
type copier struct {
	files int
	bytes int64
}

// copyTree copies the folder at src (a real path) to dst. Only regular files
// and folders are copied: a symlink or a device found inside is refused, not
// followed.
func (c *copier) copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return badRequestf("%s is not a folder", src)
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type().IsRegular():
			c.files++
			if c.files > maxCopyFiles {
				return badRequestf("more than %d files to copy", maxCopyFiles)
			}
			return c.copyFile(path, target)
		default:
			return badRequestf("%s is not a plain file or folder", path)
		}
	})
}

func (c *copier) copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(in, maxCopyBytes-c.bytes+1))
	c.bytes += n
	if err == nil && c.bytes > maxCopyBytes {
		err = badRequestf("more than %d MB to copy", maxCopyBytes>>20)
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}

// BadRequest is a refusal of the request itself (an unknown profile, a path
// outside what the profile allows), as opposed to a job that failed.
type BadRequest struct{ msg string }

func (e *BadRequest) Error() string { return e.msg }

func badRequestf(format string, args ...any) error {
	return &BadRequest{msg: fmt.Sprintf(format, args...)}
}

// IsBadRequest reports whether err is a refusal of the request.
func IsBadRequest(err error) bool {
	var b *BadRequest
	return errors.As(err, &b)
}
