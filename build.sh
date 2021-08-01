# https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04

VERSION=1.0

export CGO_ENABLED=0

# linux

GOOS=linux
GOARCH=amd64
FN=fakesni
echo ${GOOS} ${GOARCH}
env GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${FN} ./*.go
tar cfJ bin/fakesni_${VERSION}_${GOOS}_${GOARCH}.tar.xz ${FN} readme.txt

GOOS=linux
GOARCH=386
FN=fakesni
echo ${GOOS} ${GOARCH}
env GOOS=${GOOS} GOARCH=${GOARCH} GO386=softfloat go build -o ${FN} ./*.go
tar cfJ bin/fakesni_${VERSION}_${GOOS}_${GOARCH}.tar.xz ${FN} readme.txt

GOOS=linux
GOARCH=arm
FN=fakesni
echo ${GOOS} ${GOARCH}
env GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${FN} ./*.go
tar cfJ bin/fakesni_${VERSION}_${GOOS}_${GOARCH}.tar.xz ${FN} readme.txt

GOOS=linux
GOARCH=arm64
FN=fakesni
echo ${GOOS} ${GOARCH}
env GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${FN} ./*.go
tar cfJ bin/fakesni_${VERSION}_${GOOS}_${GOARCH}.tar.xz ${FN} readme.txt

rm ${FN}

# mac

GOOS=darwin
GOARCH=amd64
FN=fakesni
echo ${GOOS} ${GOARCH}
env GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${FN} ./*.go
tar cfJ bin/fakesni_${VERSION}_mac_${GOARCH}.tar.xz ${FN} readme.txt

GOOS=darwin
GOARCH=arm64
FN=fakesni
echo ${GOOS} ${GOARCH}
env GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${FN} ./*.go
tar cfJ bin/fakesni_${VERSION}_mac_${GOARCH}.tar.xz ${FN} readme.txt

rm ${FN}

# windows

GOOS=windows
GOARCH=amd64
FN=fakesni.exe
echo ${GOOS} ${GOARCH}
env GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${FN} ./*.go
zip -9 -q bin/fakesni_${VERSION}_${GOOS}_${GOARCH}.zip ${FN} readme.txt


GOOS=windows
GOARCH=386
FN=fakesni.exe
echo ${GOOS} ${GOARCH}
env GOOS=${GOOS} GOARCH=${GOARCH} GO386=softfloat go build -o ${FN} ./*.go
zip -9 -q bin/fakesni_${VERSION}_${GOOS}_${GOARCH}.zip ${FN} readme.txt

rm ${FN}
