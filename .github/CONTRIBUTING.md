# Contributing

TBD for other type of contribution.

## Release Pull Request (Release Process)
You can create a Pull Request to release a new version.

1. Update [CHANGELOG.md](../CHANGELOG.md). Copy unreleased section and create a new version section.
2. Create a Pull Request.
3. Attach a label [`bump:patch`, `bump:minor`, or `bump:major`]. reviewdog uses [haya14busa/action-bumpr](https://github.com/haya14busa/action-bumpr).
4. [The release workflow](./workflows/release.yml) automatically tag a
   new version depending on the label and create a new release on merging the
   Pull Request.
5. Check the release workflow and verify the release. [![release](https://github.com/reviewdog/reviewdog/workflows/release/badge.svg)](https://github.com/reviewdog/reviewdog/actions?query=workflow%3Arelease)

### Hot Fix Release

If you find a severe issue and want to release a fix ASAP without unreleased features,
you may want to follow the hot-fix release process instead of the usual release flow.

It's mostly similar to the usual release flow, but you need to change the target branch of release pull request.

1. Create a fix and merge it into master.
2. Create a release branch (`release-v{major}.{minor}`) based on the latest semver release `v{major}.{minor}.{patch}` if there is no exisisting release branch.
  - You need write permission to reviewdog repository to push the release branch.
3. Create **another release request branch** (any name is fine) and update [CHANGELOG.md](../CHANGELOG.md).
4. Create a pull request from the release request branch to **the release branch** (`release-v{major}.{minor}`) instead of master.
5. Attach `bump:patch` label.
6. [The release workflow](./workflows/release.yml) automatically tag a
   new hot-fix version and create a new release on merging the Pull Request.
7. Check the release workflow and verify the release. [![release](https://github.com/reviewdog/reviewdog/workflows/release/badge.svg)](https://github.com/reviewdog/reviewdog/actions?query=workflow%3Arelease)
8. Merge the release branch into master to include the CHANGELOG.md update.
