package server

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

type Installation struct {
	RepositoryFullName string // {owner}/{repo}
	RepositoryID       int64
	InstallationID     int
	RepositoryToken    string
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
	return datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		ok, oldInst, err := getInstallation(ctx, installation.RepositoryFullName)
		if err != nil {
			return err
		}
		if !ok {
			// Not found. Generate repo token and save installation.
			installation.RepositoryToken = GenerateRepositoryToken()
			return saveInstallation(ctx, &installation)
		}
		// Found existing installation.
		// Update InstallationID if and only if InstallationID is different.
		if oldInst.InstallationID != installation.InstallationID {
			oldInst.InstallationID = installation.InstallationID
			return saveInstallation(ctx, oldInst)
		}
		return nil
	}, nil)
}

func GetOrUpdateRepoToken(ctx context.Context, repoFullName string, repoID int64, regenerate bool) (string, error) {
	var token string
	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		ok, inst, err := getInstallation(ctx, repoFullName)
		if err != nil {
			return err
		}
		if !ok {
			inst = &Installation{
				RepositoryFullName: repoFullName,
				RepositoryID:       repoID,
			}
		}
		token = inst.RepositoryToken
		if token == "" || regenerate {
			token = GenerateRepositoryToken()
			inst.RepositoryToken = token
			saveInstallation(ctx, inst)
		}
		return nil
	}, nil)
	if err != nil {
		return "", err
	}
	return token, nil
}

func saveInstallation(ctx context.Context, installation *Installation) error {
	key := newInstallationKey(ctx, installation.RepositoryFullName)
	if _, err := datastore.Put(ctx, key, installation); err != nil {
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
