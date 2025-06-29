#!/usr/bin/bash

set -euo pipefail

SCRIPT_DIR=$(dirname "$(realpath "$0")")
ROOT_DIR=$(dirname "$SCRIPT_DIR")
IMAGE_DIR="$ROOT_DIR/image"
WORK_DIR="$ROOT_DIR/work"
LOG_DIR="$ROOT_DIR/log"
COMMIT_ID="${1:-}"

if [[ -z "$COMMIT_ID" ]]; then
  echo "缺少 COMMIT_ID 参数"
  exit 1
fi

LINUX_BUILD_DIR="$ROOT_DIR/build/$COMMIT_ID/linux-$COMMIT_ID"
IMAGE_PATH="$WORK_DIR/$COMMIT_ID/debian.img"
MNT_DIR="$WORK_DIR/$COMMIT_ID/mnt"
LOG_PATH="$LOG_DIR/$COMMIT_ID.log"

if [[ ! -f "$IMAGE_PATH" ]]; then
  echo "镜像文件不存在: $IMAGE_PATH"
  exit 1
fi

if [[ ! -d "$MNT_DIR" ]]; then
  mkdir -p "$MNT_DIR"
fi

if mountpoint -q "$MNT_DIR"; then
  sudo umount "$MNT_DIR"
fi

sudo mount -o loop "$IMAGE_PATH" "$MNT_DIR"
trap 'sudo umount "$MNT_DIR" || true' EXIT

VMCORE_PATH="$MNT_DIR/var/crash/vmcore"
if [[ -f "$VMCORE_PATH" ]]; then
  mkdir -p "$LINUX_BUILD_DIR"
  sudo chmod 644 "$VMCORE_PATH"
  echo "改变 vmcore 权限为644"
  sudo mv "$VMCORE_PATH" "$LINUX_BUILD_DIR"
  echo "vmcore 已移动到: $LINUX_BUILD_DIR"
else
  echo "未找到 vmcore 文件: $VMCORE_PATH"
fi

if [[ -f "$LOG_PATH" ]]; then
  mv "$LOG_PATH" "$LINUX_BUILD_DIR"
  echo "日志目录已移动到: $LINUX_BUILD_DIR"
else
  echo "日志目录不存在: $LOG_PATH"
fi
