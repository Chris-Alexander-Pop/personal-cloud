package ship

import "testing"

func TestParseArgs(t *testing.T) {
	opt, err := ParseArgs([]string{"music-serve", "--wait", "--ref", "main", "--private"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Repo != "music-serve" || !opt.Wait || opt.Ref != "main" || !opt.Private {
		t.Fatalf("%+v", opt)
	}

	opt, err = ParseArgs([]string{"--repo=owner/music-serve", "--tag=v1"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Repo != "owner/music-serve" || opt.Tag != "v1" {
		t.Fatalf("%+v", opt)
	}

	if _, err := ParseArgs([]string{"--nope"}); err == nil {
		t.Fatal("expected unknown flag")
	}
	if _, err := ParseArgs([]string{"a", "b"}); err == nil {
		t.Fatal("expected extra positional error")
	}
	if _, err := ParseArgs([]string{"--help"}); !IsHelp(err) {
		t.Fatalf("help: %v", err)
	}
}
