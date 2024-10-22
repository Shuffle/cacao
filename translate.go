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
			AppName: "Shuffle Tools",
			AppVersion: "1.2.0",
			Label: value.Name,
			Description: value.Description,
		}

		newActionID := ""
		if strings.Contains(key, "--") {
			newActionID = strings.Split(key, "--")[1]
			newAction.ID = newActionID
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

			newAction.Label = value.Name
			configureAction = true
		} else {
			log.Printf("\n\n[ERROR] Unknown key: %s\n\n", key)
			continue
		}

		appended := false
		if configureAction {
			log.Printf("[DEBUG] Configuring action: %s", key)

			if len(value.Targets) > 0 {
				log.Printf("\n\n[WARNING] Adding target configs to action: %s\n\n", key)
			}

			nextActionID := ""
			for commandCount, command := range value.Commands {
				// From the top just in case it's not overridden
				//if len(newActionID) > 0 { 
				//	newAction.ID = newActionID
				//}

				if len(value.Commands) > 1 {
					newAction.Label = fmt.Sprintf("%d", commandCount+1)+"-"+value.Name
				}

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
						if len(headerValue) == 0 {
							continue
						}

						headerString += fmt.Sprintf("%s: %s\n", headerKey, headerValue[0])
					}

					newAction.Name = method
					newAction.Parameters = []shuffle.WorkflowAppActionParameter{
						shuffle.WorkflowAppActionParameter{
							Name: "url",
							Value: url,

							Required: true,
						},
						shuffle.WorkflowAppActionParameter{
							Name: "headers",
							Value: headerString,

							Required: false,
						},
						shuffle.WorkflowAppActionParameter{
							Name: "verify",
							Value: "false",

							Required: false,
						},
						shuffle.WorkflowAppActionParameter{
							Name: "timeout",
							Value: "10",

							Required: false,
						},
					}

					if len(command.Content) > 0 {
						newAction.Parameters = append(newAction.Parameters, shuffle.WorkflowAppActionParameter{
							Name: "body",
							Value: command.Content,

							Required: false,
						})
					}
				}

				if len(value.Commands) > 1 {
					appended = true


					if commandCount != 0 && commandCount != len(value.Commands)-1 {
						if len(nextActionID) == 0 {
							newAction.ID = uuid.New().String()
						} else {
							newAction.ID = nextActionID
						}
					}

					nextActionID = uuid.New().String()
					shuffleWorkflow.Branches = append(shuffleWorkflow.Branches, shuffle.Branch{
						ID: uuid.New().String(),
						SourceID: newAction.ID,
						DestinationID: nextActionID,
					})


					shuffleWorkflow.Actions = append(shuffleWorkflow.Actions, newAction)
					continue
				}
			}

			if appended { 
				continue
			}
		} else {
			newAction.Name = "repeat_back_to_me"

			if len(newAction.Label) == 0 {
				if strings.Contains(key, "--") {
					newAction.Label = strings.Split(key, "--")[1]
				} else {
					newAction.Label = key
				}
			}

			newAction.Parameters = []shuffle.WorkflowAppActionParameter{
				shuffle.WorkflowAppActionParameter{
					Name: "call",
					Value: key,

					Required: true,
				},
			}
		}

		shuffleWorkflow.Actions = append(shuffleWorkflow.Actions, newAction)
	}

	return shuffleWorkflow
}
