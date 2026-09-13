package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"go.yaml.in/yaml/v3" // openapi.yaml 파싱 전용. 표준 라이브러리에 YAML 지원이 없어 direct 의존으로 추가했다(설계 승인).

	"go-fiber-starter/api"
)

// TestOpenAPI_NoRouteDrift는 api/openapi.yaml과 실제 등록 라우트의 일치를 양방향으로 강제한다.
// 라우트를 추가/제거하면서 이 파일을 갱신하지 않으면 CI가 실패한다(스펙 드리프트 방지).
func TestOpenAPI_NoRouteDrift(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.AuthEnabled = true // 스펙에 문서된 /api/v1/auth/login까지 포함해 비교
	cfg.AuthJWTSecret = strings.Repeat("s", 32)
	app := newTestApp(t, cfg)

	spec := parseSpecPaths(t)
	code := collectCodeRoutes(t, app)

	var specOnly, codeOnly []string
	for path, specMethods := range spec {
		codeMethods, ok := code[path]
		if !ok {
			specOnly = append(specOnly, path)
			continue
		}
		for m := range specMethods {
			if !codeMethods[m] {
				specOnly = append(specOnly, fmt.Sprintf("%s %s", m, path))
			}
		}
		for m := range codeMethods {
			if !specMethods[m] {
				codeOnly = append(codeOnly, fmt.Sprintf("%s %s", m, path))
			}
		}
	}
	for path := range code {
		if _, ok := spec[path]; !ok {
			codeOnly = append(codeOnly, path)
		}
	}

	if len(specOnly) > 0 || len(codeOnly) > 0 {
		t.Fatalf("OpenAPI 스펙과 실제 라우트가 불일치한다.\n  spec-only (스펙에만 있음): %v\n  code-only (코드에만 있음): %v\n  -> api/openapi.yaml 또는 라우트를 갱신하라",
			specOnly, codeOnly)
	}

	// 서빙되는 스펙 원문이 파싱 가능하고 비어 있지 않은지도 함께 확인한다.
	raw, err := api.FS.ReadFile("openapi.yaml")
	if err != nil || len(raw) == 0 {
		t.Fatalf("embedded openapi.yaml unreadable or empty: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET /openapi.yaml: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /openapi.yaml: want 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/yaml") {
		t.Fatalf("openapi.yaml content-type: want application/yaml, got %q", ct)
	}
}

// parseSpecPaths는 YAML에서 paths와 각 path의 HTTP 메서드 집합을 추출한다.
func parseSpecPaths(t *testing.T) map[string]map[string]bool {
	t.Helper()
	raw, err := api.FS.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}
	var doc struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	out := make(map[string]map[string]bool, len(doc.Paths))
	for path, ops := range doc.Paths {
		methods := make(map[string]bool, len(ops))
		for key := range ops {
			if isHTTPMethodKey(key) {
				methods[strings.ToUpper(key)] = true
			}
		}
		if len(methods) > 0 {
			out[normalizeSpecPath(path)] = methods
		}
	}
	return out
}

// isHTTPMethodKey는 paths 아래 operation 키(get/post/...)와 parameters/$ref 같은
// 비-operation 키를 구분한다.
func isHTTPMethodKey(key string) bool {
	switch strings.ToUpper(key) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

func normalizeSpecPath(p string) string { return trimTrailingSlash(p) }

// collectCodeRoutes는 등록된 라우트를 fiber 경로 표기(":id")에서 OpenAPI 표기("{id}")로
// 변환해 수집한다.
//
// 제외 규칙:
//   - HEAD: fiber v3는 GET마다 HEAD 라우트를 자동 생성한다(autoHead, startupProcess 시점).
//     OpenAPI는 자동 생성 메서드를 선언하지 않으므로 HEAD는 비교에서 항상 제외한다.
//   - use 미들웨어 라우트: GetRoutes(true)로 필터링된다.
func collectCodeRoutes(t *testing.T, app *fiber.App) map[string]map[string]bool {
	t.Helper()
	out := make(map[string]map[string]bool)
	for _, r := range app.GetRoutes(true) {
		if r.Method == http.MethodHead {
			continue // 위 제외 규칙 참고
		}
		path := toOpenAPIPath(r.Path)
		if out[path] == nil {
			out[path] = make(map[string]bool)
		}
		out[path][r.Method] = true
	}
	return out
}

// toOpenAPIPath는 fiber Path(원본, 후행 "/" 보존, ":param")를 OpenAPI 표기로 바꾼다.
func toOpenAPIPath(p string) string {
	parts := strings.Split(trimTrailingSlash(p), "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func trimTrailingSlash(p string) string {
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		return p[:len(p)-1]
	}
	return p
}
