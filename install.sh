#!/bin/bash
set -e

APP_NAME="fnsqldb"
INSTALL_DIR="/opt/fnSqlJob"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
GITHUB_REPO="dingdadao/FnSqlJob"
GITHUB_PROXY="https://githubotc.dension.dpdns.org"
DOWNLOAD_URL="${GITHUB_PROXY}/https://github.com/${GITHUB_REPO}/releases/latest/download/${APP_NAME}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
err() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        err "请使用 root 用户运行: sudo bash install.sh"
    fi
}

download_binary() {
    log "从 GitHub 下载最新版本 ..."
    mkdir -p "${INSTALL_DIR}"
    local tmp_file="${INSTALL_DIR}/${APP_NAME}.tmp"

    if command -v curl &>/dev/null; then
        curl -fSL -o "${tmp_file}" "${DOWNLOAD_URL}" || err "下载失败，请检查网络或手动下载"
    elif command -v wget &>/dev/null; then
        wget -O "${tmp_file}" "${DOWNLOAD_URL}" || err "下载失败，请检查网络或手动下载"
    else
        err "需要 curl 或 wget，请先安装"
    fi

    # 验证下载的是二进制文件 (不是 HTML)
    local file_size=$(stat -c%s "${tmp_file}" 2>/dev/null || stat -f%z "${tmp_file}" 2>/dev/null || echo 0)
    if [ "${file_size}" -lt 1000000 ]; then
        rm -f "${tmp_file}"
        err "下载失败: 文件仅 ${file_size} 字节，可能代理不支持重定向。请手动下载: https://github.com/${GITHUB_REPO}/releases/latest"
    fi

    local file_type=$(file "${tmp_file}" 2>/dev/null)
    if echo "${file_type}" | grep -qi "html\|text"; then
        rm -f "${tmp_file}"
        err "下载失败: 返回了 HTML 页面。请手动下载: https://github.com/${GITHUB_REPO}/releases/latest"
    fi

    mv -f "${tmp_file}" "${INSTALL_DIR}/${APP_NAME}"
    chmod +x "${INSTALL_DIR}/${APP_NAME}"
    log "下载完成 (${file_size} bytes)"
}

install_binary() {
    mkdir -p "${INSTALL_DIR}"

    # 优先使用本地文件
    if [ -f "./fnsqldb" ]; then
        log "使用本地二进制 ..."
        cp -f ./fnsqldb "${INSTALL_DIR}/${APP_NAME}"
        chmod +x "${INSTALL_DIR}/${APP_NAME}"
    elif [ -f "./fnsqldb-linux" ]; then
        log "使用本地二进制 ..."
        cp -f ./fnsqldb-linux "${INSTALL_DIR}/${APP_NAME}"
        chmod +x "${INSTALL_DIR}/${APP_NAME}"
    else
        # 本地没有则从 GitHub 下载
        download_binary
    fi
}

install_service() {
    log "创建 systemd 服务 ..."

    cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=FnSqlDB - SQLite CRUD API Service
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${APP_NAME} -dbpath /usr/local/apps/@appdata/trim.media/database/ -addr :8877
WorkingDirectory=${INSTALL_DIR}
Restart=always
RestartSec=5
StartLimitIntervalSec=60
StartLimitBurst=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${APP_NAME}

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log "systemd 服务创建完成"
}

uninstall() {
    log "卸载 ${APP_NAME} ..."
    systemctl stop "${APP_NAME}" 2>/dev/null || true
    systemctl disable "${APP_NAME}" 2>/dev/null || true
    rm -f "${SERVICE_FILE}"
    rm -rf "${INSTALL_DIR}"
    systemctl daemon-reload
    log "卸载完成"
}

show_usage() {
    echo "用法: sudo bash install.sh [命令]"
    echo ""
    echo "命令:"
    echo "  install    安装服务 (本地有二进制用本地，否则从 GitHub 下载)"
    echo "  update     从 GitHub 下载最新版本并重启"
    echo "  start      启动服务"
    echo "  stop       停止服务"
    echo "  restart    重启服务"
    echo "  status     查看服务状态"
    echo "  logs       查看实时日志"
    echo "  uninstall  卸载服务"
}

case "${1:-install}" in
    install)
        check_root
        install_binary
        install_service
        systemctl enable "${APP_NAME}"
        systemctl start "${APP_NAME}"
        log "安装完成，服务已启动"
        echo ""
        systemctl status "${APP_NAME}" --no-pager
        ;;
    update)
        check_root
        download_binary
        systemctl restart "${APP_NAME}"
        log "更新完成，服务已重启"
        systemctl status "${APP_NAME}" --no-pager
        ;;
    start)
        check_root
        systemctl start "${APP_NAME}"
        log "服务已启动"
        ;;
    stop)
        check_root
        systemctl stop "${APP_NAME}"
        log "服务已停止"
        ;;
    restart)
        check_root
        systemctl restart "${APP_NAME}"
        log "服务已重启"
        ;;
    status)
        systemctl status "${APP_NAME}" --no-pager
        ;;
    logs)
        journalctl -u "${APP_NAME}" -f --no-pager
        ;;
    uninstall)
        check_root
        uninstall
        ;;
    *)
        show_usage
        ;;
esac
