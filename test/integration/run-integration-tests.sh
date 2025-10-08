#!/bin/bash

# run-integration-tests.sh

# This script runs the DB layer integration tests on behalf of the (top-level) Makefile.

# This is set based on the path Go puts in the binary for the variables:
ldflagVarPrefix='gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db'

: ${THARSIS_DB_TEST_HOST:="localhost"}
: ${THARSIS_DB_TEST_PORT:="29432"}
: ${THARSIS_DB_TEST_NAME:="tharsisdbtest"}
: ${THARSIS_DB_TEST_SSL_MODE:="disable"}
: ${THARSIS_DB_TEST_USERNAME:="postgres"}
: ${THARSIS_DB_TEST_PASSWORD:="postgres"}

THARSIS_DB_TEST_CONTAINER_PORT=5432
THARSIS_DB_TEST_INSTANCE_NAME=postgres-integration-test-server

function cleanup {
	docker kill ${THARSIS_DB_TEST_INSTANCE_NAME} &> /dev/null || true
	docker rm ${THARSIS_DB_TEST_INSTANCE_NAME} &> /dev/null || true
}

# Remove any existing container.
cleanup

docker run -d --name ${THARSIS_DB_TEST_INSTANCE_NAME}        	  \
	-e POSTGRES_DB=${THARSIS_DB_TEST_NAME}                        \
	-e POSTGRES_USER=${THARSIS_DB_TEST_USERNAME}                  \
	-e POSTGRES_PASSWORD=${THARSIS_DB_TEST_PASSWORD}              \
	-p ${THARSIS_DB_TEST_PORT}:${THARSIS_DB_TEST_CONTAINER_PORT}  \
	postgres:16

LIMIT=40
SLEEP=1
READY=
for ((i=1;i<=LIMIT;i++)); do
	if docker exec ${THARSIS_DB_TEST_INSTANCE_NAME} pg_isready -U ${THARSIS_DB_TEST_USERNAME} -d ${THARSIS_DB_TEST_NAME} &> /dev/null; then
		READY=1
		break
	fi
	sleep ${SLEEP}
done

if [ -z "${READY}" ]; then
	echo "Docker container did not start in time."
	docker logs ${THARSIS_DB_TEST_INSTANCE_NAME}
	cleanup # Ensure we clean up the container.
	exit 1
fi

go test -count=1 -tags=integration,noui --ldflags "-X ${ldflagVarPrefix}.TestDBHost=${THARSIS_DB_TEST_HOST} -X ${ldflagVarPrefix}.TestDBPort=${THARSIS_DB_TEST_PORT} -X ${ldflagVarPrefix}.TestDBName=${THARSIS_DB_TEST_NAME} -X ${ldflagVarPrefix}.TestDBMode=${THARSIS_DB_TEST_SSL_MODE} -X ${ldflagVarPrefix}.TestDBUser=${THARSIS_DB_TEST_USERNAME} -X ${ldflagVarPrefix}.TestDBPass=${THARSIS_DB_TEST_PASSWORD}" ./...
returnStatus=$?

# Clean up the container.
cleanup

# Return the status of the test.
exit "${returnStatus}"
