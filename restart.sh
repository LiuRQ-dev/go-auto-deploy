
set -e  

SERVICE_NAME=${1:-"my-app"}
DEPLOY_LOG="/var/log/deploy.log"
BACKUP_DIR="/var/backups/app"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$DEPLOY_LOG"
}

log "開始重啟服務: $SERVICE_NAME"


if [[ $EUID -ne 0 ]]; then
   log "警告: 建議以 root 權限運行此腳本"
fi


mkdir -p "$BACKUP_DIR"


if systemctl list-units --full -all | grep -Fq "$SERVICE_NAME.service"; then
    log "使用 systemctl 管理服務"
    

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        log "服務正在運行，準備重啟"
        

        if [[ -f "/var/www/$SERVICE_NAME/app" || -f "/var/www/$SERVICE_NAME/main" ]]; then
            BACKUP_FILE="$BACKUP_DIR/${SERVICE_NAME}_$(date +%Y%m%d_%H%M%S).tar.gz"
            log "創建備份: $BACKUP_FILE"
            tar -czf "$BACKUP_FILE" -C "/var/www" "$SERVICE_NAME" 2>/dev/null || log "備份失敗，繼續執行"
        fi
        

        log "重啟服務: $SERVICE_NAME"
        systemctl restart "$SERVICE_NAME"
        

        sleep 3
        

        if systemctl is-active --quiet "$SERVICE_NAME"; then
            log "服務重啟成功"
        else
            log "服務重啟失敗"
            systemctl status "$SERVICE_NAME" >> "$DEPLOY_LOG" 2>&1
            exit 1
        fi
    else
        log "服務未運行，嘗試啟動"
        systemctl start "$SERVICE_NAME"
        
        sleep 3
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            log "服務啟動成功"
        else
            log "服務啟動失敗"
            exit 1
        fi
    fi
    
elif [[ -f "/etc/init.d/$SERVICE_NAME" ]]; then
    log "使用 init.d 管理服務"
    /etc/init.d/$SERVICE_NAME restart
    
elif command -v pm2 >/dev/null && pm2 list | grep -q "$SERVICE_NAME"; then
    log "使用 PM2 管理服務"
    pm2 restart "$SERVICE_NAME"
    pm2 save
    
elif pgrep -f "$SERVICE_NAME" > /dev/null; then
    log "使用進程管理"
    

    PID=$(pgrep -f "$SERVICE_NAME")
    log "發現進程 PID: $PID"
    

    log "發送 SIGTERM 信號"
    kill -TERM "$PID"
    

    for i in {1..10}; do
        if ! kill -0 "$PID" 2>/dev/null; then
            log "進程已優雅停止"
            break
        fi
        log "等待進程停止... ($i/10)"
        sleep 1
    done
    

    if kill -0 "$PID" 2>/dev/null; then
        log "強制停止進程"
        kill -KILL "$PID"
        sleep 1
    fi
    

    log "重新啟動應用"
    if [[ -f "/var/www/$SERVICE_NAME/start.sh" ]]; then
        cd "/var/www/$SERVICE_NAME"
        nohup ./start.sh > /dev/null 2>&1 &
    elif [[ -f "/var/www/$SERVICE_NAME/$SERVICE_NAME" ]]; then
        cd "/var/www/$SERVICE_NAME"
        nohup "./$SERVICE_NAME" > /dev/null 2>&1 &
    else
        log "無法找到啟動腳本"
        exit 1
    fi
    
    sleep 2
    if pgrep -f "$SERVICE_NAME" > /dev/null; then
        log "應用重啟成功"
    else
        log "應用重啟失敗"
        exit 1
    fi
    
else
    log "無法找到服務: $SERVICE_NAME"
    log "支持的管理方式: systemctl, init.d, pm2, 或直接進程管理"
    exit 1
fi


if [[ -n "$HEALTH_CHECK_URL" ]]; then
    log "執行健康檢查: $HEALTH_CHECK_URL"
    
    for i in {1..5}; do
        if curl -s -f "$HEALTH_CHECK_URL" > /dev/null; then
            log "健康檢查通過"
            break
        else
            log "健康檢查未通過，等待... ($i/5)"
            sleep 2
        fi
        
        if [[ $i -eq 5 ]]; then
            log "健康檢查失敗"
            exit 1
        fi
    done
fi


if [[ -d "$BACKUP_DIR" ]]; then
    log "清理舊備份文件"
    ls -t "$BACKUP_DIR"/${SERVICE_NAME}_*.tar.gz 2>/dev/null | tail -n +6 | xargs -r rm -f
fi

log "🎉 重啟完成"