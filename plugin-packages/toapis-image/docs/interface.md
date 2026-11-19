# ToAPIs Images 接口字段

## 协议身份

- 插件 ID：`toapis-image`。
- Provider ID：`toapis-image`。
- 能力：`image`。
- 默认 Base URL：`https://toapis.cn`（国内用户；国际站可改为 `https://toapis.com`）。
- 鉴权驱动：`bearer`。
- 创建：`POST /v1/images/generations`。
- 查询：`GET /v1/images/generations/{{taskId}}`。
- 生命周期：异步任务，提交返回 `generation.task`，宿主按 task id 轮询至 `completed` / `failed`。
- 参考图：需公网可访问 URL；`requiresPublicMediaUrls=true`，宿主在请求前把用户素材转换为短期签名公网 URL。

## 配置字段

| 字段 | 类型 | 必填 | 含义 |
| --- | --- | --- | --- |
| `apiKey` | secret | 是 | ToAPIs API Key，以 `Authorization: Bearer` 发送。 |

## 统一字段映射

| 统一字段 | 类型 | 必填 | 上游映射 | 说明 |
| --- | --- | --- | --- | --- |
| `model` | string | 是 | `model` | 图像模型 ID，例如 `doubao-seedream-5-5`。 |
| `prompt` | string | 否 | `prompt` | 文字描述；开启 `layer_decomposition` 时可省略。 |
| `images` | media[] | 否 | `image_urls` | 参考图公网 URL 数组，最多 10 张；由宿主短期签名。 |
| `imageCount` | integer | 否 | `n` | 固定为 1。Seedream 5.5 不支持普通组图。 |
| `aspectRatio` | string | 否 | `size`（`encoding: ratio`） | 比例作为输入，由 `create.body` 的 `$switch` 结合分辨率换算成像素；宿主不预先强制转成像素。 |
| `resolution` | string | 否 | `size` | 档位 1k / 1.5k / 2k，与比例共同决定像素。 |
| `layer_decomposition` | boolean | 否 | `layer_decomposition` | 图层拆分，仅拆分一张参考图。 |
| `layer_decomposition_options` | object | 否 | `layer_decomposition_options` | `max_images`（2–100）与 `text_recognition`。 |
| `optimize_prompt_options` | object | 否 | `optimize_prompt_options` | 提示词优化，仅 `standard`，不接受 `fast`。 |
| `metadata` | object | 否 | `metadata` | `output_format` / `background` / `watermark` / `response_format`。 |

## 尺寸映射（size）

`size` 参数的 `encoding` 为 `ratio`（纯比例线路），宿主不会在边界处强制把比例换算成像素；比例与档位作为输入交给 `create.body` 的 `$switch`，由插件模板按下表换算为 Seedream 5.5 官方像素尺寸下发，避免宿主像素预设表与插件模板双重换算冲突。输出尺寸表：

| 比例 | 1K | 1.5K | 2K |
| --- | --- | --- | --- |
| 1:1 | 1024x1024 | 1536x1536 | 2048x2048 |
| 4:3 | 1152x864 | 1792x1344 | 2368x1776 |
| 3:4 | 864x1152 | 1344x1792 | 1776x2368 |
| 16:9 | 1424x800 | 2048x1152 | 2816x1584 |
| 9:16 | 800x1424 | 1152x2048 | 1584x2816 |
| 3:2 | 1248x832 | 1872x1248 | 2496x1664 |
| 2:3 | 832x1248 | 1248x1872 | 1664x2496 |
| 21:9 | 1568x672 | 2352x1008 | 3136x1344 |

- 比例选 `auto` 时，`size` 直接下发档位字符串 `1K` / `1.5K` / `2K`，由模型决定构图。
- 图层拆分时 `size` 取 `layer_size`（`auto` / `1K` / `1.5K` / `2K`），默认 `auto` 保持原图比例。
- `providerOptions.toapis-image.size` 可覆盖以上全部计算，直接下发原始字符串（如 `2K` 或 `2048x2048`）。

## 上游请求模板逐字段清单

