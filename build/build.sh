#!/bin/bash
show_version(){
  echo "v1.0.0"
}

usage() {
  echo ""
  echo "Usage:"
  echo "  $0 [param1,param2,..]"
  echo "parameters "
  echo "--version               : show version"
  echo "-d                      : build in docker container"
  echo "-h                      : show this msg"
  echo "-p|--pack=true/false    : default true"
  echo "-o|--output=.           : special output directory"
  echo "-v|--verbose            : display verbose log "
  echo "-t|--type=server/client : build type as server|client,default server"
  echo ""
}

build_in_docker(){
  echo "pull docker compiler"
  sudo docker pull golang:1.20
  echo "pull docker compiler done"

  echo "use docker container compile"
  cmd="bash ./build.sh -o $OUTPUT_DIR"
  if [ x"$SHOW_VERBOSE" = x"true" ];then
    cmd="$cmd -v"
  fi

  sudo docker run --rm -v $CODE_DIR:$CODE_DIR \
    -v $CODE_DIR/../intf:$CODE_DIR/../intf \
    -w $CODE_DIR \
    -e GOPROXY=https://proxy.golang.com.cn,direct \
    -v /etc/localtime:/etc/localtime \
    -v $OUTPUT_DIR:$OUTPUT_DIR \
    golang:1.20 \
    bash -c "git config --global --add safe.directory '*' && cd build && $cmd "

  [ $? != 0 ] && echo "build failed" && exit 1
  echo "use docker container compile done"
}

##############################################################
# args
##############################################################
# layout vars
CURR_DIR=$(pwd)
SCRIPT_POS=$(readlink -f $0)
SCRIPT_DIR=$(dirname "$SCRIPT_POS")
CODE_DIR=$SCRIPT_DIR/../
DIST_DIR=$CODE_DIR/dist

BUILD_ARTIFACT_SERVER=jt808server
BUILD_ARTIFACT_CLIENT=jt808client
BUILD_ARTIFACT=$BUILD_ARTIFACT_SERVER

# parameters
PACK=true
OUTPUT_DIR=$(pwd)
USE_DOCKER_BUILD=false

# getopts
until [ $# -eq 0 ]; do
  case "$1" in
    --version) show_version; exit 0;;
    -h|--help) usage; exit 0;;
    -p|--pack) PACK=$2; shift 2;;
    --pack=*) PACK=`echo $1|awk -F= '{print $2}'`; shift 1;;
    -d) USE_DOCKER_BUILD=true;shift 1;;
    -o|--output) OUTPUT_DIR=$2; shift 2;;
    --output=*) OUTPUT_DIR=`echo $1|awk -F= '{print $2}'`; shift  1;;
    -v|--verbose) SHOW_VERBOSE=true; shift 1;;
    -t|--type)
      case "$2" in
        server) BUILD_ARTIFACT=$BUILD_ARTIFACT_SERVER;;
        client) BUILD_ARTIFACT=$BUILD_ARTIFACT_CLIENT;;
        *) echo " unknown params $1 $2 " && usage && exit 1;;
      esac
      shift  2;;
    --type=*)
      case `echo $1|awk -F= '{print $2}'` in
        server) BUILD_ARTIFACT=$BUILD_ARTIFACT_SERVER;;
        client) BUILD_ARTIFACT=$BUILD_ARTIFACT_CLIENT;;
        *) echo " unknown params $1 " && usage && exit 1;;
      esac
      shift  1;;
    *) echo " unknown params $1" && usage && exit 1;;
  esac
done

if [ x"$USE_DOCKER_BUILD" = x"true" ];then
  build_in_docker
  exit 0
fi

##############################################################
# check build tools
##############################################################
git version >/dev/null 2>&1
[ $? -ne 0 ] && echo "please install git first" && exit 1

go version >/dev/null 2>&1
[ $? -ne 0 ] && echo "please install go first" && exit 1

##############################################################
# variables used when building
##############################################################


