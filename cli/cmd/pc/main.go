package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/your-org/personal-cloud/cli/internal/config"
	"github.com/your-org/personal-cloud/cli/internal/git"
	ghapi "github.com/your-org/personal-cloud/cli/internal/github"
	"github.com/your-org/personal-cloud/cli/internal/manifest"
	"github.com/your-org/personal-cloud/cli/internal/ship"
	"github.com/your-org/personal-cloud/cli/internal/ui"
	"github.com/your-org/personal-cloud/cli/internal/woodpecker"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		ui.ErrorBlock(err.Error())
		os.Exit(1)
	}
}

func usage() {
	ui.Logo("deploy any repo to your personal cloud via Woodpecker")

	type cmd struct{ name, args, desc string }
	cmds := []cmd{
		{"init", "", "scaffold a .personal-cloud.yaml manifest"},
		{"validate", "[repo]", "check the manifest and Woodpecker connectivity"},
		{"ship", "[repo] [flags]", "build, deploy and route an app"},
		{"env", "[init|push]", "manage app secrets locally and on the VM"},
		{"status", "", "show the latest pipeline and its steps"},
		{"logs", "", "print the latest pipeline URL"},
	}
	ui.Heading("Commands")
	w := 0
	for _, c := range cmds {
		if l := len(c.name + " " + c.args); l > w {
			w = l
		}
	}
	for _, c := range cmds {
		label := c.name
		if c.args != "" {
			label += " " + ui.Dim(c.args)
		}
		pad := strings.Repeat(" ", w-len(c.name+" "+c.args))
		fmt.Fprintf(ui.Out, "  %s %s%s  %s\n", ui.Brand("pc"), ui.Bold(label), pad, ui.Dim(c.desc))
	}

	ui.Heading("Ship flags")
	flags := [][2]string{
		{"--local", "build & push from this machine, CI deploys only"},
		{"--public / --private", "force public ACME or Tailscale-only route"},
		{"--tag TAG", "image tag (default: dev-<git-sha>)"},
		{"--ref REF", "app git ref when shipping from GitHub (default: default branch)"},
		{"--repo SLUG", "GitHub owner/repo (or pass as the first argument)"},
		{"--branch BR", "personal-cloud Woodpecker branch (default: current, or main from GitHub)"},
		{"--allow-dirty", "ship even with uncommitted changes"},
		{"--wait", "stream pipeline progress until it finishes"},
	}
	fw := 0
	for _, f := range flags {
		if len(f[0]) > fw {
			fw = len(f[0])
		}
	}
	for _, f := range flags {
		pad := strings.Repeat(" ", fw-len(f[0]))
		fmt.Fprintf(ui.Out, "  %s %s%s  %s\n", ui.Gray("·"), ui.Cyan(f[0]), pad, ui.Dim(f[1]))
	}

	ui.Heading("Paths")
	ui.NewDetails().
		Add("config", "~/.config/pc/config.yaml").
		Add("manifest", ".personal-cloud.yaml (cwd, or fetched from GitHub)").
		Add("env secrets", "~/.config/pc/env/<app>.env").
		Render()
	fmt.Fprintln(ui.Out)
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
	case "env":
		return cmdEnv(args)
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q (run `pc help`)", cmd)
	}
}

