package sync

import (
	"errors"
	"fmt"

	"github.com/rdkr/gitsync/concurrency"

	"gopkg.in/src-d/go-git.v4"
)

const (
	StatusError = iota
	StatusCloned
	StatusFetched
	StatusUpToDate
)

type Status struct {
	Path   string
	Status int
	Output string
	Err    error
}

type GitSyncer func(Git, string) Status

func GitSyncHelper(g concurrency.Project) interface{} {
	return GitSync(g, func(concurrency.Project) Git {
		return GitSyncProject{g}
	})
}

func GitSync(g concurrency.Project, getGitClient func(concurrency.Project) Git) Status {

	p := getGitClient(g) // todo rename this variable

	repo, err := p.PlainOpen()

	if err == git.ErrRepositoryNotExists {

		progress, err := p.PlainClone()
		if err != nil {
			return Status{g.Location, StatusError, progress, fmt.Errorf("unable to clone repo: %v", err)}
		}
		return Status{g.Location, StatusCloned, progress, nil}

	} else if err != nil {
		return Status{g.Location, StatusError, "", fmt.Errorf("unable to open repo: %v", err)}
	}

	// TODO gracefully support bare repos
	// Get the working directory for the repository
	worktree, err := repo.Worktree()
	if err != nil {
		return Status{g.Location, StatusError, "", fmt.Errorf("unable to get worktree: %v", err)}
	}

	ref, err := repo.Head()
	if err != nil {
		return Status{g.Location, StatusError, "", fmt.Errorf("unable to get head: %v", err)}
	}

	if ref.Name() != "refs/heads/master" {
		progress, err := p.Fetch(repo)

		if err == git.NoErrAlreadyUpToDate || err == nil {
			return Status{g.Location, StatusError, progress, errors.New("not on master branch but fetched")}
		}
		return Status{g.Location, StatusError, progress, fmt.Errorf("not on master branch and: %v", err)}

	}

	progress, err := p.Pull(worktree)
	if err == nil {
		return Status{g.Location, StatusFetched, progress, nil}
	} else if err == git.NoErrAlreadyUpToDate {
		return Status{g.Location, StatusUpToDate, progress, nil}
	}
	return Status{g.Location, StatusError, progress, fmt.Errorf("unable to pull master: %v", err)}
}
