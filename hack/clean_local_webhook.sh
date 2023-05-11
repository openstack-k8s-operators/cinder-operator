#!/bin/bash
set -ex

oc delete validatingwebhookconfiguration/vcinder.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mcinder.kb.io --ignore-not-found
