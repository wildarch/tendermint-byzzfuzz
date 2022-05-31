package spec

import (
	"encoding/json"
	"log"
	"os"
)

func Check(ch chan Event) {
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
}
