#!/bin/bash

set -e

PN="${BASH_SOURCE[0]##*/}"
PD="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
projorg="pensando"
projname="gogstash"

renice 15 $$
cd "${PD}/.."

gobuilder version -c ">=0.1.5" &>/dev/null || go get -u -v "github.com/tsaikd/gobuilder"
gobuilder --all --check --test

if [ "${GITHUB_TOKEN}" ] ; then
	echo "[$(date -Iseconds)] ${projname} release on github"
	if ! type github-release &>/dev/null ; then
		go get -v "github.com/aktau/github-release"
	fi
	rev="$(git rev-parse HEAD)"
	version="$(./${projname} version -n)"
	github-release release \
		--user "${projorg}" \
		--repo "${projname}" \
		--tag "${version}" \
		--target "${rev}" \
		--description "curl 'https://github.com/${projorg}/${projname}/releases/download/${version}/${projname}-$(uname -s)-$(uname -m)' -SLo ${projname} && chmod +x ${projname}"
	github-release upload \
		--user "${projorg}" \
		--repo "${projname}" \
		--tag "${version}" \
		--name "${projname}-$(uname -s)-$(uname -m)" \
		--file "${projname}"
fi
