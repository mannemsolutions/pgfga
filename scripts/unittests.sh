#!/bin/bash
set -x

if [ "$(id -u)" -eq 0 ]; then
	useradd testuser -m
	su testuser -c "$0"
	exit $?
fi

cd "$(dirname "$0")/.." || exit 1

make check-coverage