type appCtx struct {
	repoRoot   string
	fromGitHub bool
	source     string
	m          *manifest.Manifest
	cfg        *config.Config
	gi         *git.Info
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadLocalContext() (*appCtx, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	repoRoot, m, err := manifest.Find(cwd)
	if err != nil {
		return nil, fmt.Errorf("%w\npass an app or owner/repo to use GitHub instead of a local clone (e.g. pc ship music-serve)", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	gi, err := git.Discover(repoRoot)
	if err != nil {
		return nil, err
	}
	return &appCtx{repoRoot: repoRoot, source: "local " + gi.Remote, m: m, cfg: cfg, gi: gi}, nil
}

func loadGitHubContext(repo, ref string) (*appCtx, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	slug, err := ghapi.NormalizeRepo(repo, cfg.GitHub.Owner)
	if err != nil {
		return nil, err
	}
	client := ghapi.New(ghapi.AccessToken(cfg), "")
	res, err := client.Resolve(slug, ref)
	if err != nil {
		return nil, err
	}
	if err := res.Manifest.Validate(""); err != nil {
		return nil, err
	}
	src := "github.com/" + slug
	if ref != "" {
		src += "@" + ref
	}
	return &appCtx{fromGitHub: true, source: src, m: res.Manifest, cfg: cfg, gi: res.Git}, nil
}

func resolveApp(repo, ref string) (*appCtx, error) {
	if repo != "" {
		return loadGitHubContext(repo, ref)
	}
	if ref != "" {
		return nil, fmt.Errorf("--ref requires a GitHub repo argument (e.g. pc ship music-serve --ref main)")
	}
	return loadLocalContext()
}

func cmdInit(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(ui.Out, "pc init — create .personal-cloud.yaml in the current directory")
		return nil
	}
	target := ".personal-cloud.yaml"
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists", target)
	}

	ui.Logo("create a deployment manifest")
	ui.Info("Press enter to accept the %s default.", ui.Dim("[bracketed]"))
	ui.Heading("Manifest")

	reader := bufio.NewReader(os.Stdin)
	ask := func(prompt, def string) string {
		if def != "" {
			fmt.Fprintf(ui.Out, "  %s %s %s ", ui.Brand("?"), prompt, ui.Dim("["+def+"]"))
		} else {
			fmt.Fprintf(ui.Out, "  %s %s ", ui.Brand("?"), prompt)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return def
		}
		return line
	}

	name := ask("App name", "")
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
	tmpl := ask("Compose template (default|with-postgres|with-data-volume|with-data-volume-host|…)", "default")

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

	ui.Heading("Summary")
	d := ui.NewDetails().
		Add("name", ui.Bold(name)).
		Add("image", image).
		Add("container", container).
		Add("port", port).
		Add("exposure", exposure)
	if host != "" {
		d.Add("host", host)
	}
	d.Add("template", tmpl).Render()

	ui.Box("Created", []string{
		ui.Green("✔ ") + ui.Bold(target),
		ui.Dim("next: ") + "pc validate " + ui.Dim("→") + " pc ship --wait",
	})
	return nil
}

func parseRepoRef(args []string) (repo, ref string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			return "", "", ship.ErrHelp
		case a == "--ref":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--ref requires a value")
			}
			i++
			ref = args[i]
		case strings.HasPrefix(a, "--ref="):
			ref = strings.TrimPrefix(a, "--ref=")
		case a == "--repo":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--repo requires a value")
			}
			i++
			repo = args[i]
		case strings.HasPrefix(a, "--repo="):
			repo = strings.TrimPrefix(a, "--repo=")
		case strings.HasPrefix(a, "-"):
			return "", "", fmt.Errorf("unknown flag %s", a)
		default:
			if repo != "" {
				return "", "", fmt.Errorf("unexpected argument %q", a)
			}
			repo = a
		}
	}
	return repo, ref, nil
}

