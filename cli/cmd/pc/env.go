package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/your-org/personal-cloud/cli/internal/config"
	"github.com/your-org/personal-cloud/cli/internal/env"
	"github.com/your-org/personal-cloud/cli/internal/manifest"
	"github.com/your-org/personal-cloud/cli/internal/ui"
)

func envUsage() {
	ui.Heading("Env commands")
	type cmd struct{ name, args, desc string }
	cmds := []cmd{
		{"init", "[--force]", "copy apps/<name>/.env.example → ~/.config/pc/env/<name>.env"},
		{"push", "[--file PATH]", "upload local env to the VM (compose.env_file)"},
	}
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
		fmt.Fprintf(ui.Out, "  %s %s%s  %s\n", ui.Brand("pc env"), ui.Bold(label), pad, ui.Dim(c.desc))
	}
	ui.NewDetails().
		Add("local secrets", "~/.config/pc/env/<app>.env").
		Add("example source", "<personal_cloud.local_path>/apps/<app>/.env.example").
		Add("remote target", "compose.env_file (default /opt/personal-cloud/apps/<app>/.env)").
		Render()
	fmt.Fprintln(ui.Out)
}

func cmdEnv(args []string) error {
	if len(args) == 0 {
		ui.Logo("manage app secrets on the VM")
		envUsage()
		return nil
	}
	switch args[0] {
	case "init":
		return cmdEnvInit(args[1:])
	case "push":
		return cmdEnvPush(args[1:])
	case "help", "-h", "--help":
		ui.Logo("manage app secrets on the VM")
		envUsage()
		return nil
	default:
		return fmt.Errorf("unknown env subcommand %q (try: pc env init, pc env push)", args[0])
	}
}

func loadManifestConfig() (*manifest.Manifest, *config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	_, m, err := manifest.Find(cwd)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	return m, cfg, nil
}

func cmdEnvInit(args []string) error {
	force := false
	for _, a := range args {
		switch a {
		case "--force":
			force = true
		case "-h", "--help":
			fmt.Fprintln(ui.Out, "pc env init — copy .env.example to ~/.config/pc/env/<app>.env")
			return nil
		default:
			return fmt.Errorf("unknown flag %s", a)
		}
	}

	ui.Logo("env init")
	load := ui.NewSpinner("Loading manifest & config…").Start()
	m, cfg, err := loadManifestConfig()
	if err != nil {
		load.Fail("Could not load context")
		return err
	}
	load.Success("App %s", ui.Bold(m.Name))

	src, err := env.ExampleSource(m.Name, cfg)
	if err != nil {
		return err
	}

	copySpin := ui.NewSpinner("Copying .env.example…").Start()
	dest, err := env.Init(m, cfg, force)
	if err != nil {
		copySpin.Fail("Could not create local env file")
		return err
	}
	copySpin.Success("Created %s", ui.Dim(dest))

	ui.Heading("Paths")
	ui.NewDetails().
		Add("from", src).
		Add("to", dest).
		Render()

	ui.Box("Next steps", []string{
		ui.Dim("1. ") + "Edit secrets in " + ui.Bold(dest),
		ui.Dim("2. ") + "pc env push",
		ui.Dim("3. ") + "pc ship --wait",
	})
	return nil
}

func cmdEnvPush(args []string) error {
	var file string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			fmt.Fprintln(ui.Out, "pc env push — upload ~/.config/pc/env/<app>.env to the VM")
			return nil
		case "--file":
			i++
			if i >= len(args) {
				return fmt.Errorf("--file requires a value")
			}
			file = args[i]
		default:
			if strings.HasPrefix(args[i], "--file=") {
				file = strings.TrimPrefix(args[i], "--file=")
				continue
			}
			return fmt.Errorf("unknown flag %s", args[i])
		}
	}

	ui.Logo("env push")
	load := ui.NewSpinner("Loading manifest & config…").Start()
	m, cfg, err := loadManifestConfig()
	if err != nil {
		load.Fail("Could not load context")
		return err
	}
	load.Success("App %s", ui.Bold(m.Name))

	sshHost := strings.TrimSpace(cfg.VM.SSH)
	if sshHost == "" {
		return fmt.Errorf("vm.ssh is required in ~/.config/pc/config.yaml (SSH config Host alias, e.g. deploy)")
	}

	local := file
	if local == "" {
		local, err = env.LocalPath(m.Name)
		if err != nil {
			return err
		}
	}

	remote := m.Compose.EnvFile
	if remote == "" {
		remote = fmt.Sprintf("/opt/personal-cloud/apps/%s/.env", m.Name)
	}

	ui.Heading("Transfer")
	ui.NewDetails().
		Add("local", local).
		Add("remote", remote).
		Add("via", sshHost).
		Render()

	push := ui.NewSpinner(fmt.Sprintf("Pushing to %s…", sshHost)).Start()
	res, err := env.Push(m, cfg, file)
	if err != nil {
		push.Fail("Push failed")
		return err
	}
	push.Success("Uploaded to %s", ui.Bold(res.SSHHost))

	ui.Box("Env on VM", []string{
		ui.Green("✔ ") + res.Local,
		ui.Dim("  → ") + res.SSHHost + ":" + res.Remote,
	})
	return nil
}
