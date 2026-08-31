# 启动

Codex 看到任何“启动项目 / 启动前后端 / 读取启动脚本启动 / 启动全部服务”类请求时，只做这一件事：在仓库根目录直接执行：

```bash
./restart_all_services.sh restart
```

不要先列文件；不要读 `README.md`、`DATABASES.md`、`package.json`、`Makefile`、`docker-compose.yml` 或脚本源码；不要分析项目结构；不要手动查端口或 PID；不要选择单个服务脚本；不要额外健康检查。脚本会自己停止旧进程、清理占用端口、启动全部服务、等待端口可用并打印地址。

在 Codex Desktop 里，如果因为沙盒导致端口监听、清理进程、`ps`/`lsof`/`launchctl`、Docker、数据库或 Redis 连接失败，立刻申请沙盒外权限重跑同一条命令，不要改命令。

命令成功后只回复脚本打印的服务地址。

只有命令失败时，才看日志：

- 运行日志目录：`.service-runtime/logs/`
- PID 目录：`.service-runtime/pids/`

停止：

```bash
./restart_all_services.sh stop
```
