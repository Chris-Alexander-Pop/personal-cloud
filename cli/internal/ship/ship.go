package ship

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/your-github-username/personal-cloud/cli/internal/config"
	"github.com/your-github-username/personal-cloud/cli/internal/git"
	"github.com/your-github-username/personal-cloud/cli/internal/manifest"
	"github.com/your-github-username/personal-cloud/cli/internal/ui"
	"github.com/your-github-username/personal-cloud/cli/internal/woodpecker"
)

type Options struct {
	Local      bool
	Public     bool
	Private    bool
	Tag        string
	Branch     string
	AllowDirty bool
	Wait       bool
}

type Result struct {
	PipelineURL string
	Number      int64
	ImageTag    string
}

func Run(cfg *config.Config, repoRoot string, m *manifest.Manifest, gi *git.Info, opt Options) (*Result, error) {
	exposure := strings.ToLower(m.Route.Exposure)
	if opt.Public {
		exposure = "public"
	}
	if opt.Private {
		exposure = "private"
	}

	if gi.Dirty && !opt.AllowDirty {
		return nil, fmt.Errorf("working tree has uncommitted changes (use --allow-dirty)")
	}

	tag := opt.Tag
	if tag == "" {
		tag = "dev-" + gi.ShortSHA
	}

	branch := opt.Branch
	if branch == "" {
		branch = gi.Branch
	}

	doBuild := "true"
	if opt.Local {
		if err := localBuild(cfg, repoRoot, m, tag); err != nil {
			return nil, err
		}
		doBuild = "false"
	}

	routeHost := m.ResolvedHost(exposure, cfg.Defaults.TailnetBase)

	imageRepo := strings.TrimPrefix(m.Image, "ghcr.io/")
	imageRepo = strings.TrimPrefix(imageRepo, "docker.io/")

	vars := map[string]string{
		"APP_NAME":            m.Name,
		"SOURCE_REPO":         gi.Remote,
		"SOURCE_REF":          gi.FullSHA,
		"IMAGE":               m.Image,
		"IMAGE_REPO":          imageRepo,
		"IMAGE_TAG":           tag,
		"DO_BUILD":            doBuild,
		"EXPOSURE":            exposure,
		"ROUTE_HOST":          routeHost,
		"SERVICE_CONTAINER":   m.Service.Container,
		"SERVICE_PORT":        fmt.Sprintf("%d", m.Service.Port),
		"SERVICE_HEALTH_PATH": m.Service.HealthPath,
		"COMPOSE_TEMPLATE":    m.Compose.Template,
		"COMPOSE_ENV_FILE":    m.Compose.EnvFile,
		"BUILD_CONTEXT":       m.Build.Context,
		"BUILD_DOCKERFILE":    m.Build.Dockerfile,
		"TEST_SCRIPT":         m.Test,
	}

	client := woodpecker.New(woodpecker.NormalizeURL(cfg.Woodpecker.URL), cfg.Woodpecker.Token)
	cachePath, _ := config.CachePath()

	resolve := ui.NewSpinner("Resolving Woodpecker repo…").Start()
	repoID, err := client.RepoID(cfg.PersonalCloud.Owner, cfg.PersonalCloud.Repo, cachePath)
	if err != nil {
		resolve.Fail("Could not resolve repo")
		return nil, err
	}
	resolve.Success("Repo resolved %s", ui.Dim(fmt.Sprintf("(%s/%s)", cfg.PersonalCloud.Owner, cfg.PersonalCloud.Repo)))

	create := ui.NewSpinner("Triggering pipeline…").Start()
	pl, err := client.CreatePipeline(repoID, woodpecker.PipelineOptions{
		Branch:    branch,
		Variables: vars,
	})
	if err != nil {
		create.Fail("Pipeline trigger failed")
		return nil, err
	}
	create.Success("Pipeline %s triggered on %s", ui.Bold(fmt.Sprintf("#%d", pl.Number)), ui.Cyan(branch))

	res := &Result{
		PipelineURL: client.PipelineURL(repoID, pl.Number),
		Number:      pl.Number,
		ImageTag:    tag,
	}

	if opt.Wait {
		ui.Heading("Pipeline #" + fmt.Sprintf("%d", pl.Number))
		ui.LinkLine("watch", res.PipelineURL)
		fmt.Fprintln(os.Stdout)
		rep := newProgressReporter()
		_, werr := client.Wait(repoID, pl.Number, 3*time.Second, rep)
		rep.finish(werr)
		if werr != nil {
			return res, werr
		}
	}

	return res, nil
}

