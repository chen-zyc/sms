#! /bin/bash

cd "$(dirname $0)"
DIR=`pwd`

if [ ! -d "${WORKSPACE}/bin" ]; then
    mkdir ${WORKSPACE}/bin
fi

WORKSPACE="$DIR/../../.."
export GOPATH=$WORKSPACE:$WORKSPACE/vendor
export GOBIN=$WORKSPACE/bin
echo "GOPATH = " $GOPATH
echo "GOBIN  = " $GOBIN

cd ${WORKSPACE}
go install -v -ldflags "-s -w" sms/example