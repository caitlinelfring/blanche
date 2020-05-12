package gh

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/google/go-github/v31/github"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

var ctx = context.Background()
var _client *github.Client

const (
	ErrTagMatchesCurrentTag  = "New tag matches the tag in existing manifest"
	ErrTagPrecedesCurrentTag = "New tag precedes existing tag"

	// TODO: These should be dynamic
	GitCommitAuthorName  = "Caitlin Elfring"
	GitCommitAuthorEmail = "celfring@renttherunway.com"
)

type gitUpdate struct {
	RepoOwner        string
	RepoName         string
	DockerImage      string
	Tag              string
	ManifestFile     string
	BaseBranch       string
	PullRequest      bool // setting to false will push a commit directly to the BaseBranch
	CloseOutdatedPRs bool // setting to true will auto-close all PRs that are currently opened that this update supercedes

	client       *github.Client
	ctx          context.Context
	targetBranch string
	baseRef      string
	targetRef    string
}

func CreateGithubClient(accessToken string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	_client = github.NewClient(tc)
	if octocat, _, err := _client.Octocat(ctx, "Nobody ever believes me when Im telling the truth - I guess its the curse of being a devastatingly beautiful woman"); err == nil {
		log.Println(octocat)
	}
	return _client
}

func NewGitUpdates(repoOwner, repoName, manifest, baseBranch, dockerImage, tag string, pullRequest, closeOutdatedPRs bool) *gitUpdate {
	g := gitUpdate{
		RepoOwner:        repoOwner,
		RepoName:         repoName,
		DockerImage:      dockerImage,
		Tag:              tag,
		ManifestFile:     manifest,
		BaseBranch:       baseBranch,
		PullRequest:      pullRequest,
		CloseOutdatedPRs: closeOutdatedPRs,
	}

	targetBranch := fmt.Sprintf("auto-release/%s/%s-%s", g.BaseBranch, g.DockerImage, g.Tag)
	if !g.PullRequest {
		targetBranch = g.BaseBranch
	}
	g.targetBranch = targetBranch
	g.targetRef = "refs/heads/" + g.targetBranch
	g.baseRef = "refs/heads/" + g.BaseBranch

	if _client == nil {
		_client = CreateGithubClient(os.Getenv("GITHUB_ACCESS_TOKEN"))
	}
	g.client = _client

	return &g
}

// CreateUpdates will create PRs/push commits for the given ManifestUpdate
func (g *gitUpdate) CreateUpdates() (err error) {
	ref, err := g.createRef(g.baseRef, g.targetRef)
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
		prURL, errInner := g.createPR(g.targetRef, g.baseRef)
		if errInner != nil {
			log.Println(errInner)
			return errInner
		}
		log.Printf("Opened new PR: %s | %+v", prURL, g)
		if g.CloseOutdatedPRs {
			if errInner = g.closeOutdatedPRs(prURL); err != nil {
				return errInner
			}
		}
	} else {
		log.Printf("Pushed new commit: %s | %+v", tree.Entries[0].GetURL(), g)
	}
	return
}

