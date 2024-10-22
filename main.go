package main

import (
	"log"
	"io/ioutil"
	"encoding/json"
)

func main() {
	filepath := "test.json"
	playbook, err := ParseFile(filepath)
	if err != nil {
		log.Printf("[ERROR] Failed to parse CACAO file %s: %s", filepath, err)
		return
	}

	log.Printf("[DEBUG] CACAO ID: %s", playbook.ID)

	// Dump to file
	shuffleWorkflow := TranslateToShuffle(playbook)
	shuffleData, err := json.MarshalIndent(shuffleWorkflow, "", "  ")
	if err != nil {
		log.Printf("[ERROR] Failed to marshal Shuffle data: %s", err)
		return
	}

	shuffleFilepath := "shuffle.json"

	if err := ioutil.WriteFile(shuffleFilepath, shuffleData, 0644); err != nil {
		log.Printf("[ERROR] Failed to write Shuffle data: %s", err)
		return
	}

	log.Printf("[DEBUG] Done writing Shuffle data to %s...", shuffleFilepath)
}