func cmdValidate(args []string) error {
	repo, ref, err := parseRepoRef(args)
	if err != nil {
		if ship.IsHelp(err) {
			fmt.Fprintln(ui.Out, "pc validate [repo] — check the manifest (local or GitHub) and Woodpecker")
			return nil
		}
		return err
	}

	ui.Logo("validate manifest & connectivity")

	ui.Heading("Checks")
	load := ui.NewSpinner("Loading manifest, config & git context…").Start()
	ac, err := resolveApp(repo, ref)
	if err != nil {
		load.Fail("Could not load context")
		return err
	}
	load.Success("Context loaded %s", ui.Dim("("+ac.source+")"))

	mv := ui.NewSpinner("Validating manifest…").Start()
	if err := ac.m.Validate(ac.repoRoot); err != nil {
		mv.Fail("Manifest invalid")
		return err
	}
	mv.Success("Manifest valid %s", ui.Dim("("+ac.m.Name+")"))

	cv := ui.NewSpinner("Validating config…").Start()
	if err := ac.cfg.Validate(); err != nil {
		cv.Fail("Config invalid")
		return err
	}
	cv.Success("Config valid")

	client := woodpecker.New(woodpecker.NormalizeURL(ac.cfg.Woodpecker.URL), ac.cfg.Woodpecker.Token)
	pv := ui.NewSpinner("Pinging Woodpecker…").Start()
	if err := client.Ping(); err != nil {
		pv.Fail("Woodpecker unreachable")
		return fmt.Errorf("woodpecker: %w", err)
	}
	pv.Success("Woodpecker reachable %s", ui.Dim("("+woodpecker.NormalizeURL(ac.cfg.Woodpecker.URL)+")"))

	exposure := strings.ToLower(ac.m.Route.Exposure)
	ui.Heading("Details")
	ui.NewDetails().
		Add("app", ui.Bold(ac.m.Name)).
		Add("image", ac.m.Image).
		Add("source", ac.source).
		Add("repo", ac.gi.Remote).
		Add("branch", ac.gi.Branch).
		Add("ref", ac.gi.ShortSHA).
		Add("exposure", exposure).
		Add("route host", ac.m.ResolvedHost(exposure, ac.cfg.Defaults.TailnetBase)).
		Add("woodpecker", woodpecker.NormalizeURL(ac.cfg.Woodpecker.URL)).
		Render()

	ui.Box("Ready to ship", []string{
		ui.Green("✔ ") + "everything checks out",
		ui.Dim("run: ") + "pc ship --wait",
	})
	return nil
}

func cmdShip(args []string) error {
	opt, err := ship.ParseArgs(args)
	if err != nil {
		if ship.IsHelp(err) {
			usage()
			return nil
		}
		return err
	}
	if opt.Public && opt.Private {
		return fmt.Errorf("cannot use --public and --private together")
	}

	ac, err := resolveApp(opt.Repo, opt.Ref)
	if err != nil {
		return err
	}
	if err := ac.m.Validate(ac.repoRoot); err != nil {
		return err
	}
	if err := ac.cfg.Validate(); err != nil {
		return err
	}
	if ac.fromGitHub {
		opt.Repo = ac.gi.Remote
	}

	exposure := strings.ToLower(ac.m.Route.Exposure)
	if opt.Public {
		exposure = "public"
	}
	if opt.Private {
		exposure = "private"
	}
	buildMode := "remote (on VM)"
	if opt.Local {
		buildMode = "local (this machine)"
	}

	ui.Logo("ship " + ac.m.Name)
	ui.Heading("Deploy plan")
	dirty := ui.Green("clean")
	if ac.gi.Dirty {
		dirty = ui.Yellow("dirty")
	}
	ui.NewDetails().
		Add("app", ui.Bold(ac.m.Name)).
		Add("image", ac.m.Image+":"+ui.Bold(resTag(opt, ac.gi))).
		Add("source", ac.source).
		Add("repo", ac.gi.Remote).
		Add("branch", branchOf(opt, ac.gi)).
		Add("ref", ac.gi.ShortSHA+"  "+dirty).
		Add("exposure", exposure).
		Add("route host", ac.m.ResolvedHost(exposure, ac.cfg.Defaults.TailnetBase)).
		Add("build", buildMode).
		Add("compose", ac.m.Compose.Template).
		Render()

	ui.Heading("Shipping")
	res, err := ship.Run(ac.cfg, ac.repoRoot, ac.m, ac.gi, opt)
	if err != nil {
		return err
	}

	lines := []string{
		ui.Dim("image    ") + ac.m.Image + ":" + ui.Bold(res.ImageTag),
		ui.Dim("pipeline ") + ui.Bold(fmt.Sprintf("#%d", res.Number)),
		ui.Dim("url      ") + ui.Link(res.PipelineURL),
	}
	if opt.Wait {
		ui.Box("Shipped", lines)
	} else {
		lines = append(lines, "", ui.Dim("tip: add ")+ui.Cyan("--wait")+ui.Dim(" to stream progress"))
		ui.Box("Pipeline started", lines)
	}
	return nil
}

