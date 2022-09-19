#!/usr/bin/env bash

function crc_ssh {
    SSH_PARAMS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.crc/machines/crc/id_ecdsa"
    SSH_REMOTE="core@`crc ip`"
    ssh $SSH_PARAMS $SSH_REMOTE "$@"
}


function crc_login {
    echo Logging in
    oc login -u kubeadmin -p 12345678 https://api.crc.testing:6443
}
