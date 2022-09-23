#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

#

#https://www.dazhuanlan.com/erlv/topics/1058820
set -x
PATH2TEST=( ./pkg/... )
tmpDir=$(mktemp -d)
mergeF="${tmpDir}/merge.out"
rm -f ${mergeF}
for (( i=0; i<${#PATH2TEST[@]}; i++)) do
    ls $tmpDir
    cov_file="${tmpDir}/$i.cover"
    go test -short --race -count=1 -covermode=atomic -coverpkg=${PATH2TEST[i]} -coverprofile=${cov_file}    ${PATH2TEST[i]}
    cat $cov_file | grep -v mode: | grep -v pkg/apis/ | grep -v /pkg/generated | grep -v fake  | grep -v pb.go >> ${mergeF}
done
#merge them
echo "mode: atomic" > coverage.out
cat ${mergeF} >> coverage.out
go tool cover -func=coverage.out
rm -rf coverage.out ${tmpDir}  ${mergeF}
