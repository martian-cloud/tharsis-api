#!/bin/bash

# run-integration-tests.sh

# This script runs the DB layer integration tests on behalf of the (top-level) Makefile.

# This is set based on the path Go puts in the binary for the variables:
ldflagVarPrefix='gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db'

LAUNCH_LOCAL_IF_ZERO="$(echo "${THARSIS_DB_TEST_HOST}" | wc -w)"

: ${THARSIS_DB_TEST_HOST:="localhost"}
: ${THARSIS_DB_TEST_PORT:="29432"}
: ${THARSIS_DB_TEST_NAME:="tharsisdbtest"}
: ${THARSIS_DB_TEST_SSL_MODE:="disable"}
: ${THARSIS_DB_TEST_USERNAME:="postgres"}
: ${THARSIS_DB_TEST_PASSWORD:="postgres"}

THARSIS_DB_TEST_CONTAINER_PORT=5432
THARSIS_DB_TEST_INSTANCE_NAME=postgres-integration-test-server

: ${THARSIS_DB_TEST_URI:="pgx://${THARSIS_DB_TEST_USERNAME}:${THARSIS_DB_TEST_PASSWORD}@${THARSIS_DB_TEST_HOST}:${THARSIS_DB_TEST_PORT}/${THARSIS_DB_TEST_NAME}?sslmode=${THARSIS_DB_TEST_SSL_MODE}"}

THARSIS_DB_TEST_MIGRATE="docker run --rm -v $(pwd)/internal/db/migrations:/migrations --network host migrate/migrate:v4.15.2 -path=/migrations/ -database ${THARSIS_DB_TEST_URI}"

if [ ${LAUNCH_LOCAL_IF_ZERO} == 0 ]; then
	docker run -d --rm --name ${THARSIS_DB_TEST_INSTANCE_NAME}                 \
		-e POSTGRES_DB=${THARSIS_DB_TEST_NAME}                             \
		-e POSTGRES_USER=${THARSIS_DB_TEST_USERNAME}                       \
		-e POSTGRES_PASSWORD=${THARSIS_DB_TEST_PASSWORD}                   \
		-p ${THARSIS_DB_TEST_PORT}:${THARSIS_DB_TEST_CONTAINER_PORT}   \
		postgres

	LIMIT=40
	SLEEP=1
	READY=
	CHECK="psql -h ${THARSIS_DB_TEST_HOST} -p ${THARSIS_DB_TEST_PORT} ${THARSIS_DB_TEST_NAME} ${THARSIS_DB_TEST_USERNAME}"
	for ((i=1;i<=LIMIT;i++)); do
		if ${CHECK} < /dev/null >& /dev/null; then
			READY=1
			break
		fi
		sleep ${SLEEP}
	done
fi

${THARSIS_DB_TEST_MIGRATE} -verbose up

go test -tags=integration --ldflags "-X ${ldflagVarPrefix}.TestDBHost=${THARSIS_DB_TEST_HOST} -X ${ldflagVarPrefix}.TestDBPort=${THARSIS_DB_TEST_PORT} -X ${ldflagVarPrefix}.TestDBName=${THARSIS_DB_TEST_NAME} -X ${ldflagVarPrefix}.TestDBMode=${THARSIS_DB_TEST_SSL_MODE} -X ${ldflagVarPrefix}.TestDBUser=${THARSIS_DB_TEST_USERNAME} -X ${ldflagVarPrefix}.TestDBPass=${THARSIS_DB_TEST_PASSWORD}" ./...
testCompletionStatus=$?

if [ ${LAUNCH_LOCAL_IF_ZERO} == 0 ]; then
	docker kill ${THARSIS_DB_TEST_INSTANCE_NAME}
fi

exit $testCompletionStatus

# The End.
