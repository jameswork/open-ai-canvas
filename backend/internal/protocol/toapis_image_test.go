package protocol

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// toAPIsSourceAdapter 直接从源码 manifest 加载，而不是从 .yingce-plugin ZIP。
// 这样插件无论作为官方包留在 plugin-packages/，还是改成后台上传的普通插件，
// 这份线协议测试都能运行（上传模式只需移走 ZIP，保留源码目录）。
func toAPIsSourceAdapter(t *testing.T) Adapter {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "plugin-packages", "toapis-image", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	adapters, err := LoadInstalledProviders(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, adapter := range adapters {
		if adapter.Metadata().ID == "toapis-image" {
			return adapter
		}
	}
	t.Fatal("provider toapis-image missing from source manifest")
	return nil
}

// ToAPIs 是异步任务接口，且要求参考图为公网 URL。这两个身份约束先钉死。
func TestToAPIsImageIdentity(t *testing.T) {
	adapter := toAPIsSourceAdapter(t)
	if !adapter.Metadata().RequiresPublicMediaURLs {
		t.Fatal("ToAPIs 只接受公网 URL 参考图，requiresPublicMediaUrls 必须为 true")
	}
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "doubao-seedream-5-5", Prompt: "a cat", Capability: CapabilityImage,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if create.Method != "POST" || create.Path != "/v1/images/generations" {
		t.Fatalf("create = %s %s", create.Method, create.Path)
	}
	poll, err := adapter.BuildPoll(context.Background(), PollContext{TaskID: "tsk_1"})
	if err != nil {
		t.Fatal(err)
	}
	if poll.Method != "GET" || poll.Path != "/v1/images/generations/tsk_1" {
		t.Fatalf("poll = %s %s", poll.Method, poll.Path)
	}
}

// 回归测试：宿主 completeCommonImageRequest 会把 /images/generations 识别成
// OpenAI 线格式，往顶层注入 quality/background。ToAPIs seedream-5.5 只认
// metadata 内的 background，顶层这两个字段会导致"参数 body 无效"。
// manifest 里用 $omitEmpty 空声明覆盖了这两个键，这里验证 suppression 生效。
func TestToAPIsImageSendsOnlyDocumentedFields(t *testing.T) {
	adapter := toAPIsSourceAdapter(t)
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "doubao-seedream-5-5", Prompt: "a cat", Capability: CapabilityImage,
		AspectRatio: "1:1", Resolution: "2k", Quality: "auto",
		ImageParameters: map[string]any{"quality": "auto", "transparentBackground": false},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)

	if body["model"] != "doubao-seedream-5-5" || body["prompt"] != "a cat" || body["n"] != float64(1) {
		t.Fatalf("body core fields = %#v", body)
	}
	// 上游明确拒绝或不支持的字段，一个都不能出现在顶层。
	for _, forbidden := range []string{
		"quality", "background", "seed", "stream", "stream_options", "tools",
		"sequential_image_generation", "sequential_image_generation_options",
	} {
		if _, present := body[forbidden]; present {
			t.Fatalf("body = %#v, 顶层不得出现 %s", body, forbidden)
		}
	}
	// 高级配置只能在 metadata 内。
	metadata, ok := body["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata = %#v", body["metadata"])
	}
	if metadata["output_format"] != "png" || metadata["response_format"] != "url" || metadata["watermark"] != false || metadata["background"] != "auto" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

// Seedream 5.5 的 size 由 resolution 档位与 aspectRatio 共同决定，
// 逐档钉死官方像素表，防止比例/档位组合落到错误尺寸。
func TestToAPIsImageSizeMapping(t *testing.T) {
	adapter := toAPIsSourceAdapter(t)
	cases := []struct {
		ratio, resolution, want string
	}{
		{"1:1", "1k", "1024x1024"}, {"1:1", "1.5k", "1536x1536"}, {"1:1", "2k", "2048x2048"},
		{"16:9", "1k", "1424x800"}, {"16:9", "2k", "2816x1584"},
		{"9:16", "1.5k", "1152x2048"}, {"21:9", "2k", "3136x1344"},
		{"auto", "2k", "2K"},
	}
	for _, tc := range cases {
		t.Run(tc.ratio+"@"+tc.resolution, func(t *testing.T) {
			create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
				Model: "doubao-seedream-5-5", Prompt: "x", Capability: CapabilityImage,
				AspectRatio: tc.ratio, Resolution: tc.resolution,
			}})
			if err != nil {
				t.Fatal(err)
			}
			body := manifestTestBody(t, create)
			if body["size"] != tc.want {
				t.Fatalf("size = %v, 期望 %q", body["size"], tc.want)
			}
		})
	}
}

