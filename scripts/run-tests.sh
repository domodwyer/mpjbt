#!/usr/bin/env bash

# run-tests.sh
# 
# Runs all workloads against both databases, switching database on and off as
# required, creating indexes and sizing the ZFS ARC.
# 
# Results are wrote to ./output/<test name> which is expected to be a git
# repository - after each test completes "git push" is called to push the
# results to the remote.
# 
# By default an index is created on the ID field and a partial index covers
# records with an "age" field covering 40 < X < 75. To change, edit
# reset_postgres or reset_mongo respectively.

set -EC

#############################################
# CONFIG
# 
# You probably want to set these outside of this script (say, in run-suite.sh)
# or at the command line.

# Git config
GIT=${GIT:-"/usr/bin/git"}

# Service control
MONGO_ON=${MONGO_ON:-"service mongod onestart"}
MONGO_OFF=${MONGO_OFF:-"service mongod onestop"}

PG_ON=${PG_ON:-"service postgresql onestart"}
PG_OFF=${PG_OFF:-"service postgresql onestop"}

# Mongo config
MONGO_CONN=${MONGO_CONN:-"mongodb://127.0.0.1/test"} # MongoDB connection string
MONGO_SHELL=${MONGO_SHELL:-"/usr/local/bin/mongo $MONGO_CONN"}
MONGO_SET_ARC=${MONGO_SET_ARC:-""}

# Postgres config
PG_SHELL=${PG_SHELL:-"/usr/local/bin/psql --dbname test"} #DB name must match below
PG_CONN=${PG_CONN:-"postgres://127.0.0.1/test?sslmode=disable;binary_parameters=yes"}
PG_SET_ARC=${PG_SET_ARC:-""}

# Benchmark config
BENCH_TOOL=${BENCH_TOOL:-"./mpjbt"}
WORKERS=${WORKERS:-30}
PADDING=${PADDING:-""}

# Number of operations - updates seem slow so let's do less
OPS_COUNT=${OPS_COUNT:-100000}
UPDATE_COUNT=${UPDATE_COUNT:-10000}

### END OF USER CONFIG ###

# Script vars
TEST_NAME=$1
TABLE_NAME=$(echo $TEST_NAME | tr -cd '[a-zA-Z0-9]')
OUTPUT_DIR="./output/$TEST_NAME"

# Ensure we have a test name
if [ "$#" -ne 1 ]; then
	echo "Usage: $0	TESTNAME"
	exit 1
fi

# Ensure the test name has not already been used
if [ -d "$OUTPUT_DIR" ]; then
	echo "$OUTPUT_DIR already exists - use a different test name!"
	exit 1
fi

# Create the data directory for this run
mkdir -p $OUTPUT_DIR

# Ensure the git repo is configured
if [ ! -d "$OUTPUT_DIR/../.git" ]; then
	echo "./output/.git doesn't exist - is it a git repo?"
	exit 1
fi

# git_push stages any unstaged files, commits using the first argument as the
# message, and attempts to push.
# 
# Failures to push are not fatal.
# 
# git_push <commit message>
git_push() {
	pushd $OUTPUT_DIR
	$GIT add -A
	$GIT status
	$GIT commit -m "$1"
	$GIT push || true # Attempt to push
	popd
}

# error_cleanup catches the output of any failed commands and writes it to
# $OUTPUT_DIR/error.txt - the repo is then pushed.
error_cleanup() {
	ret=$?
	msg="ERROR: '$this_command' returned $ret"
	echo 
	echo $msg
	echo 
	echo $msg >> $OUTPUT_DIR/error.txt
	git_push "failed test: $msg"
	exit $ret
}

trap 'error_cleanup' ERR
trap 'previous_command=$this_command; this_command=$BASH_COMMAND' DEBUG

# reset_mongo drops the test collection
reset_mongo() {
	$MONGO_SHELL --eval "db.$TABLE_NAME.drop()"
	$MONGO_SHELL --eval "db.$TABLE_NAME.createIndex( { 'age': 1 },{ partialFilterExpression: { 'age': { '\$gt': 45,'\$lt': 75} } })"
	return $?
}

# reset_postgres drops the test table and indexes and then recreates them.
reset_postgres() {
	$PG_SHELL -c "DROP TABLE $TABLE_NAME;"  || true
	$PG_SHELL -c "DROP INDEX idx_json_data;"  || true
	$PG_SHELL -c "DROP INDEX idx_json_data_age;"  || true

	$PG_SHELL -c "CREATE TABLE $TABLE_NAME (data jsonb);"
	$PG_SHELL -c "CREATE INDEX idx_json_data ON $TABLE_NAME USING BTREE ((data->'id'));"
	$PG_SHELL -c "CREATE INDEX idx_json_data_age ON $TABLE_NAME USING BTREE ((data->'age')) where data->'age' > '45' and data->'age' < '75';"
}

