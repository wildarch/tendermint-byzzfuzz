package docker

import (
	"log"
	"os/exec"
)

func PrepDockerCompose() {
	localNetStop := exec.Command("make", "localnet-stop")
	localNetStop.Dir = "third_party/tendermint-pct-instrumentation"
	err := localNetStop.Run()
	if err != nil {
		log.Fatalf("Failed to stop previous local net: %v", err)
	}

	dockerComposeUpNoStart := exec.Command("docker-compose", "up", "--no-start")
	dockerComposeUpNoStart.Dir = "third_party/tendermint-pct-instrumentation"
	err = dockerComposeUpNoStart.Run()
	if err != nil {
		log.Fatalf("Failed to prepare network: %v", err)
	}
}
