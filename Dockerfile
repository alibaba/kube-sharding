# This dockerfile uses the go image
# VERSION 2 - EDITION 1
# Author: @author
# Command format: Instruction [arguments / command] ..

# Base image to use, this must be set as the first line
FROM reg.docker.alibaba-inc.com/hippo/hippo_alios7u2_base:1.8

ARG APP_NAME
ENV BUILD_APP_NAME=$APP_NAME
ENV APP_DIR /home/admin/c2

RUN ls -h \
    && mkdir -p ${APP_DIR}/{bin,conf,logs} \
    && chmod 777 -R ${APP_DIR}/logs  

COPY target/c2 ${APP_DIR}/bin/ 
COPY target/carbon-proxy ${APP_DIR}/bin/ 
COPY target/assets ${APP_DIR}/ 

WORKDIR ${APP_DIR}

VOLUME ${APP_DIR}/logs
VOLUME ${APP_DIR}/data
