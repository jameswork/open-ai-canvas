# ToAPIs Images

该目录是独立官方协议插件源码。后端从生成的 `toapis-image.yingce-plugin` 包加载，不依赖系统内置 `host:` 适配器。

适配 [ToAPIs](https://docs.toapis.cn) 的图片生成异步任务接口（`POST /v1/images/generations` + `GET /v1/images/generations/{task_id}`），覆盖 Seedream 5.5（`doubao-seedream-5-5`）等文生图 / 参考图编辑 / 图层拆分模型。鉴权、私网校验、轮询、结果下载与计费由宿主统一负责。

完整接口见 [docs/interface.md](docs/interface.md)。
