#!/bin/bash

log() {
    local LEVEL=$1
    shift
    echo "[$LEVEL] $(date +'%Y-%m-%d %H:%M:%S') $*"
}

error_exit() {
    log "ERROR" "$1"
    exit 1
}

SCRIPT_DIR=$(dirname "$(realpath "$0")")
ROOT_DIR=$(dirname "$SCRIPT_DIR")
IMAGE_DIR="$ROOT_DIR/image"
WORK_DIR="$ROOT_DIR/work"
COMMIT_ID=$1
LINUX_BUILD_DIR="$ROOT_DIR/build/$COMMIT_ID/linux-$COMMIT_ID"

if [ -z "$COMMIT_ID" ]; then
    error_exit "Commit ID is required as an argument."
fi

log "INFO" "Starting process with COMMIT_ID: $COMMIT_ID"

cd "$WORK_DIR" || error_exit "Failed to change directory to $WORK_DIR"

TARGET_DIR="$WORK_DIR/$COMMIT_ID"
if [ -d "$TARGET_DIR" ]; then
    log "INFO" "Removing existing directory: $TARGET_DIR"
    sudo rm -rf "$TARGET_DIR" || error_exit "Failed to remove directory: $TARGET_DIR"
else
    log "INFO" "Directory $TARGET_DIR does not exist, no need to remove."
fi