package storage

import (
	"context"

	"google.golang.org/appengine/datastore"
)

// GitHubInstallation represents GitHub Apps Installation data.
// Installation is per org or user account, not repository.
type GitHubInstallation struct {
	InstallationID int64  // https://github.com/settings/installations/<InstallationID>
	AccountName    string // https://github/<AccountName>. Org or user account.
	AccountID      int64  // GitHub ID of <AccountName>.
}

// GitHubInstallationStore represents GitHubInstallation storage interface.
type GitHubInstallationStore interface {
	// Put upserts GitHub InstallationID entity. If InstallationID is not
	// updated, the whole entity won't be saved.
	Put(ctx context.Context, inst *GitHubInstallation) error
	// Get get GitHubInstallation entity by account name.
	// - If the entity is found, return inst with ok is true.
	// - If the entity is not found, ok is false.
	// - If error occurs, it returns err.
	Get(ctx context.Context, accountName string) (ok bool, inst *GitHubInstallation, err error)
}

// GitHubInstallationDatastore is store of GitHubInstallation by Datastore of
// Google Appengine.
type GitHubInstallationDatastore struct{}

func (g *GitHubInstallationDatastore) newKey(ctx context.Context, accountName string) *datastore.Key {
	const kind = "GitHubInstallation"
	return datastore.NewKey(ctx, kind, accountName, 0, nil)
}

// Put save GitHubInstallation. It reduces datastore write call as much as possible.
func (g *GitHubInstallationDatastore) Put(ctx context.Context, inst *GitHubInstallation) error {
	return datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		ok, foundInst, err := g.Get(ctx, inst.AccountName)
		if err != nil {
			return err
		}
		// Insert if not found or installation ID is different.
		if !ok || foundInst.InstallationID != inst.InstallationID {
			return g.put(ctx, inst)
		}
		return nil // Do nothing.
	}, nil)
}

func (g *GitHubInstallationDatastore) put(ctx context.Context, inst *GitHubInstallation) error {
	key := g.newKey(ctx, inst.AccountName)
	_, err := datastore.Put(ctx, key, inst)
	return err
}

func (g *GitHubInstallationDatastore) Get(ctx context.Context, accountName string) (ok bool, inst *GitHubInstallation, err error) {
	key := g.newKey(ctx, accountName)
	inst = new(GitHubInstallation)
	if err := datastore.Get(ctx, key, inst); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, inst, nil
}
