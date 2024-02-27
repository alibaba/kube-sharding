#!/bin/sh
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

cd $DIR
mockery --dir=. --outpkg=testutil --output=$DIR/testutil --name=FStorage
mockery --dir=. --outpkg=testutil --output=$DIR/testutil --name=Location
