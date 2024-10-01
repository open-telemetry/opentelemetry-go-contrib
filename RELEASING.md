# Release Process

This project uses the [`multimod` releaser
tool](https://github.com/open-telemetry/opentelemetry-go-build-tools/tree/main/multimod)
to manage releases. This document will walk you through how to perform a
release using this tool for this repository.

## Before releasing

### Verify OTel changes

Before releasing, it is important to verify that the changes in the upstream
go.opentelemetry.io/otel packages are compatible with the contrib repository.

Follow the following steps to verify the changes.

1. Pick the GIT SHA on the [main branch](https://github.com/open-telemetry/opentelemetry-go/commits/main) that you want to verify.
2. Run the following command to update the OTel dependencies with the GIT SHA picked in step 1.

   ```sh
   export GITSHA=<the GIT SHA you want to verify>
   make update-all-otel-deps
   make go-mod-tidy
   ```

3. Verify the changes.

   ```sh
   git diff
   ```

   This should have changed the version for all OTel modules to be the GIT SHA picked in step 1.

4. Run the lint and tests to verify that the changes are compatible with the contrib repository.

   ```sh
   make precommit
   ```

   This command should be passed without any errors.

## Start a release

First, decide which module sets will have their versions changed and what those
versions will be. If you are making a release to upgrade the upstream
go.opentelemetry.io/otel packages, all module sets will likely need to be
released.

### Breaking changes validation

You can run `make gorelease` that runs [gorelease](https://pkg.go.dev/golang.org/x/exp/cmd/gorelease)
to ensure that there are no unwanted changes done in the public API.

You can check/report problems with `gorelease` [here](https://golang.org/issues/26420).

### Create a release branch

Update the versions of the module sets you have identified in `versions.yaml`.
Commit this change to a new release branch.

### Upgrade go.opentelemetry.io/otel packages

If the upstream go.opentelemetry.io/otel project has made a release, this
project needs to be upgraded to use that release.

```sh
make sync-core COREPATH=<path to go.opentelemetry.io/otel repository>
```

This will use `multimod` to upgrade all go.opentelemetry.io/otel packages to
the latest tag found in the local copy of the project. Be sure to have this
project up to date.

Commit these changes to your release branch.

### Update module set versions

Set the version for all the module sets you have identified to be released.

```sh
make prerelease MODSET=<module set>
```

This will use `multimod` to upgrade the module's versions and create a new
"prerelease" branch for the changes. Verify the changes that were made.

```sh
git diff HEAD..prerelease_<module set>_<version>
```

Fix any issues if they exist in that prerelease branch, and when ready, merge
it into your release branch.

```sh
git merge prerelease_<module set>_<version>
```

### Update the CHANGELOG.md

Update the [Changelog](./CHANGELOG.md). Make sure only changes relevant to this
release are included and the changes are communicated in language that
non-contributors to the project can understand.

Double check there is no change missing by looking directly at the commits
since the last release tag.

```sh
git --no-pager log --pretty=oneline "<last tag>..HEAD"
```

Make sure the new released section is under the comment for released section,
like `<!-- Released section -->`, so it is protected from being overwritten in the future.

Be sure to update all the appropriate links at the bottom of the file.

Finally, commit this change to your release branch.

### Make a Pull Request

Push your release branch and create a pull request for the changes. Be sure to
include the curated changes your included in the changelog in the description.
Especially include the change PR references, as this will help show viewers of
the repository looking at these PRs that they are included in the release.

## Tag a release

Once the Pull Request with all the version changes has been approved and merged
it is time to tag the merged commit.

***IMPORTANT***: It is critical you use the same tag that you used in the
Pre-Release step! Failure to do so will leave things in a broken state. As long
as you do not change `versions.yaml` between pre-release and this step, things
should be fine.

1. For each module set that will be released, run the `add-tags` make target
   using the `<commit-hash>` of the commit on the main branch for the merged
   Pull Request.

   ```sh
   make add-tags MODSET=<module set> COMMIT=<commit hash>
   ```

   It should only be necessary to provide an explicit `COMMIT` value if the
   current `HEAD` of your working directory is not the correct commit.

2. Push tags to the upstream remote (not your fork:
   `github.com/open-telemetry/opentelemetry-go-contrib.git`). Make sure you
   push all sub-modules as well.

   ```sh
   export VERSION="<version>"
   for t in $( git tag -l | grep "$VERSION" ); do git push upstream "$t"; done
   ```

## Release

Finally create a Release on GitHub. If you are release multiple versions for
different module sets, be sure to use the stable release tag but be sure to
include each version in the release title (i.e. `Release v1.0.0/v0.25.0`). The
release body should include all the curated changes from the Changelog for this
release.

## Verify Examples

After releasing verify that examples build outside of the repository.

```sh
./verify_examples.sh
```

The script copies examples into a different directory removes any `replace` declarations in `go.mod` and builds them.
This ensures they build with the published release, not the local copy.
