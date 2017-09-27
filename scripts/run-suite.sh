#!/usr/bin/env bash

# Ensure we have a test name
if [ "$#" -ne 1 ]; then
	echo "Usage: $0	TESTNAME"
	exit 1
fi

#############################################
# DEFAULT SETTINGS
# 
# Do not modify to run differing tests - all tests inherit these defaults.
# Instead, explicitly set it when calling run-tests.sh for each test.
# 
# If you're not using ZFS, unset PG_SET_ARC and MONGO_SET_ARC.

# Git config
export GIT="/usr/local/bin/git"

# SSH command to log into the database server
export SSH_CONN="/usr/bin/ssh 10.1.0.42"

# Mongo config
export MONGO_CONN="mongodb://10.1.0.42/testdb" # MongoDB connection string
export MONGO_SHELL="/usr/local/bin/mongo $MONGO_CONN"
export MONGO_SET_ARC="$SSH_CONN \"sysctl vfs.zfs.arc_max=20615843020\"" # 20GB ARC

# Postgres config
export PG_SHELL="/usr/local/bin/psql -h 10.1.0.42 -p 5432 -U benchmark testdb" #DB name must match value in PG_CONN
export PG_CONN="postgres://benchmark:password@10.1.0.42:6432/testdb?sslmode=disable;binary_parameters=yes"
export PG_SET_ARC="$SSH_CONN \"sysctl vfs.zfs.arc_max=8589934592\"" # 8.5GB ARC

# Benchmark config
export BENCH_TOOL="./mpjbt"
export WORKERS=30
export PADDING=""

# Number of operations to perform.
# 
# Updates are slower, so do less.
export OPS_COUNT=1000000
export UPDATE_COUNT=250000

# Service control
export MONGO_ON="$SSH_CONN \"jexec mongo /usr/sbin/service mongod onestart\""
export MONGO_OFF="$SSH_CONN \"jexec mongo /usr/sbin/service mongod onestop\""
export PG_ON="$SSH_CONN -t \"jexec postgres /usr/sbin/service postgresql onestart\""
export PG_OFF="$SSH_CONN -t \"jexec postgres /usr/sbin/service postgresql onestop\""

#############################################
# Tests to run

WORKERS=30  ./run-tests.sh $1-workers-30
WORKERS=100 ./run-tests.sh $1-workers-100
WORKERS=300 ./run-tests.sh $1-workers-300

OPS_COUNT=100000 UPDATE_COUNT=50000 PADDING="1k" WORKERS=30  ./run-tests.sh $1-workers-30-padding-1k
OPS_COUNT=100000 UPDATE_COUNT=50000 PADDING="1k" WORKERS=100 ./run-tests.sh $1-workers-100-padding-1k
PADDING="1k" WORKERS=300 ./run-tests.sh $1-workers-300-padding-1k

OPS_COUNT=10000 UPDATE_COUNT=5000 PADDING="1mb" WORKERS=30  ./run-tests.sh $1-workers-30-padding-1mb
OPS_COUNT=10000 UPDATE_COUNT=5000 PADDING="1mb" WORKERS=100 ./run-tests.sh $1-workers-100-padding-1mb
PADDING="1mb" WORKERS=300 ./run-tests.sh $1-workers-300-padding-1mb