| 上游位置 | 值或转换表达式 |
| --- | --- |
| `create.method` | `"POST"` |
| `create.path` | `"/v1/images/generations"` |
| `create.contentType` | `"application/json"` |
| `create.body.model` | `{"$ref":"request.model"}` |
| `create.body.prompt` | `{"$omitEmpty":{"$ref":"request.prompt"}}` |
| `create.body.n` | `1` |
| `create.body.size` | `{"$coalesce":[{"$ref":"request.providerOptions.toapis-image.size"},{"$if":{...}}]}` — 见尺寸映射。 |
| `create.body.image_urls` | `{"$omitEmpty":{"$map":{"from":{"$ref":"request.images"},"as":"media","in":{"$ref":"media.value"}}}}` |
| `create.body.layer_decomposition` | `{"$omitEmpty":{"$ref":"request.providerOptions.toapis-image.layer_decomposition"}}` |
| `create.body.layer_decomposition_options` | 拆层开启时组装 `max_images` / `text_recognition`，否则省略。 |
| `create.body.optimize_prompt_options` | `{"$omitEmpty":{"$ref":"request.providerOptions.toapis-image.optimize_prompt_options"}}` |
| `create.body.metadata` | `{"$merge":[...]}` — `output_format`（默认 png）、`background`（拆层时不发送）、`watermark`（默认 false）、`response_format`（固定 url）。 |
| `poll.method` | `"GET"` |
| `poll.path` | `"/v1/images/generations/{{taskId}}"` |

## Provider 扩展键

`providerOptions.toapis-image` 命名空间下的扩展键：

- `size`：直接覆盖 `size` 字段。
- `layer_size`：图层拆分尺寸。
- `layer_decomposition`：图层拆分开关。
- `max_images`：拆分最多输出数。
- `text_recognition`：文字识别开关。
- `output_format`：`metadata.output_format`。
- `background`：`metadata.background`。
- `watermark`：`metadata.watermark`。
- `optimize_prompt_options`：提示词优化对象，如 `{"mode":"standard"}`。

动态模型允许在 `providerOptions.toapis-image.optimize_prompt_options` 内传递对象；该对象是协议本身的开放 schema，不会被宿主裁剪。

## 响应映射逐字段清单

| 映射位置 | 上游路径或转换表达式 |
| --- | --- |
| `response.taskId` | `{"$coalesce":[{"$ref":"response.id"},{"$ref":"response.task_id"},{"$ref":"taskId"}]}` |
| `response.status` | `{"$coalesce":[{"$ref":"response.status"},"pending"]}` |
| `response.message` | `{"$coalesce":[{"$ref":"response.error.message"},{"$ref":"response.error"},{"$ref":"response.message"}]}` |
| `response.images` | `{"$coalesce":[{"$ref":"response.result.data"},{"$ref":"response.data"}]}` |
| `response.errorPaths` | `["error.code","error.message","error"]` |

## 响应与错误

- 提交与查询都返回任务对象：`status` 为 `pending` / `queued` / `in_progress` / `completed` / `failed`，宿主按统一词表归一化并持续轮询，直至成功或失败终态。
- 成功完成后读取 `result.data` 数组，每项含 `url`、`size`、`output_format`，图层拆分还含 `z_index`、`bounding_box`、`name`、`description`、`background`、`text_recognition`；这些元数据由宿主自动保留到结果项。
- `resultEphemeral=true`：结果 URL 为短期地址，宿主在任务完成后立即下载并持久化，任务记录只保存宿主资源引用。
- 失败时从 `error.code` / `error.message` / `error` 读取原因，保持失败语义，不把 HTTP 200 包装成成功。
- Seedream 5.5 不支持 `n>1`、`stream=true`、`tools`、`seed`、`sequential_image_generation`；本插件不发送这些字段，无法绕过上游限制。

## 兼容边界

本包只代表 ToAPIs OpenAI 风格异步图片任务协议（`POST /v1/images/generations` + `GET /v1/images/generations/{task_id}`）。同一品牌其他端点、官方 Volcengine Ark 原生接口（`/api/v3/images/generations`）或其他网关包装必须使用独立插件，不能根据模型名猜测。国内站默认 `https://toapis.cn`，国际站需在渠道中把 Base URL 改为 `https://toapis.com`。

<!-- YINGCE_MANIFEST_CONTRACT_START -->
## Manifest 完整接口定义

以下 JSON 与插件包内实际 `manifest.json` 逐字段一致，覆盖插件身份、权限、配置、鉴权、参数、校验、创建、Agent、查询、取消、结果下载、响应和 Agent 响应映射。`documentation` 字段的值就是当前完整文档；为避免文档在自身内部无限递归，JSON 中仅用等义占位文本表示正文。

