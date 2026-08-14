#!/bin/bash
set -e

APP_NAME="fnsqldb"
INSTALL_DIR="/opt/fnSqlJob"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
BINARY_URL=""  # 留空则从本地复制

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

install_binary() {
    log "安装 ${APP_NAME} 到 ${INSTALL_DIR} ..."
    mkdir -p "${INSTALL_DIR}"

    if [ -f "./fnsqldb" ]; then
        cp -f ./fnsqldb "${INSTALL_DIR}/${APP_NAME}"
    elif [ -f "./fnsqldb-linux" ]; then
        cp -f ./fnsqldb-linux "${INSTALL_DIR}/${APP_NAME}"
    else
        err "未找到 fnsqldb 二进制文件，请先执行 build.sh 编译"
    fi

    chmod +x "${INSTALL_DIR}/${APP_NAME}"
    log "二进制安装完成"
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
    echo "  install    安装服务并设置开机自启 (默认)"
    echo "  start      启动服务"
    echo "  stop       停止服务"
    echo "  restart    重启服务"
    echo "  status     查看服务状态"
    echo "  logs       查看实时日志"
    echo "  uninstall  卸载服务"
    echo "  update     更新二进制文件 (不停止服务会自动重启)"
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
        echo "  启动: bash install.sh start"
        echo "  停止: bash install.sh stop"
        echo "  重启: bash install.sh restart"
        echo "  状态: bash install.sh status"
        echo "  日志: bash install.sh logs"
        echo "  卸载: bash install.sh uninstall"
        echo ""
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
    update)
        check_root
        install_binary
        systemctl restart "${APP_NAME}"
        log "更新完成，服务已重启"
        ;;
    *)
        show_usage
        ;;
esac
