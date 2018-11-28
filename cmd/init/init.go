package init

import (
	"errors"
	"fmt"
	"os"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/tenderly/tenderly-cli/config"
	"github.com/tenderly/tenderly-cli/model"
	"github.com/tenderly/tenderly-cli/rest"
	"github.com/tenderly/tenderly-cli/rest/call"
)

func Start(rest rest.Rest) {
	if !config.IsLoggedIn() {
		fmt.Println("In order to use the tenderly CLI, you need to login first.")
		fmt.Println("")
		fmt.Println("Please use the", aurora.Bold(aurora.Cyan("tenderly login")), "command to get started.")
		os.Exit(0)
	}

	projects, err := rest.Project.GetProjects(config.GetOrganisation())
	if err != nil {
		fmt.Println("unable to fetch projects")
		os.Exit(0)
	}

	project, err := promptProjectSelect(projects, rest)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	config.SetProjectConfig(config.ProjectName, project.Name)
	config.SetProjectConfig(config.ProjectSlug, project.Slug)
	config.SetProjectConfig(config.Organisation, config.GetOrganisation())
	config.WriteProjectConfig()
}

func promptDefault(attribute string) (string, error) {
	validate := func(input string) error {
		if len(input) < 6 {
			return errors.New("project name must have more than 6 characters")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    attribute,
		Validate: validate,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func promptProjectSelect(projects []*model.Project, rest rest.Rest) (*model.Project, error) {
	var projectNames []string
	projectNames = append(projectNames, "Create new project")
	for _, project := range projects {
		projectNames = append(projectNames, project.Name)
	}

	promptProjects := promptui.Select{
		Label: "Select Project",
		Items: projectNames,
	}

	_, result, err := promptProjects.Run()
	if err != nil {
		return nil, fmt.Errorf("Prompt failed %v\n", err)
	}

	// TODO refactor
	if result == "Create new project" {
		name, err := promptDefault("Project")
		if err != nil {
			return nil, fmt.Errorf("Prompt failed %v\n", err)
		}

		project, err := rest.Project.CreateProject(
			call.ProjectRequest{
				Name: name,
			})
		if err != nil {
			return nil, fmt.Errorf("Request failed %v\n", err)
		}

		return project, nil
	}

	for _, project := range projects {
		if result == project.Name {
			return project, nil
		}
	}

	return nil, fmt.Errorf("Prompt failed %v\n", err)
}
