package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/your-github-username/personal-cloud/cli/internal/config"
	"github.com/your-github-username/personal-cloud/cli/internal/git"
	"github.com/your-github-username/personal-cloud/cli/internal/manifest"
	"github.com/your-github-username/personal-cloud/cli/internal/ship"
	"github.com/your-github-username/personal-cloud/cli/internal/woodpecker"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "pc: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `pc — deploy apps to personal-cloud via Woodpecker

Usage:
  pc init
  pc validate
  pc ship [--local] [--public|--private] [--tag TAG] [--branch BR] [--allow-dirty] [--wait]
  pc status
  pc logs

Config: ~/.config/pc/config.yaml
Manifest: .personal-cloud.yaml (walks up from cwd)

`)
}

func run(cmd string, args []string) error {
	switch cmd {
	case "init":
		return cmdInit(args)
	case "validate":
		return cmdValidate(args)
	case "ship":
		return cmdShip(args)
	case "status":
		return cmdStatus(args)
	case "logs":
		return cmdLogs(args)
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func loadContext() (string, *manifest.Manifest, *config.Config, *git.Info, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, nil, nil, err
	}
	repoRoot, m, err := manifest.Find(cwd)
	if err != nil {
		return "", nil, nil, nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return "", nil, nil, nil, err
	}
	gi, err := git.Discover(repoRoot)
	if err != nil {
		return "", nil, nil, nil, err
	}
	return repoRoot, m, cfg, gi, nil
}

func cmdInit(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Println("pc init — create .personal-cloud.yaml in the current directory")
		return nil
	}
	target := ".personal-cloud.yaml"
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists", target)
	}

	reader := bufio.NewReader(os.Stdin)
	ask := func(prompt, def string) string {
		if def != "" {
			fmt.Printf("%s [%s]: ", prompt, def)
		} else {
			fmt.Printf("%s: ", prompt)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		return line
	}

	name := ask("App name (personal-cloud/apps/<name>)", "")
	image := ask("Image (ghcr.io/owner/name)", "")
	ctx := ask("Build context", ".")
	df := ask("Dockerfile", "Dockerfile")
	container := ask("Container name", name)
	port := ask("Service port", "8080")
	exposure := ask("Exposure (public|private)", "private")
	host := ""
	if strings.ToLower(exposure) == "public" {
		host = ask("Public hostname", "")
	}
	tmpl := ask("Compose template (default|with-postgres)", "default")

	content := fmt.Sprintf(`name: %s
image: %s

build:
  context: %s
  dockerfile: %s

service:
  container: %s
  port: %s
  health_path: /health

route:
  exposure: %s
`, name, image, ctx, df, container, port, exposure)
	if host != "" {
		content += fmt.Sprintf("  host: %s\n", host)
	}
	content += fmt.Sprintf(`
compose:
  template: %s
`, tmpl)

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Created %s\n", target)
	return nil
}

func cmdValidate(args []string) error {
	repoRoot, m, cfg, gi, err := loadContext()
	if err != nil {
		return err
	}
	if err := m.Validate(repoRoot); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	client := woodpecker.New(woodpecker.NormalizeURL(cfg.Woodpecker.URL), cfg.Woodpecker.Token)
	if err := client.Ping(); err != nil {
		return fmt.Errorf("woodpecker: %w", err)
	}
	fmt.Printf("OK  manifest=%s repo=%s ref=%s\n", m.Name, gi.Remote, gi.ShortSHA)
	return nil
}

func cmdShip(args []string) error {
	var opt ship.Options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--local":
			opt.Local = true
		case "--public":
			opt.Public = true
		case "--private":
			opt.Private = true
		case "--allow-dirty":
			opt.AllowDirty = true
		case "--wait":
			opt.Wait = true
		case "--tag":
			i++
			if i >= len(args) {
				return fmt.Errorf("--tag requires a value")
			}
			opt.Tag = args[i]
		case "--branch":
			i++
			if i >= len(args) {
				return fmt.Errorf("--branch requires a value")
			}
			opt.Branch = args[i]
		default:
			return fmt.Errorf("unknown flag %s", args[i])
		}
	}

	repoRoot, m, cfg, gi, err := loadContext()
	if err != nil {
		return err
	}
	if err := m.Validate(repoRoot); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	if opt.Public && opt.Private {
		return fmt.Errorf("cannot use --public and --private together")
	}

	res, err := ship.Run(cfg, repoRoot, m, gi, opt)
	if err != nil {
		return err
	}
	fmt.Printf("Pipeline #%d started (image tag %s)\n%s\n", res.Number, res.ImageTag, res.PipelineURL)
	return nil
}

func cmdStatus(args []string) error {
	_, m, cfg, _, err := loadContext()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	client := woodpecker.New(woodpecker.NormalizeURL(cfg.Woodpecker.URL), cfg.Woodpecker.Token)
	cachePath, _ := config.CachePath()
	repoID, err := client.RepoID(cfg.PersonalCloud.Owner, cfg.PersonalCloud.Repo, cachePath)
	if err != nil {
		return err
	}
	pl, err := client.LatestPipelineForRepo(repoID)
	if err != nil {
		return err
	}
	fmt.Printf("personal-cloud latest pipeline #%d: %s\n", pl.Number, pl.State)
	if pl.Error != "" {
		fmt.Printf("error: %s\n", pl.Error)
	}
	fmt.Println(client.PipelineURL(repoID, pl.Number))
	_ = m
	return nil
}

func cmdLogs(args []string) error {
	_, _, cfg, _, err := loadContext()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	client := woodpecker.New(woodpecker.NormalizeURL(cfg.Woodpecker.URL), cfg.Woodpecker.Token)
	cachePath, _ := config.CachePath()
	repoID, err := client.RepoID(cfg.PersonalCloud.Owner, cfg.PersonalCloud.Repo, cachePath)
	if err != nil {
		return err
	}
	pl, err := client.LatestPipelineForRepo(repoID)
	if err != nil {
		return err
	}
	fmt.Println(client.PipelineURL(repoID, pl.Number))
	return nil
}
