#!/usr/bin/env bash

kill_param="$1"
sigint_received=false

function handle_sigint() {
    echo "捕获到 SIGINT 信号，正在尝试停止进程: $pid"
    kill -SIGINT $pid
    sigint_received=true
    sleep 2  # 等待一段时间，看进程是否响应 SIGINT
    if ps -p $pid > /dev/null; then
        echo "SIGINT 未能停止进程，强制执行 kill -9"
        kill -9 $pid
    fi
}

function main(){
    if [ $# -ne 1 ]; then
        help
    fi
    pid=$(ps -ef | grep "${kill_param}" | grep -v grep | grep -v "$(basename $0)" | awk '{print $2}' | xargs)
    if [ -n "$pid" ]; then
        echo "正在尝试停止进程: $pid"
        trap handle_sigint SIGINT
        kill -SIGINT $pid
        sleep 2  # 等待一段时间，看进程是否响应 SIGINT
        if ps -p $pid > /dev/null; then
            echo "SIGINT 未能停止进程，强制执行 kill -9"
            kill -9 $pid
        fi

        if [ $? -eq 0 ]; then
            echo "成功停止进程: $pid"
        else
            echo "停止进程失败: $pid"
            exit 1
        fi
    else
        echo "No process found for: ${kill_param}"
    fi
}

function help(){
    echo "Usage: bash $0 <kill_param>"
    exit 1
}

main "$@"