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
