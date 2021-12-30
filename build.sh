# https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04

VERSION=v1.3

export CGO_ENABLED=0
export GO386=softfloat

BUILD_DIR=../fakesni_${VERSION}
mkdir ${BUILD_DIR}

go_build() {
  go build -o ${FN} -ldflags="-X 'main.Version=${VERSION}'" ./*.go
}

# linux

FN=fakesni
export GOOS=linux

export GOARCH=amd64
echo ${GOOS} ${GOARCH}
go_build
tar -czf ${BUILD_DIR}/fakesni_${VERSION}_${GOOS}_${GOARCH}.tar.gz ${FN} readme.txt

export GOARCH=386
echo ${GOOS} ${GOARCH}
go_build
tar -czf ${BUILD_DIR}/fakesni_${VERSION}_${GOOS}_${GOARCH}.tar.gz ${FN} readme.txt

export GOARCH=arm
echo ${GOOS} ${GOARCH}
go_build
tar -czf ${BUILD_DIR}/fakesni_${VERSION}_${GOOS}_${GOARCH}.tar.gz ${FN} readme.txt

rm ${FN}

# mac

FN=fakesni
export GOOS=darwin

export GOARCH=amd64
echo ${GOOS} ${GOARCH}
go_build
tar -czf ${BUILD_DIR}/fakesni_${VERSION}_mac_intel.tar.gz ${FN} readme.txt

export GOARCH=arm64
echo ${GOOS} ${GOARCH}
go_build
tar -czf ${BUILD_DIR}/fakesni_${VERSION}_mac_${GOARCH}.tar.gz ${FN} readme.txt

rm ${FN}

# windows

FN=fakesni.exe
export GOOS=windows

export GOARCH=amd64
echo ${GOOS} ${GOARCH}
go_build
zip -9 -q ${BUILD_DIR}/fakesni_${VERSION}_${GOOS}_${GOARCH}.zip ${FN} readme.txt

rm ${FN}
