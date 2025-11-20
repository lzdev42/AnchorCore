//go:build generate && generate_completions

package main

import "github.com/sagernet/sing-box/log"

func main() {
	err := generateCompletions()
	if err != nil {
		log.Fatal(err)
	}
}

func generateCompletions() error {
	err := mainCommand.GenBashCompletionFile("release/completions/anchor.bash")
	if err != nil {
		return err
	}
	err = mainCommand.GenFishCompletionFile("release/completions/anchor.fish", true)
	if err != nil {
		return err
	}
	err = mainCommand.GenZshCompletionFile("release/completions/anchor.zsh")
	if err != nil {
		return err
	}
	return nil
}
