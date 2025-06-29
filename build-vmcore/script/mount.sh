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

create_dir() {
    local DIR=$1
    if [ ! -d "$DIR" ]; then
        mkdir -p "$DIR" || error_exit "Failed to create directory: $DIR"
        log "INFO" "Directory created: $DIR"
    fi
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

create_dir "$WORK_DIR"

cd "$WORK_DIR" || error_exit "Failed to change directory to $WORK_DIR"

create_dir "$WORK_DIR/$COMMIT_ID"
cd "$COMMIT_ID" || error_exit "Failed to change directory to $WORK_DIR/$COMMIT_ID"

log "INFO" "Copying debian.img..."
rsync -av "$IMAGE_DIR/debian.img" ./ || error_exit "Failed to copy debian.img"

create_dir "mnt"

log "INFO" "Mounting debian.img..."
sudo mount -o loop debian.img mnt || error_exit "Failed to mount debian.img"

log "INFO" "Copying Linux headers..."
cd mnt/usr/include || error_exit "Failed to enter mnt/usr/include"
sudo rm -rf ./asm || error_exit "Failed to remove asm directory"
sudo rm -rf ./linux || error_exit "Failed to remove linux directory"
sudo cp -R "$LINUX_BUILD_DIR/linux-header/include/asm" ./ || error_exit "Failed to copy asm headers"
sudo cp -R "$LINUX_BUILD_DIR/linux-header/include/linux" ./ || error_exit "Failed to copy linux headers"

log "INFO" "Copying bug.c to root..."
cd ../../ || error_exit "Failed to return to mnt directory"
sudo cp -R "$LINUX_BUILD_DIR/bug.c" ./root || error_exit "Failed to copy bug.c"

log "INFO" "Unmounting debian.img..."
cd ..
sudo umount mnt || error_exit "Failed to unmount debian.img"

cp "$LINUX_BUILD_DIR/arch/x86_64/boot/bzImage" ./

log "INFO" "Image has been successfully copied."