#!/bin/bash

QMLOG=qm.log

fail() {
    echo "$1"
    exit 1
}

usage() {
    echo "Hello"
    exit 1
}

TestKubernetesCommunication() {
    echo "Kubernetes versions"
    kubectl version
}

BuildQM() {
    make
}

StartQuartermaster() {
    # Start server
    echo "Starting quartermaster"
    rm -f $QMLOG > /dev/null 2>&1
    ./quartermaster --kubeconfig=$KUBECONFIG > $QMLOG 2>&1 &
    QUARTERMASTER_PID=$!
    sleep 2
}

StopQuartermaster() {
    echo "Stopping quartermaster"
    kill $QUARTERMASTER_PID
}

if [ $# -eq 0 ] ; then
    usage
fi

while test $# -gt 0 ; do
    case "$1" in
        --kubeconfig*)
            export KUBECONFIG=`echo $1 | sed -e 's/^[^=]*=//g'`
            if [ -z "$KUBECONFIG" -o "$1" == "--kubeconfig" ] ; then
                fail "Missing kubeconfig path"
            fi
            echo "Got $KUBECONFIG"
            shift
            ;;
        *)
            usage
            ;;
    esac
done

if [ -z "$KUBECONFIG" ] ; then
    fail "Must provide kubeconfig path"
fi

TestKubernetesCommunication || fail "Unable to communicate with kubernetes cluster"
BuildQM || fail "Failed to build quartermaster"
StartQuartermaster || fail "Unable to start quartermaster"
sleep 10
StopQuartermaster || fail "Unable to stop quartermaster"
