# Contributing to opentelemetry-go-contrib

The Go special interest group (SIG) meets regularly. See the
OpenTelemetry
[community](https://github.com/open-telemetry/community#golang-sdk)
repo for information on this and other language SIGs.

See the [public meeting
notes](https://docs.google.com/document/d/1A63zSWX0x2CyCK_LoNhmQC4rqhLpYXJzXbEPDUQ2n6w/edit#heading=h.9tngw7jdwd6b)
for a summary description of past meetings. To request edit access,
join the meeting or get in touch on
[Gitter](https://gitter.im/open-telemetry/opentelemetry-go).

## Development

There are some generated files checked into the repo. To make sure
that the generated files are up-to-date, run `make` (or `make
precommit` - the `precommit` target is the default).

The `precommit` target also fixes the formatting of the code and
checks the status of the go module files.

If after running `make precommit` the output of `git status` contains
`nothing to commit, working tree clean` then it means that everything
is up-to-date and properly formatted.

## Pull Requests

### How to Send Pull Requests

Everyone is welcome to contribute code to `opentelemetry-go-contrib` via
GitHub pull requests (PRs).

To create a new PR, fork the project in GitHub and clone the upstream
repo:

```sh
$ git clone https://github.com/open-telemetry/opentelemetry-go
```
This would put the project in the `opentelemetry-go-contrib` directory in
current working directory.

Enter the newly created directory and add your fork as a new remote:

```sh
$ git remote add <YOUR_FORK> git@github.com:<YOUR_GITHUB_USERNAME>/opentelemetry-go
```

Check out a new branch, make modifications, run linters and tests, and
push the branch to your fork:

```sh
$ git checkout -b <YOUR_BRANCH_NAME>
# edit files
$ make precommit
$ git add -p
$ git commit
$ git push <YOUR_FORK> <YOUR_BRANCH_NAME>
```

Open a pull request against the main `opentelemetry-go-contrib` repo.

### How to Receive Comments

* If the PR is not ready for review, please put `[WIP]` in the title,
  tag it as `work-in-progress`, or mark it as
  [`draft`](https://github.blog/2019-02-14-introducing-draft-pull-requests/).
* Make sure CLA is signed and CI is clear.

### How to Get PRs Merged

A PR is considered to be **ready to merge** when:

* It has received two approvals from Collaborators/Maintainers (at
  different companies).
* Major feedbacks are resolved.
* It has been open for review for at least one working day. This gives
  people reasonable time to review.
* Trivial change (typo, cosmetic, doc, etc.) doesn't have to wait for
  one day.
* Urgent fix can take exception as long as it has been actively
  communicated.

Any Collaborator/Maintainer can merge the PR once it is **ready to
merge**.

## Style Guide

* Make sure to run `make precommit` - this will find and fix the code
  formatting.

## Adding a new Contrib package

To add a new contrib package follow an existing one. An empty Sample plugin
provides base structure with an example and a test. Each contrib package 
should be its own module. A contrib package may contain more than one go package.

### Folder Structure
- plugins/\<plugin-package>  (**Common**)
- plugins/\<plugin-package>/trace (**specific to trace**)
- plugins/\<plugin-package>/metrics (**specific to metrics**)

#### Example
- plugins/gorm/trace
- plugins/kafka/metrics

## Approvers and Maintainers

Approvers:

- [Krzesimir Nowak](https://github.com/krnowak), Kinvolk
- [Liz Fong-Jones](https://github.com/lizthegrey), Honeycomb
- [Gustavo Silva Paiva](https://github.com/paivagustavo), Stilingue
- [Ted Young](https://github.com/tedsuo), LightStep
- [Anthony Mirabella](https://github.com/Aneurysm9), Centene

Maintainers:

- [Josh MacDonald](https://github.com/jmacd), LightStep
- [Tyler Yahn](https://github.com/MrAlias), New Relic

### Become an Approver or a Maintainer

See the [community membership document in OpenTelemetry community
repo](https://github.com/open-telemetry/community/blob/master/community-membership.md).