// progressReporter renders live Woodpecker step transitions with a spinner.
type progressReporter struct {
	sp        *ui.Spinner
	lastState string
}

func newProgressReporter() *progressReporter {
	return &progressReporter{sp: ui.NewSpinner("Waiting for runner…").Start()}
}

func (p *progressReporter) Status(number int64, status string) {
	p.lastState = status
	p.sp.Update("pipeline %s  %s", ui.Bold(fmt.Sprintf("#%d", number)), ui.Badge(status))
}

func (p *progressReporter) Step(workflow, step, state, errMsg string) {
	switch state {
	case "success", "skipped":
		p.sp.Println(fmt.Sprintf("  %s %s  %s", ui.Green("✔"), step, ui.StateDot(state)))
	case "failure", "error", "killed":
		p.sp.Println(fmt.Sprintf("  %s %s  %s", ui.Red("✗"), step, ui.StateDot(state)))
		if errMsg != "" {
			p.sp.Println("    " + ui.Red(errMsg))
		}
	case "running", "started":
		p.sp.Update("running %s", ui.Bold(step))
	default:
		// pending/waiting: keep spinner, note the step quietly
		p.sp.Update("queued %s", step)
	}
}

func (p *progressReporter) Linter(msg string) {
	p.sp.Println("  " + ui.Yellow("! ") + ui.Yellow(msg))
}

func (p *progressReporter) finish(err error) {
	if err != nil {
		p.sp.Fail("Pipeline failed")
		return
	}
	p.sp.Success("Pipeline succeeded")
}

func localBuild(cfg *config.Config, repoRoot string, m *manifest.Manifest, tag string) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not in PATH (required for --local)")
	}
	ctx := filepath.Join(repoRoot, m.Build.Context)
	df := filepath.Join(repoRoot, m.Build.Dockerfile)
	image := m.Image + ":" + tag

	args := []string{
		"buildx", "build",
		"--push",
		"-t", image,
		"-f", df,
		ctx,
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	token := cfg.GHCR.Token
	if token == "" {
		token = os.Getenv("GHCR_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token != "" {
		user := cfg.GitHub.Owner
		login := ui.NewSpinner("Logging in to ghcr.io…").Start()
		cmdLogin := exec.Command("docker", "login", "ghcr.io", "-u", user, "--password-stdin")
		cmdLogin.Stdin = strings.NewReader(token)
		if out, err := cmdLogin.CombinedOutput(); err != nil {
			login.Fail("docker login ghcr.io failed")
			return fmt.Errorf("docker login ghcr.io: %s: %w", strings.TrimSpace(string(out)), err)
		}
		login.Success("Authenticated to ghcr.io as %s", ui.Cyan(user))
	}

	ui.Step("Building & pushing %s", ui.Bold(image))
	fmt.Fprintln(os.Stdout)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker buildx: %w", err)
	}
	ui.Success("Image pushed %s", ui.Bold(image))
	return nil
}

// PipelineVariables builds the map for Woodpecker (exported for tests).
func PipelineVariables(m *manifest.Manifest, gi *git.Info, cfg *config.Config, exposure, tag, doBuild string) map[string]string {
	return map[string]string{
		"APP_NAME":          m.Name,
		"SOURCE_REPO":       gi.Remote,
		"SOURCE_REF":        gi.FullSHA,
		"IMAGE":             m.Image,
		"IMAGE_TAG":         tag,
		"DO_BUILD":          doBuild,
		"EXPOSURE":          exposure,
		"ROUTE_HOST":        m.ResolvedHost(exposure, cfg.Defaults.TailnetBase),
		"SERVICE_CONTAINER": m.Service.Container,
		"SERVICE_PORT":      fmt.Sprintf("%d", m.Service.Port),
	}
}
