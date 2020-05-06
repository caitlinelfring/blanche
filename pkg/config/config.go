package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/RentTheRunway/blanche/pkg/gh"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"
)

const (
	ErrTagNotValid = "Tag is not valid semver"
)

type ManifestConfigs []ManifestConfig

type ManifestConfig struct {
	DockerRepo string `yaml:"docker_repo"`
	Manifests  []struct {
		ConfigRepo  string `yaml:"config_repo"`
		File        string `yaml:"file"`
		BaseBranch  string `yaml:"base_branch"`
		PullRequest bool   `yaml:"pull_request"`
	}
}

func (mcs *ManifestConfigs) GetManifest(dockerRepo string) *ManifestConfig {
	for _, m := range *mcs {
		if m.DockerRepo == dockerRepo {
			return &m
		}
	}
	return nil
}

func (mcs *ManifestConfigs) Load(filename string) error {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlFile, &mcs)
}

func (m *ManifestConfig) GenerateGitUpdates(name, tag string) error {
	// Only update tags that match semver
	if !(semver.IsValid(tag)) {
		log.Printf("tag was not semver, skipping: %s", tag)
		return errors.New(ErrTagNotValid)
	}

	for _, mc := range m.Manifests {
		targetBranch := fmt.Sprintf("auto-release/%s/%s-%s", mc.BaseBranch, name, tag)
		if !mc.PullRequest {
			targetBranch = mc.BaseBranch
		}
		repoOwner := strings.Split(mc.ConfigRepo, "/")[0]
		repoName := strings.Split(mc.ConfigRepo, "/")[1]
		if err := gh.CreateGitUpdates(
			repoOwner,
			repoName,
			mc.File,
			mc.BaseBranch,
			targetBranch,
			name,
			tag,
			mc.PullRequest,
		); err != nil {
			log.Printf("%s:%s | %s\n%+v", name, tag, err, mc)
		}
	}

	return nil
}