func branchOf(opt ship.Options, gi *git.Info) string {
	if opt.Branch != "" {
		return opt.Branch
	}
	return gi.Branch
}

func resTag(opt ship.Options, gi *git.Info) string {
	if opt.Tag != "" {
		return opt.Tag
	}
	return "dev-" + gi.ShortSHA
}

func cmdStatus(args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ui.Logo("latest pipeline status")
	client := woodpecker.New(woodpecker.NormalizeURL(cfg.Woodpecker.URL), cfg.Woodpecker.Token)
	cachePath, _ := config.CachePath()

	sp := ui.NewSpinner("Fetching latest pipeline…").Start()
	repoID, err := client.RepoID(cfg.PersonalCloud.Owner, cfg.PersonalCloud.Repo, cachePath)
	if err != nil {
		sp.Fail("Could not resolve repo")
		return err
	}
	latest, err := client.LatestPipelineForRepo(repoID)
	if err != nil {
		sp.Fail("Could not list pipelines")
		return err
	}
	pl, err := client.GetPipeline(repoID, latest.Number)
	if err != nil {
		sp.Fail("Could not fetch pipeline")
		return err
	}
	sp.Success("Fetched pipeline %s", ui.Bold(fmt.Sprintf("#%d", pl.Number)))

	ui.Heading("Pipeline #" + fmt.Sprintf("%d", pl.Number))
	ui.NewDetails().
		Add("repo", cfg.PersonalCloud.Owner+"/"+cfg.PersonalCloud.Repo).
		Add("status", ui.Badge(pl.EffectiveStatus())).
		Render()

	if pl.Error != "" {
		ui.Fail("%s", pl.Error)
	}
	for _, pe := range pl.Errors {
		if pe != nil && pe.Message != "" {
			ui.Warn("%s", pe.Message)
		}
	}

	hasSteps := false
	for _, wf := range pl.Workflows {
		for range wf.Children {
			hasSteps = true
		}
	}
	if hasSteps {
		ui.Heading("Steps")
		for _, wf := range pl.Workflows {
			for _, step := range wf.Children {
				state := step.State
				if state == "" {
					state = "pending"
				}
				mark := ui.Gray("·")
				switch strings.ToLower(state) {
				case "success", "skipped":
					mark = ui.Green("✔")
				case "failure", "error", "killed":
					mark = ui.Red("✗")
				case "running", "started":
					mark = ui.Brand("•")
				}
				fmt.Fprintf(ui.Out, "  %s %-22s %s\n", mark, step.Name, ui.Badge(state))
				if step.Error != "" {
					fmt.Fprintf(ui.Out, "    %s\n", ui.Red(step.Error))
				}
			}
		}
	}

	fmt.Fprintln(ui.Out)
	ui.LinkLine("open", client.PipelineURL(repoID, pl.Number))
	return nil
}

func cmdLogs(args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	client := woodpecker.New(woodpecker.NormalizeURL(cfg.Woodpecker.URL), cfg.Woodpecker.Token)
	cachePath, _ := config.CachePath()

	sp := ui.NewSpinner("Finding latest pipeline…").Start()
	repoID, err := client.RepoID(cfg.PersonalCloud.Owner, cfg.PersonalCloud.Repo, cachePath)
	if err != nil {
		sp.Fail("Could not resolve repo")
		return err
	}
	pl, err := client.LatestPipelineForRepo(repoID)
	if err != nil {
		sp.Fail("Could not list pipelines")
		return err
	}
	sp.Success("Latest pipeline %s", ui.Bold(fmt.Sprintf("#%d", pl.Number)))
	ui.LinkLine("logs", client.PipelineURL(repoID, pl.Number))
	return nil
}
