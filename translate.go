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


		configureAction := false
		if strings.HasPrefix(key, "start--") {
			newAction.Label = "start"
			shuffleWorkflow.Start = newAction.ID
			newAction.IsStartNode = true

			if len(value.Commands) > 0 {
				configureAction = true
			} else {
				continue
			}
		} else if strings.HasPrefix(key, "end--") {
			newAction.Label = "end"

			if len(value.Commands) > 0 {
				configureAction = true
			} else {
				continue
			}
		} else if strings.HasPrefix(key, "action--") {
			actions[key] = value

			newAction.Label = value.Name
			configureAction = true
		} else {
			log.Printf("\n\n[ERROR] Unknown key: %s\n\n", key)
			continue
		}

		branchAdded := false
		if len(value.OnCompletion) > 0 {
			// Branch addition
			if strings.Contains(value.OnCompletion, "--") {
				shuffleWorkflow.Branches = append(shuffleWorkflow.Branches, shuffle.Branch{
					ID: uuid.New().String(),
					SourceID: newActionID, 
					DestinationID: strings.Split(value.OnCompletion, "--")[1],
					Label: "From " + strings.Split(key, "--")[0],
				})

				branchAdded = true 


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

		appended := false
		if configureAction {
			log.Printf("[DEBUG] Configuring action: %s", key)

			relevantTarget := Target{}
			relevantAuth := AuthenticationInfo{}
			if len(value.Targets) > 0 {
				// Mapping targets to the commands for fillin
				if len(value.Targets) > 0 {
					log.Printf("\n\n[ERROR] Only allowing one CACAO target for now. ID: %s", cacao.ID)
				}

				for _, target := range value.Targets {
					log.Printf("[DEBUG] Target: %s", target)

					// Check for the target in the actions
					if _, ok := cacao.TargetDefinitions[target]; !ok { 
						log.Printf("\n\n[ERROR] Target not found in cacao: %s\n\n", target)
						continue
					}

					// Check for the agent in the actions
					relevantTarget = cacao.TargetDefinitions[target]
					if len(relevantTarget.AuthenticationInfo) > 0 {
						if _, ok := cacao.AuthenticationInfoDefinitions[relevantTarget.AuthenticationInfo]; !ok {
							log.Printf("\n\n[ERROR] AuthenticationInfo not found in cacao: %s\n\n", relevantTarget.AuthenticationInfo)
							continue
						} 

						relevantAuth = cacao.AuthenticationInfoDefinitions[relevantTarget.AuthenticationInfo]
					}

					break
				}
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

				if len(relevantAuth.Type) > 0 || len(relevantAuth.Name) > 0 {
					if relevantAuth.Type == "basic" || relevantAuth.Type == "http-basic" || relevantAuth.Type == "user-auth" { 
						newAction.Parameters = append(newAction.Parameters, shuffle.WorkflowAppActionParameter{
							Name: "username",
							Value: "",

							Configuration: true,
							Required: true,
						})
						newAction.Parameters = append(newAction.Parameters, shuffle.WorkflowAppActionParameter{
							Name: "password",
							Value: "",

							Configuration: true,
							Required: true,
						})
					} else {
						log.Printf("\n\n[ERROR] Type %s for auth translation from CACAO is not supported yet.", relevantAuth.Type)
					}
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

					if len(relevantTarget.Type) > 0 || len(relevantTarget.Name) > 0 {
						log.Printf("[DEBUG] Target (%s): %s", relevantTarget.Type, relevantTarget.Name)
						newStartinfo := "http://"
						if relevantTarget.Port == "443" {
							newStartinfo = "https://"
						} 

						if len(relevantTarget.Address.Domain) > 0 {
							if len(relevantTarget.Address.Domain) > 1 {
								log.Printf("[ERROR] Using first domain to map because there are %d for target %s", len(relevantTarget.Address.Domain), relevantTarget.Name)
							}

							newStartinfo += relevantTarget.Address.Domain[0]
						} else if len(relevantTarget.Address.Url) > 0 {
							if len(relevantTarget.Address.Ipv4) > 1 {
								log.Printf("[ERROR] Using first url to map because there are %d for target %s", len(relevantTarget.Address.Ipv4), relevantTarget.Name)
							}

							newStartinfo = relevantTarget.Address.Url[0]
						} else if len(relevantTarget.Address.Ipv4) > 0 {
							if len(relevantTarget.Address.Ipv4) > 1 {
								log.Printf("[ERROR] Using first ipv4 address to map because there are %d for target %s", len(relevantTarget.Address.Ipv4), relevantTarget.Name)
							}

							newStartinfo += relevantTarget.Address.Ipv4[0]
						} else if len(relevantTarget.Address.Ipv6) > 0 {
							if len(relevantTarget.Address.Ipv6) > 1 {
								log.Printf("[ERROR] Using first ipv6 address to map because there are %d for target %s", len(relevantTarget.Address.Ipv6), relevantTarget.Name)
							}

							newStartinfo += relevantTarget.Address.Ipv6[0]
						} else {
							log.Printf("[ERROR] No address or name found for target %s", relevantTarget.Name)
						}

						if relevantTarget.Port != "80" && relevantTarget.Port != "443" {
							if !strings.Contains(newStartinfo, ":") {
								if !strings.HasSuffix(newStartinfo, "/") {
									newStartinfo += newStartinfo+":"+relevantTarget.Port
								} else {
									newStartinfo += newStartinfo[0:len(newStartinfo)-1]+":"+relevantTarget.Port+"/"
								}
							}
						}

						url = fmt.Sprintf("%s%s", newStartinfo, url)
					}

					//relevantTarget := Target{}
					//relevantAuth := AuthenticationInfo{}

					headerString := ""
					for headerKey, headerValue := range command.Headers {
						if len(headerValue) == 0 {
							continue
						}

						headerString += fmt.Sprintf("%s: %s\n", headerKey, headerValue[0])
					}

					newAction.Name = method
					newAction.Parameters = append(newAction.Parameters,
						shuffle.WorkflowAppActionParameter{
							Name: "url",
							Value: url,

							Required: true,
							Configuration: true,
						},
					)

					newAction.Parameters = append(newAction.Parameters,
						shuffle.WorkflowAppActionParameter{
							Name: "headers",
							Value: headerString,

							Required: false,
							Multiline: true,
						},
					)

					newAction.Parameters = append(newAction.Parameters,
						shuffle.WorkflowAppActionParameter{
							Name: "verify",
							Value: "false",

							Required: false,
						},
					)

					newAction.Parameters = append(newAction.Parameters,
						shuffle.WorkflowAppActionParameter{
							Name: "timeout",
							Value: "30",

							Required: false,
						},
					)

					if len(command.Content) > 0 {
						newAction.Parameters = append(newAction.Parameters, shuffle.WorkflowAppActionParameter{
							Name: "body",
							Value: command.Content,
							Multiline: true,

							Required: false,
						})
					}

				}

				// Handles proper branching
				if len(value.Commands) > 1 {
					appended = true
					//if commandCount != 0 && commandCount != len(value.Commands)-1 {
					if commandCount != 0 {
						if len(nextActionID) == 0 {
							newAction.ID = uuid.New().String()
						} else {
							newAction.ID = nextActionID
						}
					}

					if commandCount == 0 && branchAdded {
						// Remove last branch
						shuffleWorkflow.Branches = shuffleWorkflow.Branches[:len(shuffleWorkflow.Branches)-1]
					}

					parsedLabel := fmt.Sprintf("command %d to %d", commandCount+1, commandCount+2)

					nextActionID = uuid.New().String()
					if commandCount == len(value.Commands)-1 {
						if len(value.OnCompletion) > 0 {
							nextActionID = strings.Split(value.OnCompletion, "--")[1]

							parsedLabel = fmt.Sprintf("command %d to %s", commandCount+1, strings.Split(value.OnCompletion, "--")[0])
						}
					}

					shuffleWorkflow.Branches = append(shuffleWorkflow.Branches, shuffle.Branch{
						ID: uuid.New().String(),
						SourceID: newAction.ID,
						DestinationID: nextActionID,

						Label: parsedLabel,
					})


					shuffleWorkflow.Actions = append(shuffleWorkflow.Actions, newAction)
					continue
				}
			}

			// Not re-appending an action
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
