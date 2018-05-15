package server

import (
	"context"

	"google.golang.org/appengine/datastore"
)

type Installation struct {
	RepositoryID       int64
	InstallationID     int
	RepositoryFullName string
}

func newInstallationKey(ctx context.Context, repositoryName string) *datastore.Key {
	kind := "Installation"
	return datastore.NewKey(ctx, kind, repositoryName, 0, nil)
}

type CheckSuiteEvent struct {
	Action     string `json:"action,omitempty"`
	Repository struct {
		ID       int64  `json:"id,omitempty"`
		FullName string `json:"full_name,omitempty"`
	} `json:"repository,omitempty"`
	Installation struct {
		ID int `json:"id,omitempty"`
	} `json:"installation,omitempty"`
}

func SaveInstallationFromCheckSuite(ctx context.Context, c CheckSuiteEvent) error {
	installation := Installation{
		InstallationID:     c.Installation.ID,
		RepositoryFullName: c.Repository.FullName,
		RepositoryID:       c.Repository.ID,
	}
	return saveInstallation(ctx, installation)
}

func saveInstallation(ctx context.Context, installation Installation) error {
	key := newInstallationKey(ctx, installation.RepositoryFullName)
	if _, err := datastore.Put(ctx, key, &installation); err != nil {
		return err
	}
	return nil
}

func getInstallation(ctx context.Context, repositoryFullName string) (bool, *Installation, error) {
	i := new(Installation)
	key := newInstallationKey(ctx, repositoryFullName)
	if err := datastore.Get(ctx, key, i); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, i, nil
}
