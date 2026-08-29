package ship

import (
	"fmt"
	"strings"
)

// ParseArgs reads ship flags and an optional GitHub repo argument
// (owner/repo, repo name, or github.com URL).
func ParseArgs(args []string) (Options, error) {
	var opt Options
	take := func(i int, flag string) (string, int, error) {
		if i+1 >= len(args) {
			return "", i, fmt.Errorf("%s requires a value", flag)
		}
		return args[i+1], i + 1, nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			return opt, ErrHelp
		case a == "--local":
			opt.Local = true
		case a == "--public":
			opt.Public = true
		case a == "--private":
			opt.Private = true
		case a == "--allow-dirty":
			opt.AllowDirty = true
		case a == "--wait":
			opt.Wait = true
		case a == "--tag":
			v, ni, err := take(i, a)
			if err != nil {
				return opt, err
			}
			opt.Tag, i = v, ni
		case strings.HasPrefix(a, "--tag="):
			opt.Tag = strings.TrimPrefix(a, "--tag=")
		case a == "--branch":
			v, ni, err := take(i, a)
			if err != nil {
				return opt, err
			}
			opt.Branch, i = v, ni
		case strings.HasPrefix(a, "--branch="):
			opt.Branch = strings.TrimPrefix(a, "--branch=")
		case a == "--ref":
			v, ni, err := take(i, a)
			if err != nil {
				return opt, err
			}
			opt.Ref, i = v, ni
		case strings.HasPrefix(a, "--ref="):
			opt.Ref = strings.TrimPrefix(a, "--ref=")
		case a == "--repo":
			v, ni, err := take(i, a)
			if err != nil {
				return opt, err
			}
			opt.Repo, i = v, ni
		case strings.HasPrefix(a, "--repo="):
			opt.Repo = strings.TrimPrefix(a, "--repo=")
		case strings.HasPrefix(a, "-"):
			return opt, fmt.Errorf("unknown flag %s", a)
		default:
			if opt.Repo != "" {
				return opt, fmt.Errorf("unexpected argument %q", a)
			}
			opt.Repo = a
		}
	}
	return opt, nil
}

// ErrHelp is returned when -h/--help is passed.
var ErrHelp = fmt.Errorf("help")

// IsHelp reports whether ParseArgs saw -h/--help.
func IsHelp(err error) bool {
	return err == ErrHelp
}
