#!/usr/bin/env bash
set -e

LOCATION=$(realpath `dirname -- $BASH_SOURCE[0]`)

if [ $# -eq 0 ]; then
	echo "Error, missing arguments: $0 <PR#> [<remote_name> [<local_branch>]]"
	exit 1
fi

PR=$1
REMOTE=${2:-upstream}
BRANCH=${3:-pr$1}

$LOCATION/cleandeps.sh

# Get the code
echo Fetching PR $PR on $REMOTE/$BRANCH
git fetch $REMOTE pull/$PR/head:$BRANCH
git checkout $BRANCH

# Get dependencies
echo Checking PR dependecies
deps=`$LOCATION/showdeps.py $PR`
if [[ -n "$deps" ]]; then
    echo Setting dependencies: $deps
    $LOCATION/setdeps.py $deps
else
    echo PR has no dependencies
fi
