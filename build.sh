#!/bin/bash

if [[ ${APP_NAME} = "" ]];then
  APP_NAME=$1
fi
if [[ "${SCHEMA_NAME}" = "" ]];then
  SCHEMA_NAME=$2
fi
if [[ "${BUILD_WORK_PATH}" = "" ]]; then
  BUILD_WORK_PATH=$3
fi
if [[ "${PACKAGE_PATH}" = "" ]]; then
  PACKAGE_PATH=$4
fi

srcdir=`pwd`

echo "[info]:now at $srcdir"

echo "HIGHLIGHT__APP_NAME = ${APP_NAME}"
echo "HIGHLIGHT__SCHEMA_NAME = ${SCHEMA_NAME}"
echo "HIGHLIGHT__BUILD_WORK_PATH = ${BUILD_WORK_PATH}"
echo "HIGHLIGHT__PACKAGE_PATH = ${PACKAGE_PATH}"

export GO111MODULE=on;

make 
if [ $? != 0 ];then
  exit 1
fi

ls -l target

if [[ "${SCHEMA_NAME}" != ""  ]]; then
  TGZ_NAME=${APP_NAME}____${SCHEMA_NAME}.tgz
else
  TGZ_NAME=${APP_NAME}.tgz
fi

#脚本内置get_tgz函数可以直接引用。会把源码目录的打成压缩包。并且放到指定目录，并且生成 md5 文件
#get_tgz第一个参数是压缩包名，第二个参数是进入目录，第三个参数是要打包的目录或war包
get_tgz "${TGZ_NAME}" "${BUILD_WORK_PATH}" "target"

check_result "$?" "execute default build get_tgz function"

ls -l ${BUILD_WORK_PATH}

#信息打印
build_info "Package done!!!"