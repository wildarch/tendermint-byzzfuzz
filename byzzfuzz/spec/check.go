package spec

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
)

func Check(ch chan Event) bool {
	// We don't expect more messages
	close(ch)

	f, err := os.Create("spec.log")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	for len(ch) > 0 {
		event := <-ch
		js, err := json.Marshal(event)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write(js)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.WriteString("\n")
		if err != nil {
			log.Fatal(err)
		}
	}

	f.Sync()

	return runAnalysis()
}

func runAnalysis() bool {
	cmd := exec.Command("python3", "analyse.py")
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err == nil {
		return true
	}
	exitErr, ok := err.(*exec.ExitError)
	if ok {
		return exitErr.Success()
	} else {
		log.Fatalf("Analysis failed to run: %v", err)
		return false
	}
}
