#!/usr/bin/env bash

kill_param="$1"

function main(){
    if [ $# -ne 1 ]; then
        help
    fi
    pid=$(ps -ef | grep "${kill_param}" | grep -v grep |grep -v "$(basename $0)"| awk '{print $2}'|xargs )
    if [ -n "$pid" ]; then
        echo "正在尝试停止进程: $pid"
        kill -9 ${pid}

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