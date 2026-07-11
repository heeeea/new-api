# Yingxiaotai unified New API integration

This deployment exposes one OpenAI-compatible base URL and one downstream token. Provider credentials remain in New API channels.

## Supported request contracts

- Text: `POST /v1/chat/completions`
- Images: `POST /v1/images/generations` and `POST /v1/images/edits`
- Videos: `POST /v1/videos` or `POST /v1/video/generations`, then poll the matching task URL
- Video understanding: `POST /v1/chat/completions` or `POST /v1/responses`, depending on the model

## Model matrix

| Provider | Yingxiaotai model | New API adapter | State |
| --- | --- | --- | --- |
| Agnes | `agnes-2.0-flash` | OpenAI text | Real-tested |
| Agnes | `agnes-image-2.0-flash`, `agnes-image-2.1-flash` | Agnes image compatibility in OpenAI channel | Real-tested; reference fields now preserved |
| Agnes | `agnes-video-v2.0` | OpenAI video task | Real-tested |
| Alibaba Bailian | `platform::qwen-video-understanding` | Ali compatible chat | Configured |
| Alibaba Bailian | `qwen-image`, `qwen-image-edit-2511` | Ali multimodal image | Contract-tested |
| Alibaba Bailian | `wan2.7-image`, `wan2.7-image-pro` | Ali image | Configured |
| Alibaba Bailian | `wan2.6-i2v-flash`, `wan2.6-i2v`, `wan2.6-t2v` | Ali async video | Contract-tested |
| Alibaba Bailian | `happy-horse-1.1`, `happy-horse-1.0` | Ali reference-video compatibility | Contract-tested |
| Volcengine Ark | `platform::doubao-video-understanding` | Ark chat/responses | Configured |
| Volcengine Ark | `seedream-4-0` | Volcengine image | Configured |
| Volcengine Ark | `doubao-seedance-2-0-mini-260615`, `doubao-seedance-2-0-fast-260128`, `doubao-seedance-2-0-260128` | Volcengine async video | Configured |
| xAI | `grok-4.3` | xAI chat | Configured |
| xAI | `grok-imagine-image`, `grok-imagine-image-quality` | xAI image | Configured |
| xAI | `grok-imagine-video`, `grok-imagine-video-1.5` | OpenAI video task over xAI channel | Contract-tested |
| MiniMax | `minimax-image-01` | MiniMax image | Configured |
| Tencent Hunyuan | `hunyuan-image` | TC3 `TextToImageLite` image adapter | Contract-tested |
| Tencent MPS | `viduq3-turbo`, `viduq3-pro` | TC3 MPS async video | Contract-tested |
| Tencent MPS | `kling-3.0`, `kling-o1`, `kling-omni` | TC3 MPS async video | Contract-tested |
| Tencent MPS | `hailuo-2.3-fast`, `hunyuan-video`, `pixverse-v6` | TC3 MPS async video | Contract-tested |

## Compatibility rules added for Yingxiaotai

- Agnes image requests keep provider-specific `extra_body.image` references and move an incompatible top-level `response_format` into `extra_body.response_format`.
- xAI channels use the OpenAI video task submit/poll adapter.
- HappyHorse aliases are converted into Bailian `input.media` entries with `reference_image` types.
- Tencent MPS credentials are converted into TC3-signed `CreateAigcVideoTask` and `DescribeAigcVideoTask` calls.
- Tencent Hunyuan images are converted into TC3-signed `TextToImageLite` calls and normalized to OpenAI image responses.

## Acceptance boundary

Provider contract tests do not spend paid-provider quota. Agnes is the only provider authorized for real text, image, reference-image, and video generation during integration. Other paid providers must be real-tested by the owner before their channels are considered production-verified.

New API records requests, models, channels, response times, and configured quota consumption. It cannot query every provider's cash balance through the unified token; provider balance synchronization remains provider-specific.
