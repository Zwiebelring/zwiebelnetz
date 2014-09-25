#!/bin/sh

#
# Copyright (c) 2014
#   Dario Brandes
#   Thies Johannsen
#   Paul Kröger
#   Sergej Mann
#   Roman Naumann
#   Sebastian Thobe
# All rights reserved.
#
# Redistribution and use in source and binary forms, with or without modification,
# are permitted provided that the following conditions are met:
#
# 1. Redistributions of source code must retain the above copyright notice, this
#    list of conditions and the following disclaimer.
#
# 2. Redistributions in binary form must reproduce the above copyright notice,
#    this list of conditions and the following disclaimer in the documentation
#    and/or other materials provided with the distribution.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
# ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
# WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
# DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
# ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
# (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
# LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
# ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
# (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
# SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
#

set -e

NAME=$1
PW=$2

if [ -z "$NAME" ]; then
  logger -s "ERROR: no username supplied to init_user.sh"
  exit 1
fi
if [ -z "$PW" ]; then
  logger -s "ERROR: no password supplied to init_user.sh"
  exit 1
fi

if [ $(id -u) != 0 ] ; then
  logger -s "ERROR: call init_user.sh as root!"
  exit 1
fi

set -u

logger -s "INFO: fixing permissions on start scripts"
chmod ugo+x /home/pi/zwiebelnetz/wizard/start_syncerd.sh
chmod ugo+x /home/pi/zwiebelnetz/wizard/start_server.sh
chmod ugo+x /home/pi/zwiebelnetz/wizard/change_hostname.sh
chmod uog+x /home/pi/zwiebelnetz/wizard/expand_fs.sh

logger -s "INFO: adding ssn user"

adduser --gecos "" --disabled-password ssn_${NAME}

logger -s "INFO: writing torrc"

cat <<EOF > /etc/tor/torrc
HiddenServiceDir /var/lib/tor/ssn_${NAME}/
HiddenServicePort 3141 127.0.0.1:3141
EOF

mkdir -p /var/lib/tor
chown debian-tor:debian-tor /var/lib/tor

service tor restart

# wait for private key and onion-address to be generated
while [ ! -f "/var/lib/tor/ssn_${NAME}/private_key" -o ! -f "/var/lib/tor/ssn_${NAME}/hostname" ]; do
  logger -s "INFO: Tor still generating private key..."
  sleep 1s
done


cp /var/lib/tor/ssn_${NAME}/private_key /home/ssn_${NAME}/
chown ssn_${NAME}:ssn_${NAME}  /home/ssn_${NAME}/private_key

onion_addr=$(cat /var/lib/tor/ssn_${NAME}/hostname)

logger -s "initializing database as user ssn_${NAME}..."
su -l ssn_${NAME} <<EOF
/home/pi/zwiebelnetz/test/client_main -cmd init -nickname ${NAME} -onion ${onion_addr} -password ${PW} -key \$HOME/private_key
rm \$HOME/private_key
EOF

logger -s "...finished initializing database, exiting..."

exit 0