# package vars
BUILD_PRODUCT=jt808
PRODUCT_VERSION=1.0
ARTIFACT_VERSION=$(cat "$CODE_DIR"/VERSION)
BUILD_OS=$(go env GOOS)
BUILD_ARCH=$(go env GOARCH)
BUILD_TIME=$(date +%Y%m%d%H%M%S)

#COMPILE_TARGET=$SCRIPT_DIR/$BUILD_ARTIFACT
PACKAGE_NAME=$BUILD_PRODUCT-$PRODUCT_VERSION-$BUILD_ARTIFACT-$ARTIFACT_VERSION-$BUILD_OS-$BUILD_ARCH-$BUILD_TIME.sh
# e.g.: jt808-1.0-jt808server-1.2.0-windows-amd64-202304271123.sh

GO_VERSION=$(go version | awk '{print $3}')
GIT_HASH=$(git show -s --format=%H)
PACKER=$SCRIPT_DIR/makeself.sh
chmod +x $SCRIPT_DIR/*sh

##############################
## functions
##############################


validate_args() {
  # check cmd line args if any
  echo "todo"
}

make_dirs() {
  $SUDO rm -rf "$DIST_DIR"/bin
  mkdir -p "$DIST_DIR"/bin
  mkdir -p "$DIST_DIR"/etc
}

compile() {
  echo "=== ready to compile ==="
  # prepare vars to be injected into code
  goVersion="main.goVersion=$GO_VERSION"
  gitHash="main.gitHash=$GIT_HASH"
  buildTime="main.buildTime=$BUILD_TIME"
  version="main.version=$ARTIFACT_VERSION"
  #echo $goVersion $gitHash $buildTime $mVersion
  ldflags="-X "$goVersion" -X "$gitHash" -X "$buildTime" -X "$version""
  #echo "$ldflags"

  buildArgs=
  if [ x"$SHOW_VERBOSE" = x"true" ];then
    buildArgs="-v"
  fi

  cd "$CODE_DIR"
  go mod tidy
  [ $? -ne 0 ] && echo "go mod tidy failed" && exit 1

  if [ x"$BUILD_ARTIFACT" = x"$BUILD_ARTIFACT_SERVER"  ];then
    go build $buildArgs -ldflags "$ldflags" -o "$SCRIPT_DIR"/"$BUILD_ARTIFACT" .
  else
    go build $buildArgs -ldflags "$ldflags" -o "$SCRIPT_DIR"/"$BUILD_ARTIFACT" test/client/main.go
  fi

  [ $? -ne 0 ] && echo "build failed" && exit 1

  echo "=== compilation done ==="
}

pack() {
  echo "=== ready to pack ==="

  cd "$SCRIPT_DIR"

  strip "$BUILD_ARTIFACT"

  if [ x"$BUILD_ARTIFACT" = x"$BUILD_ARTIFACT_CLIENT"  ];then
    cp -rf $CODE_DIR/test/client/configs/* $DIST_DIR/etc/.
  fi
  cp -rf "$BUILD_ARTIFACT" "$DIST_DIR"/bin
  cp -rf install.sh "$DIST_DIR"
  chmod a+x $DIST_DIR/install.sh

  [ ! -d $OUTPUT_DIR ] && mkdir -p $OUTPUT_DIR

  packArgs="--quiet"
  if [ x"$SHOW_VERBOSE" = x"true" ];then
    packArgs=
  fi

  $PACKER $packArgs "$DIST_DIR" $OUTPUT_DIR/"$PACKAGE_NAME" "obts $BUILD_ARTIFACT installer" ./install.sh $BUILD_ARTIFACT

  echo "packing output to: $OUTPUT_DIR/$PACKAGE_NAME"
  echo "=== packing done ==="
}

###########################################
# main
###########################################

#show_version
#validate_args
make_dirs
compile

if [ x"${PACK}" = x"true" ] ; then
  pack
fi

cd "$CURR_DIR"

echo "=== build successfully ==="