```json
{
  "apiVersion": "yingce.plugin/v2",
  "id": "toapis-image",
  "name": "ToAPIs Images",
  "version": "1.0.0",
  "author": "影策社区",
  "description": "ToAPIs 异步图片生成协议插件（Seedream 5.5 等），宿主统一鉴权、轮询与资源持久化。",
  "permissions": [
    "generation.run",
    "media.read"
  ],
  "configuration": {
    "fields": [
      {
        "name": "apiKey",
        "type": "secret",
        "label": "API Key",
        "required": true
      }
    ]
  },
  "contributes": {
    "providers": [
      {
        "id": "toapis-image",
        "label": "ToAPIs Images",
        "capabilities": [
          "image"
        ],
        "scopes": [
          "admin.system-channel",
          "user.custom-channel",
          "canvas",
          "creation",
          "agent"
        ],
        "baseUrl": "https://toapis.cn",
        "requiresPublicMediaUrls": true,
        "auth": {
          "type": "bearer",
          "field": "apiKey"
        },
        "parameters": [
          {
            "name": "model",
            "type": "string",
            "required": true,
            "mapping": "model",
            "description": "图像模型 ID，如 doubao-seedream-5-5。"
          },
          {
            "name": "prompt",
            "type": "string",
            "required": false,
            "mapping": "prompt",
            "description": "文字描述；图层拆分时可省略。"
          },
          {
            "name": "images",
            "type": "media[]",
            "required": false,
            "mapping": "image_urls",
            "description": "参考图公网 URL，最多 10 张，由宿主转换为临时公网地址。"
          },
          {
            "name": "size",
            "type": "string",
            "required": false,
            "mapping": "size",
            "description": "1K/1.5K/2K 或显式像素尺寸。"
          },
          {
            "name": "imageCount",
            "type": "integer",
            "required": false,
            "mapping": "n",
            "description": "固定为 1。"
          },
          {
            "name": "layer_decomposition",
            "type": "boolean",
            "required": false,
            "mapping": "layer_decomposition",
            "description": "图层拆分开关。"
          },
          {
            "name": "layer_decomposition_options",
            "type": "object",
            "required": false,
            "mapping": "layer_decomposition_options",
            "description": "拆层选项 max_images 与 text_recognition。"
          },
          {
            "name": "optimize_prompt_options",
            "type": "object",
            "required": false,
            "mapping": "optimize_prompt_options",
            "description": "提示词优化选项，仅支持 standard。"
          },
          {
            "name": "metadata",
            "type": "object",
            "required": false,
            "mapping": "metadata",
            "description": "output_format、background、watermark、response_format。"
          }
        ],
        "imageParameters": [
          {
            "name": "size",
            "label": "画幅 / 尺寸",
            "type": "string",
            "description": "图片画幅比例。选择后按分辨率档位映射到 Seedream 5.5 官方像素尺寸。",
            "values": [
              "1:1",
              "4:3",
              "3:4",
              "16:9",
              "9:16",
              "3:2",
              "2:3",
              "21:9",
              "auto"
            ],
            "default": "1:1",
            "pricing": true,
            "binding": "aspectRatio",
            "encoding": "ratio"
          },
          {
            "name": "imageResolution",
            "label": "分辨率",
            "type": "string",
            "description": "Seedream 5.5 分辨率档位 1K / 1.5K / 2K。",
            "values": [
              "1k",
              "1.5k",
              "2k"
            ],
            "default": "2k",
            "pricing": true,
            "binding": "resolution"
          },
          {
            "name": "count",
            "label": "生成数量",
            "type": "integer",
            "description": "ToAPIs Seedream 5.5 每次仅支持 1 张。",
            "min": 1,
            "max": 1,
            "step": 1,
            "default": 1,
            "binding": "imageCount"
          },
          {
            "name": "outputFormat",
            "label": "输出格式",
            "type": "string",
            "description": "metadata.output_format。透明背景须使用 png。",
            "values": [
              "png",
              "jpeg"
            ],
            "default": "png",
            "binding": "providerOptions.toapis-image.output_format"
          },
          {
            "name": "background",
            "label": "背景",
            "type": "string",
            "description": "metadata.background。图层拆分时不发送。",
            "values": [
              "auto",
              "opaque",
              "transparent"
            ],
            "default": "auto",
            "binding": "providerOptions.toapis-image.background"
          },
          {
            "name": "watermark",
            "label": "水印",
            "type": "boolean",
            "description": "metadata.watermark，默认关闭。",
            "default": false,
            "binding": "providerOptions.toapis-image.watermark"
          },
          {
            "name": "layerDecomposition",
            "label": "图层拆分",
            "type": "boolean",
            "description": "开启后拆分唯一一张参考图为底图与多个透明 PNG 图层，可省略 prompt。",
            "default": false,
            "binding": "providerOptions.toapis-image.layer_decomposition"
          },
          {
            "name": "layerSize",
            "label": "拆层尺寸",
            "type": "string",
            "description": "图层拆分时的尺寸，默认 auto 保持原图比例。",
            "values": [
              "auto",
              "1K",
              "1.5K",
              "2K"
            ],
            "default": "auto",
            "binding": "providerOptions.toapis-image.layer_size"
          },
          {
            "name": "maxImages",
            "label": "最多图层数",
            "type": "integer",
            "description": "layer_decomposition_options.max_images，含底图，2 到 100。",
            "min": 2,
            "max": 100,
            "step": 1,
            "default": 30,
            "binding": "providerOptions.toapis-image.max_images"
          },
          {
            "name": "textRecognition",
            "label": "文字识别",
            "type": "integer",
            "description": "layer_decomposition_options.text_recognition 的布尔开关（0/1）。",
            "values": [
              0,
              1
            ],
            "default": 0,
            "binding": "providerOptions.toapis-image.text_recognition"
          },
          {
            "name": "sizeOverride",
            "label": "尺寸覆盖",
            "type": "string",
            "description": "高级：直接覆盖 size 字段，如 2K、1K、2048x2048。留空则按画幅与分辨率自动计算。",
            "binding": "providerOptions.toapis-image.size"
          }
        ],
        "create": {
          "method": "POST",
          "path": "/v1/images/generations",
          "contentType": "application/json",
          "body": {
            "model": {
              "$ref": "request.model"
            },
            "prompt": {
              "$omitEmpty": {
                "$ref": "request.prompt"
              }
            },
            "n": 1,
            "quality": {
              "$omitEmpty": ""
            },
            "background": {
              "$omitEmpty": ""
            },
            "size": {
              "$coalesce": [
                {
                  "$ref": "request.providerOptions.toapis-image.size"
                },
                {
                  "$if": {
                    "condition": {
                      "$eq": [
                        {
                          "$ref": "request.providerOptions.toapis-image.layer_decomposition"
                        },
                        true
                      ]
                    },
                    "then": {
                      "$coalesce": [
                        {
                          "$ref": "request.providerOptions.toapis-image.layer_size"
                        },
                        "auto"
                      ]
                    },
                    "else": {
                      "$switch": {
                        "cases": [
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "1:1"
                                  ]
                                }
                              ]
                            },
                            "then": "1024x1024"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "4:3"
                                  ]
                                }
                              ]
                            },
                            "then": "1152x864"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "3:4"
                                  ]
                                }
                              ]
                            },
                            "then": "864x1152"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "16:9"
                                  ]
                                }
                              ]
                            },
                            "then": "1424x800"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "9:16"
                                  ]
                                }
                              ]
                            },
                            "then": "800x1424"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "3:2"
                                  ]
                                }
                              ]
                            },
                            "then": "1248x832"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "2:3"
                                  ]
                                }
                              ]
                            },
                            "then": "832x1248"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "21:9"
                                  ]
                                }
                              ]
                            },
                            "then": "1568x672"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1.5k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "1:1"
                                  ]
                                }
                              ]
                            },
                            "then": "1536x1536"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1.5k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "4:3"
                                  ]
                                }
                              ]
                            },
                            "then": "1792x1344"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1.5k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "3:4"
                                  ]
                                }
                              ]
                            },
                            "then": "1344x1792"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1.5k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "16:9"
                                  ]
                                }
                              ]
                            },
                            "then": "2048x1152"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1.5k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "9:16"
                                  ]
                                }
                              ]
                            },
                            "then": "1152x2048"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1.5k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "3:2"
                                  ]
                                }
                              ]
                            },
                            "then": "1872x1248"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1.5k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "2:3"
                                  ]
                                }
                              ]
                            },
                            "then": "1248x1872"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1.5k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "21:9"
                                  ]
                                }
                              ]
                            },
                            "then": "2352x1008"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "2k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "1:1"
                                  ]
                                }
                              ]
                            },
                            "then": "2048x2048"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "2k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "4:3"
                                  ]
                                }
                              ]
                            },
                            "then": "2368x1776"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "2k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "3:4"
                                  ]
                                }
                              ]
                            },
                            "then": "1776x2368"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "2k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "16:9"
                                  ]
                                }
                              ]
                            },
                            "then": "2816x1584"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "2k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "9:16"
                                  ]
                                }
                              ]
                            },
                            "then": "1584x2816"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "2k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "3:2"
                                  ]
                                }
                              ]
                            },
                            "then": "2496x1664"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "2k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "2:3"
                                  ]
                                }
                              ]
                            },
                            "then": "1664x2496"
                          },
                          {
                            "when": {
                              "$and": [
                                {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "2k"
                                  ]
                                },
                                {
                                  "$eq": [
                                    {
                                      "$ref": "request.aspectRatio"
                                    },
                                    "21:9"
                                  ]
                                }
                              ]
                            },
                            "then": "3136x1344"
                          }
                        ],
                        "default": {
                          "$switch": {
                            "cases": [
                              {
                                "when": {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1k"
                                  ]
                                },
                                "then": "1K"
                              },
                              {
                                "when": {
                                  "$eq": [
                                    {
                                      "$lower": {
                                        "$trim": {
                                          "$ref": "request.resolution"
                                        }
                                      }
                                    },
                                    "1.5k"
                                  ]
                                },
                                "then": "1.5K"
                              }
                            ],
                            "default": "2K"
                          }
                        }
                      }
                    }
                  }
                }
              ]
            },
            "image_urls": {
              "$omitEmpty": {
                "$map": {
                  "from": {
                    "$ref": "request.images"
                  },
                  "as": "media",
                  "in": {
                    "$ref": "media.value"
                  }
                }
              }
            },
            "layer_decomposition": {
              "$omitEmpty": {
                "$ref": "request.providerOptions.toapis-image.layer_decomposition"
              }
            },
            "layer_decomposition_options": {
              "$omitEmpty": {
                "$if": {
                  "condition": {
                    "$eq": [
                      {
                        "$ref": "request.providerOptions.toapis-image.layer_decomposition"
                      },
                      true
                    ]
                  },
                  "then": {
                    "max_images": {
                      "$coalesce": [
                        {
                          "$ref": "request.providerOptions.toapis-image.max_images"
                        },
                        30
                      ]
                    },
                    "text_recognition": {
                      "$coalesce": [
                        {
                          "$ref": "request.providerOptions.toapis-image.text_recognition"
                        },
                        false
                      ]
                    }
                  },
                  "else": null
                }
              }
            },
            "optimize_prompt_options": {
              "$omitEmpty": {
                "$ref": "request.providerOptions.toapis-image.optimize_prompt_options"
              }
            },
            "metadata": {
              "$merge": [
                {
                  "output_format": {
                    "$coalesce": [
                      {
                        "$ref": "request.providerOptions.toapis-image.output_format"
                      },
                      "png"
                    ]
                  }
                },
                {
                  "background": {
                    "$omitEmpty": {
                      "$if": {
                        "condition": {
                          "$eq": [
                            {
                              "$ref": "request.providerOptions.toapis-image.layer_decomposition"
                            },
                            true
                          ]
                        },
                        "then": null,
                        "else": {
                          "$coalesce": [
                            {
                              "$ref": "request.providerOptions.toapis-image.background"
                            },
                            "auto"
                          ]
                        }
                      }
                    }
                  }
                },
                {
                  "watermark": {
                    "$coalesce": [
                      {
                        "$ref": "request.providerOptions.toapis-image.watermark"
                      },
                      false
                    ]
                  }
                },
                {
                  "response_format": "url"
                }
              ]
            }
          }
        },
        "poll": {
          "method": "GET",
          "path": "/v1/images/generations/{{taskId}}"
        },
        "response": {
          "taskId": {
            "$coalesce": [
              {
                "$ref": "response.id"
              },
              {
                "$ref": "response.task_id"
              },
              {
                "$ref": "taskId"
              }
            ]
          },
          "status": {
            "$coalesce": [
              {
                "$ref": "response.status"
              },
              "pending"
            ]
          },
          "message": {
            "$coalesce": [
              {
                "$ref": "response.error.message"
              },
              {
                "$ref": "response.error"
              },
              {
                "$ref": "response.message"
              }
            ]
          },
          "images": {
            "$coalesce": [
              {
                "$ref": "response.result.data"
              },
              {
                "$ref": "response.data"
              }
            ]
          },
          "errorPaths": [
            "error.code",
            "error.message",
            "error"
          ],
          "resultEphemeral": true
        }
      }
    ]
  },
  "documentation": "<当前插件的完整 documentation，由 README.md 与 docs/interface.md 拼接而成；为避免 JSON 递归，此处不重复展开正文。>"
}
```
<!-- YINGCE_MANIFEST_CONTRACT_END -->
