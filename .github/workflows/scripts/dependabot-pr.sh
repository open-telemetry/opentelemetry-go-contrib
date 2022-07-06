#!/bin/zsh -ex

git config user.name $GITHUB_ACTOR
git config user.email $GITHUB_ACTOR@users.noreply.github.com

PR_NAME=dependabot-prs/`date +'%Y-%m-%dT%H%M%S'`
git checkout -b $PR_NAME

IFS=$'\n'
requests=($(gh pr list --search "author:app/dependabot" --json number,title --template '{{range .}}{{tablerow .title}}{{end}}'))
message=""
dirs=(`find . -type f -name "go.mod" -exec dirname {} \; | sort | egrep  '^./'`)

declare -A mods

for line in $requests; do
    echo $line
    if [[ $line != build\(deps\)* ]]; then
        continue
    fi

    module=$(echo $line | cut -f 3 -d " ")
    if [[ $module == go.opentelemetry.io/contrib* ]]; then
        continue
    fi
    version=$(echo $line | cut -f 7 -d " ")

    mods[$module]=$version
    message+=$line
    message+=$'\n'
done

for module version in ${(kv)mods}; do
    topdir=`pwd`
    for dir in $dirs; do
        echo "checking $dir"
        cd $dir && if grep -q "$module " go.mod; then go get "$module"@v"$version"; fi
        cd $topdir
    done
done

make go-mod-tidy
make build

git add go.sum go.mod
git add "**/go.sum" "**/go.mod"
git commit -m "dependabot updates `date`
$message"
git push origin $PR_NAME

gh pr create --title "dependabot updates `date`" --body "$message" -l "Skip Changelog"