func (g *gitUpdate) createRef(baseBranchRef, targetBranchRef string) (ref *github.Reference, err error) {
	if ref, _, err = g.client.Git.GetRef(
		ctx,
		g.RepoOwner,
		g.RepoName,
		targetBranchRef); err == nil {
		// ref already exists
		return ref, nil
	}

	// Get base branch ref to create new ref from
	baseRef, _, err := g.client.Git.GetRef(
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
	ref, _, err = g.client.Git.CreateRef(ctx, g.RepoOwner, g.RepoName, newRef)
	return ref, err
}

func (g *gitUpdate) getManifestFileContents(ref *github.Reference) (string, error) {
	contents, _, _, err := g.client.Repositories.GetContents(
		ctx,
		g.RepoOwner,
		g.RepoName,
		g.ManifestFile,
		&github.RepositoryContentGetOptions{Ref: ref.GetRef()},
	)
	if err != nil {
		return "", err
	}
	return contents.GetContent()
}

func updateImageTag(data string, newTag string) (string, error) {
	var contents map[interface{}]interface{}
	if err := yaml.Unmarshal([]byte(data), &contents); err != nil {
		return "", err
	}

	imageMap := contents["image"].(map[interface{}]interface{})
	currentTag := imageMap["tag"].(string)
	if currentTag == newTag {
		return "", errors.New(ErrTagMatchesCurrentTag)
	}

	// The result will be 0 if a == b, -1 if a < b, or +1 if a > b.
	if semver.Compare(currentTag, newTag) >= 0 {
		return "", errors.New(ErrTagPrecedesCurrentTag)
	}

	imageMap["tag"] = newTag
	contents["image"] = imageMap

	b, err := yaml.Marshal(contents)
	if err != nil {
		return "", err
	}
	return string(b), nil
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
		Content: github.String(newFileContents),
		Mode:    github.String("100644"),
	})

	tree, _, err = g.client.Git.CreateTree(
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
	parent, _, err := g.client.Repositories.GetCommit(ctx, g.RepoOwner, g.RepoName, ref.Object.GetSHA())
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
		Name:  github.String(GitCommitAuthorName),
		Email: github.String(GitCommitAuthorEmail),
	}
	commit := &github.Commit{
		Author:  author,
		Message: github.String(fmt.Sprintf("[auto-release] %s:%s [ci skip]", g.DockerImage, g.Tag)),
		Tree:    tree,
		Parents: []*github.Commit{parent.Commit},
	}
	newCommit, _, err := g.client.Git.CreateCommit(ctx, g.RepoOwner, g.RepoName, commit)
	if err != nil {
		return err
	}

	// Attach the commit to the master branch.
	ref.Object.SHA = newCommit.SHA
	_, _, err = g.client.Git.UpdateRef(ctx, g.RepoOwner, g.RepoName, ref, false)
	return err
}

// createPR creates a pull request. Based on: https://godoc.org/github.com/google/go-github/github#example-PullRequestsService-Create
func (g *gitUpdate) createPR(head, base string) (url string, err error) {
	newPR := &github.NewPullRequest{
		Title: github.String(fmt.Sprintf("[auto-release] %s:%s for %s", g.DockerImage, g.Tag, g.BaseBranch)),
		Body:  github.String(fmt.Sprintf("This PR was automatically generated by a creation of a new docker image: %s:%s", g.DockerImage, g.Tag)),
		Head:  github.String(head),
		Base:  github.String(base),
	}

	pr, _, err := g.client.PullRequests.Create(ctx, g.RepoOwner, g.RepoName, newPR)
	if err != nil {
		return "", err
	}

	return pr.GetHTMLURL(), nil
}

func (g *gitUpdate) closeOutdatedPRs(supersededPRURL string) error {
	prsToClose, err := g.getOutdatedPRs()
	if err != nil {
		return err
	}
	for _, pr := range prsToClose {
		// This will return only the error of the first failure,
		// and won't attempt to close any PRs after the first failure

		if _, _, err := g.client.PullRequests.CreateComment(ctx, g.RepoOwner, g.RepoName, pr.GetNumber(), &github.PullRequestComment{
			Body: github.String("Superseded by " + supersededPRURL),
		}); err != nil {
			// Not failing the whole loop because the PR comment failed, we still want it closed
			log.Printf("Failed to close pr %s: %s", pr.GetHTMLURL(), err)
		}

		pr.State = github.String("closed")
		if _, _, err := g.client.PullRequests.Edit(ctx, g.RepoOwner, g.RepoName, pr.GetNumber(), pr); err != nil {
			return err
		}
	}
	return nil
}

func (g *gitUpdate) getOutdatedPRs() (prs []*github.PullRequest, err error) {
	openPRs, _, err := g.client.PullRequests.List(ctx, g.RepoOwner, g.RepoName, &github.PullRequestListOptions{State: "opened"})
	if err != nil {
		return
	}

	for _, pr := range openPRs {
		if isOlderVersionBumpPR(g.DockerImage, g.Tag, pr) {
			prs = append(prs, pr)
		}
	}
	return
}

func isOlderVersionBumpPR(dockerImage, dockerTag string, pr *github.PullRequest) bool {
	re := regexp.MustCompile(`\[auto-release\] (.*):(.*) for .*`)
	if title := re.FindStringSubmatch(pr.GetTitle()); title != nil {
		prDockerImage := title[1]
		prDockerTag := title[2]
		return dockerImage == prDockerImage && isNewerVersion(dockerTag, prDockerTag)
	}
	return false
}

func isNewerVersion(a, b string) bool {
	return semver.IsValid(a) && semver.IsValid(b) &&
		// The result will be 0 if a == b, -1 if a < b, or +1 if a > b
		semver.Compare(a, b) > 0
}
