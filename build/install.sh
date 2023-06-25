#! /bin/sh

########################
## vars
########################
ARTIFACT=$1
ARTIFACT_HOME=/opt/obts/$ARTIFACT
EXEC_START=${ARTIFACT_HOME}/bin/$ARTIFACT
SERVICE_FILE=/etc/systemd/system/obts-$ARTIFACT.service
SERVICE_NAME=obts-$ARTIFACT.service
UNINSTALL=${ARTIFACT_HOME}/uninstall.sh

########################
## functions
########################
rc_check() {
  [ $1 -ne 0 ] && echo "$2 step failed" && exit 1
}

log_start() {
  echo "step $1. $2 begin..."
}

log_done() {
  echo "step $1. $2 done"
}

create_systemd_service() {
  log_start 1 "systemd: creating service file ${SERVICE_FILE}"
  $SUDO tee ${SERVICE_FILE} >/dev/null <<EOF
[Unit]
Description=$ARTIFACT for OBTS
Wants=multi-user.target
After=multi-user.target

[Install]
WantedBy=multi-user.target
Alias=${SERVICE_NAME}

[Service]
Type=simple
KillMode=process
#Restart=always
#RestartSec=3s
ExecStart=$EXEC_START

EOF
  log_done 1 "systemd: creating service file ${SERVICE_FILE}"

  log_start 2 "systemd: reload and start ${SERVICE_FILE}"

  $SUDO systemctl daemon-reload
  #$SUDO systemctl enable ${SERVICE_NAME}
  #$SUDO systemctl start ${SERVICE_NAME}

  log_done 2 "systemd: reload and start ${SERVICE_FILE}"

}

create_uninstall() {
  log_start 3 "creating uninstall script ${UNINSTALL}"
  $SUDO tee ${UNINSTALL} >/dev/null <<EOF
#!/bin/sh

# remove service
systemctl stop ${SERVICE_NAME}
systemctl disable ${SERVICE_NAME}
#systemctl reset-failed ${SERVICE_NAME}
systemctl daemon-reload
rm -f ${SERVICE_FILE}

#trap rm -f ${UNINSTALL} EXIT

rm -rf ${ARTIFACT_HOME}
EOF
  $SUDO chmod 755 ${UNINSTALL}

  log_done 3 "creating uninstall script ${UNINSTALL}"
}

########################
# main
########################
# check user
SUDO=sudo
if [ $(id -u) -eq 0 ]; then
  SUDO=
fi

# check nats
# todo
log_done 0 "check nats"

# installation
$SUDO mkdir -p $ARTIFACT_HOME

# copy bin dir to target dir
$SUDO cp -rf `ls |grep -v install.sh` $ARTIFACT_HOME
#$SUDO cp -rf ./lib $ARTIFACT_HOME

# replace sample.db to trainer.db
# the sample.db include default test user
$SUDO cp -rf $ARTIFACT_HOME/data/sample.db $ARTIFACT_HOME/data/trainer.db

# create service
create_systemd_service

# generate uninstall script
create_uninstall

echo "-------Install Finished-------"
