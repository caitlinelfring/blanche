package gh

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/go-github/v31/github"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

var ctx = context.Background()
var client *github.Client

const (
	ErrTagMatchesCurrentTag  = "New tag matches the tag in existing manifest"
	ErrTagPrecedesCurrentTag = "New tag precedes existing tag"
)

type gitUpdate struct {
	RepoOwner    string
	RepoName     string
	DockerImage  string
	Tag          string
	ManifestFile string
	BaseBranch   string
	TargetBranch string
	PullRequest  bool // setting to false will push a commit directly to the BaseBranch
}

func init() {
	CreateGithubClient(os.Getenv("GITHUB_ACCESS_TOKEN"))
}

func CreateGithubClient(accessToken string) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client = github.NewClient(tc)
	if octocat, _, err := client.Octocat(ctx, "Nobody ever believes me when Im telling the truth - I guess its the curse of being a devastatingly beautiful woman"); err == nil {
		log.Println(octocat)
	}
}

func CreateGitUpdates(repoOwner, repoName, manifest, baseBranch, targetBranch, dockerImage, tag string, pullRequest bool) error {
	g := gitUpdate{
		RepoOwner:    repoOwner,
		RepoName:     repoName,
		DockerImage:  dockerImage,
		Tag:          tag,
		ManifestFile: manifest,
		BaseBranch:   baseBranch,
		TargetBranch: targetBranch,
		PullRequest:  pullRequest,
	}
	return g.CreateUpdates()
}

// CreateUpdates will create PRs/push commits for the given ManifestUpdate
func (g *gitUpdate) CreateUpdates() (err error) {
	baseRef := "refs/heads/" + g.BaseBranch
	targetRef := "refs/heads/" + g.TargetBranch

	ref, err := g.createRef(baseRef, targetRef)
	if err != nil {
		log.Println(err)
		return
	}

	tree, err := g.newTreeWithChanges(ref)
	if err != nil {
		log.Println(err)
		return
	}

	if err = g.pushCommit(ref, tree); err != nil {
		log.Println(err)
		return
	}
	if g.PullRequest {
		prURL, errInner := g.createPR(targetRef, baseRef)
		if errInner != nil {
			log.Println(errInner)
			return errInner
		}
		log.Printf("Opened new PR: %s | %+v", prURL, g)
	} else {
		log.Printf("Pushed new commit: %s | %+v", tree.Entries[0].GetURL(), g)
	}
	return
}

func (g *gitUpdate) createRef(baseBranchRef, targetBranchRef string) (ref *github.Reference, err error) {
	if ref, _, err = client.Git.GetRef(
		ctx,
		g.RepoOwner,
		g.RepoName,
		targetBranchRef); err == nil {
		// ref already exists
		return ref, nil
	}

	// Get base branch ref to create new ref from
	baseRef, _, err := client.Git.GetRef(
		ctx,
		g.RepoOwner,
		g.RepoName,
		baseBranchRef,
	)
	if err != nil {
		return nil, err
	}

	newRef := &github.Reference{
		Ref:    github.String(targetBranchRef),
		Object: &github.GitObject{SHA: baseRef.Object.SHA},
	}
	ref, _, err = client.Git.CreateRef(ctx, g.RepoOwner, g.RepoName, newRef)
	return ref, err
}

func (g *gitUpdate) getManifestFileContents(ref *github.Reference) (config map[interface{}]interface{}, err error) {
	contents, _, _, err := client.Repositories.GetContents(
		ctx,
		g.RepoOwner,
		g.RepoName,
		g.ManifestFile,
		&github.RepositoryContentGetOptions{Ref: ref.GetRef()},
	)
	if err != nil {
		return
	}
	sContents, err := contents.GetContent()
	if err != nil {
		return
	}

	err = yaml.Unmarshal([]byte(sContents), &config)
	return
}

func updateImageTag(contents map[interface{}]interface{}, newTag string) ([]byte, error) {
	imageMap := contents["image"].(map[interface{}]interface{})
	currentTag := imageMap["tag"].(string)
	if currentTag == newTag {
		return nil, errors.New(ErrTagMatchesCurrentTag)
	}

	// The result will be 0 if a == b, -1 if a < b, or +1 if a > b.
	if semver.Compare(currentTag, newTag) >= 0 {
		return nil, errors.New(ErrTagPrecedesCurrentTag)
	}

	imageMap["tag"] = newTag
	contents["image"] = imageMap

	return yaml.Marshal(contents)
}

func (g *gitUpdate) newTreeWithChanges(ref *github.Reference) (tree *github.Tree, err error) {
	contents, err := g.getManifestFileContents(ref)
	if err != nil {
		return nil, err
	}

	newFileContents, err := updateImageTag(contents, g.Tag)
	if err != nil {
		return nil, err
	}

	treeEntries := []*github.TreeEntry{}
	treeEntries = append(treeEntries, &github.TreeEntry{
		Path:    github.String(g.ManifestFile),
		Type:    github.String("blob"),
		Content: github.String(string(newFileContents)),
		Mode:    github.String("100644"),
	})

	tree, _, err = client.Git.CreateTree(
		ctx,
		g.RepoOwner,
		g.RepoName,
		ref.GetRef(),
		treeEntries,
	)
	return tree, err
}

func (g *gitUpdate) pushCommit(ref *github.Reference, tree *github.Tree) (err error) {
	// Get the parent commit to attach the commit to.,
	parent, _, err := client.Repositories.GetCommit(ctx, g.RepoOwner, g.RepoName, *ref.Object.SHA)
	if err != nil {
		return err
	}
	// This is not always populated, but is needed.
	parent.Commit.SHA = parent.SHA

	// Create the commit using the tree.
	date := time.Now()
	author := &github.CommitAuthor{
		Date: &date,
		// TODO: dynamic
		Name:  github.String("Caitlin Elfring"),
		Email: github.String("celfring@gmail.com"),
	}
	commit := &github.Commit{
		Author:  author,
		Message: github.String(fmt.Sprintf("[auto-release] %s:%s [ci skip]", g.DockerImage, g.Tag)),
		Tree:    tree,
		Parents: []*github.Commit{parent.Commit},
	}
	newCommit, _, err := client.Git.CreateCommit(ctx, g.RepoOwner, g.RepoName, commit)
	if err != nil {
		return err
	}

	// Attach the commit to the master branch.
	ref.Object.SHA = newCommit.SHA
	_, _, err = client.Git.UpdateRef(ctx, g.RepoOwner, g.RepoName, ref, false)
	return err
}

// createPR creates a pull request. Based on: https://godoc.org/github.com/google/go-github/github#example-PullRequestsService-Create
func (g *gitUpdate) createPR(head, base string) (url string, err error) {
	newPR := &github.NewPullRequest{
		Title: github.String(fmt.Sprintf("[auto-release] %s:%s for %s", g.DockerImage, g.Tag, base)),
		Body:  github.String(fmt.Sprintf("This PR was automatically generated by a creation of a new docker image: `%s:%s`", g.DockerImage, g.Tag)),
		Head:  github.String(head),
		Base:  github.String(base),
	}

	pr, _, err := client.PullRequests.Create(ctx, g.RepoOwner, g.RepoName, newPR)
	if err != nil {
		return "", err
	}

	return pr.GetHTMLURL(), nil
}
