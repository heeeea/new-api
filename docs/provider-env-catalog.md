# ENV 渠道目录

New API 可在启动时读取根目录 `.env`，自动创建或更新带 `env-managed` 标签的渠道。手工创建的渠道不会被覆盖。

## 使用方式

1. 设置 `ENV_CHANNEL_BOOTSTRAP_ENABLED=true`。
2. 只填写需要启用的平台凭据；Key 留空的渠道不会创建。
3. 重启 New API。支持 `/models` 的渠道会定时发现并补充上游模型。
4. 在后台「渠道」页测试渠道，在「模型」页确认模型和价格。

## 已提供的官方渠道槽位

| 平台 | 凭据变量 | 额外必填项 |
| --- | --- | --- |
| OpenAI / Sora | `OPENAI_API_KEY` | 无 |
| Anthropic | `ANTHROPIC_API_KEY` | 无 |
| Azure OpenAI | `AZURE_OPENAI_API_KEY` | Base URL、部署名模型列表 |
| Gemini | `GEMINI_API_KEY` | 无 |
| Vertex AI | `VERTEX_AI_CREDENTIALS` | 区域 JSON |
| AWS Bedrock | `AWS_BEDROCK_CREDENTIALS` | `AccessKey\|SecretKey\|Region` |
| DeepSeek | `DEEPSEEK_API_KEY` | 无 |
| xAI | `XAI_API_KEY` | 无 |
| 阿里百炼 | `DASHSCOPE_API_KEY` | 无 |
| 火山方舟 / 豆包视频 | `VOLCENGINE_API_KEY` | 无 |
| 腾讯混元 | `TENCENT_HUNYUAN_CREDENTIALS` | `AppId\|SecretId\|SecretKey` |
| MiniMax | `MINIMAX_API_KEY` | 无 |
| Moonshot | `MOONSHOT_API_KEY` | 无 |
| 智谱 | `ZHIPU_API_KEY` | 无 |
| 百度千帆 | `BAIDU_QIANFAN_API_KEY` | 无 |
| 讯飞星火 | `IFLYTEK_CREDENTIALS` | `APPID\|APISecret\|APIKey` |
| 360 智脑 | `QIHOO360_API_KEY` | 无 |
| OpenRouter | `OPENROUTER_API_KEY` | 无 |
| Perplexity | `PERPLEXITY_API_KEY` | 无 |
| 零一万物 | `LINGYI_API_KEY` | 无 |
| Cohere | `COHERE_API_KEY` | 无 |
| Jina | `JINA_API_KEY` | 无 |
| Cloudflare Workers AI | `CLOUDFLARE_API_TOKEN` | Account ID |
| SiliconFlow | `SILICONFLOW_API_KEY` | 无 |
| Mistral | `MISTRAL_API_KEY` | 无 |
| Coze | `COZE_API_KEY` | Bot ID |
| Kling | `KLING_API_KEY` | 无 |
| 即梦 | `JIMENG_CREDENTIALS` | `AccessKey\|SecretKey` |
| Vidu | `VIDU_API_KEY` | 无 |
| Replicate | `REPLICATE_API_TOKEN` | 无 |

另外提供 3 个 `OPENAI_COMPATIBLE_*` 槽位，可接 Agnes 等 OpenAI 兼容服务；Key、Base URL、模型列表必须一起填写。

## 价格规则

- `ENV_PRICING_BOOTSTRAP_ENABLED=true` 时，启动会从 New API 官方维护的倍率预设补齐缺失的输入、输出、缓存和固定价格。
- 默认 `ENV_PRICING_BOOTSTRAP_OVERWRITE=false`，不会覆盖管理员在后台手工维护的价格。
- 价格源不可访问时只记录错误并继续启动，原价格保持不变。
- 图片、视频、音频常按张、秒、分辨率或字符计费，不能统一伪装成输入/输出 token 价；对应适配器会按请求参数追加动态计费倍率。
- New API 官方倍率预设是持续维护的聚合配置，不等同于每个厂家账单承诺。最终实际支出仍以厂家控制台账单为准。

## 边界

这里覆盖的是当前 New API 已有原生适配器和 OpenAI 兼容协议。没有公开 API、没有稳定协议，或使用厂家专有异步协议且 New API 尚无适配器的平台，不能只填一个 Key 就自动接入；这类平台仍需单独开发适配器。
