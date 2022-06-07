package podmanager

import (
	"context"
	"testing"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/pkg/api"
)

func Test_remove(t *testing.T) {
	dcp := newDockerComposeManager()
	err := dcp.api.Remove(context.TODO(), "compose", api.RemoveOptions{
		Services: []string{"ubuntu-bygolang"},
		Force:    true,
		DryRun:   true,
	})
	t.Log("remove err:", err)
}

func Test_createAndRun(t *testing.T) {
	dcp := newDockerComposeManager()
	label := types.Labels{}
	label.Add(api.ServiceLabel, "ubuntu-bygolang")
	label.Add(api.ProjectLabel, "compose")
	label.Add(api.OneoffLabel, "False")
	label.Add(api.WorkingDirLabel, "/mnt/c/Users/LinHao/go/test/compose")
	label.Add(api.ConfigFilesLabel, "docker-compose.yml")

	project := types.Project{
		Name: "compose",
		Services: types.Services{
			types.ServiceConfig{
				Name:         "ubuntu-bygolang",
				Command:      types.ShellCommand{"sleep", "10d"},
				Image:        "ubuntu:latest",
				CustomLabels: types.Labels{},
				Labels:       label,
				Scale:        1,
				Restart:      "always",
			},
		},
	}

	err := dcp.api.Up(context.TODO(), &project, api.UpOptions{
		Create: api.CreateOptions{
			Inherit: true,
			//Recreate: "force",
		},
		Start: api.StartOptions{Project: &project},
	})
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("up success")
}

func Test_ps(t *testing.T) {
	dcp := newDockerComposeManager()
	sum, err := dcp.api.Ps(context.TODO(), "compose", api.PsOptions{})
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(sum)
}

func Test_convert(t *testing.T) {
	dcp := newDockerComposeManager()
	pro := &types.Project{
		Name: "compose",
	}
	dcp.api.Convert(context.Background(), pro, api.ConvertOptions{Format: "yaml", Output: "/mnt/c/Users/LinHao/go/test"})
}
