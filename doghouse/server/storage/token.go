package storage

import (
	"context"
	"fmt"
	"github.com/philippgille/gokv"

	"cloud.google.com/go/datastore"
)

// GitHubRepositoryToken represents token data for authenticating reviewdog CLI
// to the target repository.
type GitHubRepositoryToken struct {
	Token string
	// https://github/<RepositoryOwner>/<RepositoryName>.
	RepositoryOwner string
	RepositoryName  string
	RepositoryID    int64
}

// GitHubRepositoryTokenStore represents GitHubRepositoryToken storage interface.
type GitHubRepositoryTokenStore interface {
	// Put upserts GitHubRepositoryToken entity.
	Put(ctx context.Context, token *GitHubRepositoryToken) error
	// Get get GitHubRepositoryToken entity by owner and repo name.
	// - If the entity is found, return token with ok is true.
	// - If the entity is not found, ok is false.
	// - If error occurs, it returns err.
	Get(ctx context.Context, owner, repo string) (ok bool, token *GitHubRepositoryToken, err error)
}

// GoogleGitHubRepoTokenDatastore is kvStore of GitHubRepositoryToken by Datastore of
// Google Appengine.
type GoogleGitHubRepoTokenDatastore struct{}

type LocalKVGitHubRepoTokenStore struct {
	KvStore gokv.Store
}

func (l LocalKVGitHubRepoTokenStore) Put(ctx context.Context, token *GitHubRepositoryToken) error {

	err := l.KvStore.Set(token.RepositoryOwner+"/"+token.RepositoryName, token)

	if err != nil {
		return err
	}

	return nil
}

func (l LocalKVGitHubRepoTokenStore) Get(ctx context.Context, owner, repo string) (ok bool, token *GitHubRepositoryToken, err error) {

	token = &GitHubRepositoryToken{}

	get, err := l.KvStore.Get(owner+"/"+repo, token)
	if err != nil {
		return false, nil, err
	}

	return get, token, nil
}

func (g *GoogleGitHubRepoTokenDatastore) newKey(owner, repo string) *datastore.Key {
	kind := "GitHubRepositoryToken"
	return datastore.NameKey(kind, fmt.Sprintf("%s/%s", owner, repo), nil)
}

// Put upserts GitHubRepositoryToken.
func (g *GoogleGitHubRepoTokenDatastore) Put(ctx context.Context, token *GitHubRepositoryToken) error {
	key := g.newKey(token.RepositoryOwner, token.RepositoryName)
	d, err := datastoreClient(ctx)
	if err != nil {
		return err
	}
	_, err = d.Put(ctx, key, token)
	return err
}

func (g *GoogleGitHubRepoTokenDatastore) Get(ctx context.Context, owner, repo string) (ok bool, token *GitHubRepositoryToken, err error) {
	key := g.newKey(owner, repo)
	token = new(GitHubRepositoryToken)
	d, err := datastoreClient(ctx)
	if err != nil {
		return false, nil, err
	}
	if err := d.Get(ctx, key, token); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, token, nil
}
