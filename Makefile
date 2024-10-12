.PHONY: clean reload run

# 当前时间
NOW = $(shell date -u '+%Y%m%d%I%M%S')
# APP_NAME
APP 			= log-engine-sdk
# ./cmd/log-engine-sdk/log-engine-sdk 工程编译的文件地址
SERVER_BIN  	= ./cmd/$(APP)/$(APP)

# 可发布版本号
RELEASE_VERSION = v1.0.1
# 可发布的版本存储路径的根目录
RELEASE_ROOT 	= release
# 可发布的版本存储路径  release/log-engine-sdk
RELEASE_SERVER 	= release/$(APP)

# 统计git提交的总数
GIT_COUNT 	= $(shell git rev-list --all --count)
# 统计git当前版本的hash值
GIT_HASH    = $(shell git rev-parse --short HEAD)
# 拼接成可发版版本的命名tag
RELEASE_TAG = $(RELEASE_VERSION).$(GIT_COUNT).$(GIT_HASH)

# ldflags 参数 , -X 命令可以用于往main包传入参数这里传入了 version, tag, build 3个参数值
GO_LDFLAGS = -X main.Version=$(RELEASE_VERSION)  -X main.Tag=$(RELEASE_TAG) -X main.BuildTime=$(NOW)

DIRS = log scripts

# 命令名称
all: pack

test:
	@go test -v ./pkg/...

build:
	@mkdir -p log
	@go build -ldflags "-w -s $(GO_LDFLAGS)" -x -a -o $(SERVER_BIN) ./cmd/main.go

clean:
	@rm -rf $(SERVER_BIN) $(RELEASE_ROOT) log/*

pack: clean build
	# 创建release/$(APP) 目录
	@mkdir -p $(RELEASE_SERVER)
	# 将必要的文件和目录都copy到发布目录
	@cp -r $(SERVER_BIN) configs Makefile $(DIRS) $(RELEASE_SERVER)
	# 给发布目录打包
	@cd $(RELEASE_ROOT) && tar -zcvf $(APP)_$(RELEASE_TAG).tar.gz $(APP) && rm -rf $(APP)
	# pack proj cap-gin path :
	@echo "project pack path :" $(shell pwd)/$(RELEASE_ROOT)/$(APP)_$(RELEASE_TAG).tar.gz

run:
	@go run -ldflags "-w -s $(GO_LDFLAGS)" ./cmd/main.go

start:
	./${APP} > log/$(APP).log 2>&1 &

stop:
	./scripts/stop.sh $(APP)

reload: stop start