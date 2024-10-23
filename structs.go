package main

import (
	"time"
)

type Command struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Content     string `json:"content"`
	Headers     map[string][]string `json:"headers"`
}

type Action struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	OnCompletion string `json:"on_completion"`
	Type         string `json:"type"`
	Commands  []Command `json:"commands"`
	Agent   string   `json:"agent"`
	Targets []string `json:"targets"`

	// Conditions 
	Condition    string   `json:"condition"`
	OnTrue       string   `json:"on_true"`
	InArgs       []string `json:"in_args"`
}

type Target struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Address struct {
		Ipv4 []string `json:"ipv4"`
		Ipv6 []string `json:"ipv6"`
		Url  []string `json:"url"`
		Domain []string `json:"domain"`
	} `json:"address"`
	AuthenticationInfo string `json:"authentication_info"`
	Port               string `json:"port"`
} 

type Agent struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type Variable struct {
	Type     string `json:"type"`
	Constant bool   `json:"constant"`
	External bool   `json:"external"`
	Value    string `json:"value"`
}

type AuthenticationInfo struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Kms      bool   `json:"kms"`
}

type Cacao struct {
	Type              string    `json:"type"`
	SpecVersion       string    `json:"spec_version"`
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	CreatedBy         string    `json:"created_by"`
	Created           time.Time `json:"created"`
	Modified          time.Time `json:"modified"`
	Revoked           bool      `json:"revoked"`
	WorkflowStart string `json:"workflow_start"`
	DerivedFrom []string `json:"derived_from"`

	Workflow map[string]Action `json:"workflow"`
	PlaybookVariables map[string]Variable `json:"playbook_variables"`

	AgentDefinitions map[string]Agent `json:"agent_definitions"`
	TargetDefinitions map[string]Target `json:"target_definitions"`

	AuthenticationInfoDefinitions map[string]AuthenticationInfo `json:"authentication_info_definitions"`
}
