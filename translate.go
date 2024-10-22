package main

import (
	"log"
	"fmt"
	"strings"
	"io/ioutil"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/shuffle/shuffle-shared"
)

func ParseFile(path string) (*Cacao, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("[ERROR] Failed to read file: %s", err)
		return nil, err
	}

	return ParseCacao(data)
}

func ParseCacao(data []byte) (*Cacao, error) {
	var cacao Cacao
	if err := json.Unmarshal(data, &cacao); err != nil {
		log.Printf("[ERROR] Failed to unmarshal Cacao data: %s", err)

		return &cacao, err
	}

	return &cacao, nil
}

func TranslateToShuffle(cacao *Cacao) shuffle.Workflow {
	shuffleWorkflow := shuffle.Workflow{
		ID: cacao.ID,
		Name: cacao.Name,
		Description: cacao.Description,

		UpdatedBy: cacao.CreatedBy,
		Created: cacao.Created.Unix(),
		Edited: cacao.Modified.Unix(),

		WorkflowType: "CACAO "+cacao.Type,
	}


	if strings.Contains(cacao.ID, "--") {
		shuffleWorkflow.ID = strings.Split(cacao.ID, "--")[1]
	}

	log.Printf("[DEBUG] Variables in cacao: %d", len(cacao.Workflow))
	log.Printf("[DEBUG] Actions in cacao: %d", len(cacao.Workflow))

	actions := make(map[string]Action)
	for key, value := range cacao.Workflow {
		newAction := shuffle.Action{
			AppName: "shuffle tools",
			AppVersion: "1.2.0",
			Name: value.Name,
			Description: value.Description,
		}

		if strings.Contains(key, "--") {
			newAction.ID = strings.Split(key, "--")[1]
		} else {
			log.Printf("\n\n[ERROR] No ID found in key: %s\n\n", key)
			continue
		}

		if len(value.OnCompletion) > 0 {
			// Branch addition
			if strings.Contains(value.OnCompletion, "--") {
				shuffleWorkflow.Branches = append(shuffleWorkflow.Branches, shuffle.Branch{
					ID: uuid.New().String(),
					SourceID: newAction.ID,
					DestinationID: strings.Split(value.OnCompletion, "--")[1],
				})


			} else {
				log.Printf("\n\n[ERROR] No ID found in OnCompletion: %s\n\n", value.OnCompletion)
				continue
			}
		} else {
			if !strings.Contains(key, "end-") {
				log.Printf("\n\n[ERROR] No OnCompletion found for key: %s\n\n", key)
				continue
			} 
		}

		configureAction := false
		if strings.HasPrefix(key, "start--") {
			newAction.Label = "start"
			shuffleWorkflow.Start = newAction.ID
			newAction.IsStartNode = true

			if len(value.Commands) > 0 {
				configureAction = true
			}
		} else if strings.HasPrefix(key, "end--") {
			newAction.Label = "end"

			if len(value.Commands) > 0 {
				configureAction = true
			}
		} else if strings.HasPrefix(key, "action--") {
			actions[key] = value
			configureAction = true
		} else {
			log.Printf("\n\n[ERROR] Unknown key: %s\n\n", key)
			continue
		}

		if configureAction {
			log.Printf("[DEBUG] Configuring action: %s", key)

			for _, command := range value.Commands {
				if command.Type == "http-api" || command.Type == "http" {
					newAction.AppName = "http"
					newAction.AppVersion = "1.4.0"

					// Parse out the command:
					method := "GET"
					url := command.Command


                    // "command": "POST /api/firewall/filter/addRule HTTP/1.1",
					commandSplit := strings.Split(command.Command, " ")
					if len(commandSplit) > 1 {
						method = commandSplit[0]
					}

					if len(commandSplit) > 2 {
						url = commandSplit[1]
					}

					headerString := ""
					for headerKey, headerValue := range command.Headers {
						headerString += fmt.Sprintf("%s: %s\n", headerKey, headerValue)
					}

					newAction.Name = method
					newAction.Parameters = []shuffle.WorkflowAppActionParameter{
						shuffle.WorkflowAppActionParameter{
							Name: "url",
							Value: url,
						},
						shuffle.WorkflowAppActionParameter{
							Name: "headers",
							Value: headerString,
						},
						shuffle.WorkflowAppActionParameter{
							Name: "body",
							Value: command.Content,
						},
					}

				}
			}


			if len(value.Targets) > 0 {
				log.Printf("\n\n[WARNING] Adding target configs to action: %s\n\n", key)
			}

		} else {
			newAction.Name = "repeat_back_to_me"
			if strings.Contains(key, "--") {
				newAction.Label = strings.Split(key, "--")[1]
			} else {
				newAction.Label = key
			}

			newAction.Parameters = []shuffle.WorkflowAppActionParameter{
				shuffle.WorkflowAppActionParameter{
					Name: "call",
					Value: key,
				},
			}
		}

		shuffleWorkflow.Actions = append(shuffleWorkflow.Actions, newAction)
	}

	return shuffleWorkflow
}