TEST_NUMBER=0

# run_test runs a single benchmark test.
# 
# The first argument is the workload name. The second argument is the connection
# string to use - for more info, check the tool --help output.
# 
# When a test successfully completes the results are pushed to git (see
# git_push).
# 
# run_test <workload> <connection_string>
function run_test {
	workload=$1
	connection=$2
	type=$(echo $connection | cut -f1 -d ":")
	printf -v TEST_NUMBER "%02d" $(( ${TEST_NUMBER#0} +1))

	mkdir -p $OUTPUT_DIR/$type

	echo 
	echo "Starting test: $type $workload"
	echo 

	$BENCH_TOOL -workload=$workload \
		-connect=$connection \
		-histogram=$OUTPUT_DIR/$type/$TEST_NUMBER-$workload.csv \
		-ops=$OPS_COUNT \
		-padding=$PADDING \
		-table=$TABLE_NAME \
		-workers=$WORKERS \
		2>&1 | tee $OUTPUT_DIR/$type/$TEST_NUMBER-$workload.log
	
	git_push "$TEST_NAME - $TEST_NUMBER - $type $workload"
}

eval $PG_OFF || true
eval $MONGO_OFF || true

# #############################################
# # Mongo tests

eval $MONGO_SET_ARC
eval $MONGO_ON
sleep 10
reset_mongo

# Write mongo config - thanks stack overflow!
# 
# https://stackoverflow.com/questions/31028235/how-to-execute-mongo-commands-from-bash
$MONGO_SHELL \
	--eval "db=db.getSiblingDB('admin');db.admin.runCommand({getCmdLineOpts:1})" \
	> $OUTPUT_DIR/config-mongo.txt

run_test "insert" $MONGO_CONN
run_test "select-zipfian" $MONGO_CONN
run_test "select-uniform" $MONGO_CONN

reset_mongo
run_test "insert" $MONGO_CONN
OPS_COUNT=$UPDATE_COUNT run_test "insert-update" $MONGO_CONN

reset_mongo
run_test "insert" $MONGO_CONN
run_test "insert5-select95" $MONGO_CONN

reset_mongo
run_test "insert" $MONGO_CONN
OPS_COUNT=$UPDATE_COUNT run_test "select-update-uniform" $MONGO_CONN

reset_mongo
run_test "insert" $MONGO_CONN
OPS_COUNT=$UPDATE_COUNT run_test "select-update-zipfian" $MONGO_CONN

reset_mongo
run_test "insert" $MONGO_CONN
OPS_COUNT=$UPDATE_COUNT run_test "update-zipfian" $MONGO_CONN

reset_mongo
run_test "insert" $MONGO_CONN
OPS_COUNT=$UPDATE_COUNT run_test "update-uniform" $MONGO_CONN

reset_mongo
run_test "insert" $MONGO_CONN
run_test "read-range" $MONGO_CONN


#############################################
# Postgres tests

# Drop the postgres table and create indexes
eval $MONGO_OFF || true
eval $PG_SET_ARC
eval $PG_ON
sleep 10
reset_postgres

# Write postgres config to file
$PG_SHELL -c "SHOW ALL;" > $OUTPUT_DIR/config-postgres.txt

run_test "insert" $PG_CONN
run_test "select-zipfian" $PG_CONN
run_test "select-uniform" $PG_CONN

reset_postgres
run_test "insert" $PG_CONN
OPS_COUNT=$UPDATE_COUNT run_test "insert-update" $PG_CONN

reset_postgres
run_test "insert" $PG_CONN
run_test "insert5-select95" $PG_CONN

reset_postgres
run_test "insert" $PG_CONN
OPS_COUNT=$UPDATE_COUNT run_test "select-update-uniform" $PG_CONN

reset_postgres
run_test "insert" $PG_CONN
OPS_COUNT=$UPDATE_COUNT run_test "select-update-zipfian" $PG_CONN

reset_postgres
run_test "insert" $PG_CONN
OPS_COUNT=$UPDATE_COUNT run_test "update-zipfian" $PG_CONN

reset_postgres
run_test "insert" $PG_CONN
OPS_COUNT=$UPDATE_COUNT run_test "update-uniform" $PG_CONN

reset_postgres
run_test "insert" $PG_CONN
run_test "read-range" $PG_CONN