// 参考图必须走 image_urls 公网数组；sizeOverride 允许高级用户直接覆盖尺寸。
func TestToAPIsImageReferencesAndSizeOverride(t *testing.T) {
	adapter := toAPIsSourceAdapter(t)
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "doubao-seedream-5-5", Prompt: "x", Capability: CapabilityImage,
		AspectRatio: "1:1", Resolution: "2k",
		Images:          []MediaReference{{URL: "https://example.com/a.png"}, {URL: "https://example.com/b.png"}},
		ProviderOptions: map[string]map[string]any{"toapis-image": {"size": "2560x1440"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	if body["size"] != "2560x1440" {
		t.Fatalf("sizeOverride 未生效: size = %v", body["size"])
	}
	urls, ok := body["image_urls"].([]any)
	if !ok || len(urls) != 2 || urls[0] != "https://example.com/a.png" {
		t.Fatalf("image_urls = %#v", body["image_urls"])
	}
}

// 图层拆分：size 取 layer_size（auto），background 在 metadata 内被省略，
// layer_decomposition_options 携带 max_images / text_recognition。
func TestToAPIsImageLayerDecomposition(t *testing.T) {
	adapter := toAPIsSourceAdapter(t)
	create, err := adapter.BuildCreate(context.Background(), RequestContext{Request: GenerationRequest{
		Model: "doubao-seedream-5-5", Capability: CapabilityImage,
		Images: []MediaReference{{URL: "https://example.com/input.png"}},
		ProviderOptions: map[string]map[string]any{"toapis-image": {
			"layer_decomposition": true, "layer_size": "auto", "max_images": 10, "text_recognition": true,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	body := manifestTestBody(t, create)
	if body["layer_decomposition"] != true || body["size"] != "auto" {
		t.Fatalf("layer body = %#v", body)
	}
	options, ok := body["layer_decomposition_options"].(map[string]any)
	if !ok || manifestInt(options["max_images"]) != 10 || options["text_recognition"] != true {
		t.Fatalf("layer_decomposition_options = %#v", body["layer_decomposition_options"])
	}
	metadata, _ := body["metadata"].(map[string]any)
	if _, present := metadata["background"]; present {
		t.Fatalf("图层拆分必须在 metadata 内省略 background: metadata = %#v", metadata)
	}
}

// 异步任务：create 返回 pending，poll 完成后从 result.data 提取图片并保留
// z_index 等图层元数据；失败时从 error 读取原因。
func TestToAPIsImageAsyncTaskParsing(t *testing.T) {
	adapter := toAPIsSourceAdapter(t)

	created, err := adapter.ParseCreate(context.Background(), []byte(`{"id":"tsk_1","status":"pending","object":"generation.task"}`))
	if err != nil || created.TaskID != "tsk_1" || created.Status != StatusPending {
		t.Fatalf("created = %#v err=%v", created, err)
	}

	done, err := adapter.ParsePoll(context.Background(), PollContext{TaskID: "tsk_1"}, []byte(`{
		"id":"tsk_1","status":"completed",
		"result":{"data":[
			{"url":"https://files.toapis.com/base.png","size":"1024x1024","z_index":0,"name":"base"},
			{"url":"https://files.toapis.com/layer.png","z_index":1,"bounding_box":[0,0,10,10],"output_format":"png"}
		]}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusSucceeded || done.Result == nil || len(done.Result.Images) != 2 {
		t.Fatalf("done = %#v", done)
	}
	if done.Result.Images[0].Metadata["z_index"] == nil || done.Result.Images[1].Metadata["bounding_box"] == nil {
		t.Fatalf("图层元数据未保留: %#v", done.Result.Images)
	}

	failed, err := adapter.ParsePoll(context.Background(), PollContext{TaskID: "tsk_1"}, []byte(`{"status":"failed","error":{"code":"invalid_request","message":"参数 body 无效."}}`))
	if err != nil || failed.Status != StatusFailed || failed.Message != "参数 body 无效." {
		t.Fatalf("failed = %#v err=%v", failed, err)
	}
}

// 保证 manifest 的 documentation 与源码 README + interface.md 一致，
// 防止上传包后后台文档与线协议脱节。
func TestToAPIsImageDocumentationInSync(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "plugin-packages", "toapis-image", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Documentation string `json:"documentation"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Documentation) == 0 {
		t.Fatal("manifest.documentation 为空，需运行 embed-documentation.mjs")
	}
}
