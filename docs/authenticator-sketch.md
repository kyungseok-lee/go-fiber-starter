# DB 기반 Authenticator 스케치

**문서-only, 구현 아님.** env 기반 demo authenticator(`internal/modules/auth/service.go`)를
실제 사용자 스토어로 교체할 때의 참고 스케치다. 이 파일은 구현 계약이 아니며, 실제 교체 시
`internal/modules/task/` 모듈 패턴에 맞춰 user 모듈로 확장해야 한다.

핵심 요건:

- 비밀번호는 평문 저장 금지. bcrypt 등 적응형 해시로 저장한다.
- 타이밍 평준화: 사용자 미존재 시에도 더미 해시 대조를 수행해 "사용자 없음"과
  "비밀번호 틀림"의 응답 시간 차이를 제거한다(사용자 열거 방지).
- `Authenticator` 인터페이스는 그대로 유지하고 `internal/router/router.go`의 `NewDemoAuthenticator` 조립을 교체한다.
- `internal/config/config.go`의 prod 데모 기본값 검증도 새 인증 방식에 맞게 수정한다. 현재는 조립만 교체해도
  `AUTH_DEMO_USERNAME`/`AUTH_DEMO_PASSWORD` 검사가 계속 실행된다.

```go
// bcrypt 기반 dbAuthenticator 스케치 (문서-only, 컴파일 보장 없음)
type dbAuthenticator struct{ users UserRepository }

var dummyHash = []byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5B0X0QG0eN6Rl1s0dE9uY3vTnM0Ku") // 고정 더미

func (d *dbAuthenticator) Verify(ctx context.Context, username, password string) (auth.Identity, error) {
	u, err := d.users.FindByUsername(ctx, username) // not found여도 에러로 단락 평가하지 않는다
	hash := dummyHash
	if err == nil {
		hash = u.PasswordHash
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil || err != nil {
		return auth.Identity{}, apperror.ErrInvalidCredential // 항상 동일 오류/유사 시간 반환
	}
	return auth.Identity{Username: u.Username}, nil
}
```

마이그레이션: users 테이블은 `make migrate-new name=add_users`로 postgres/mysql SQL을,
sqlite AutoMigrate 목록에는 User 모델을 함께 추가한다(이원화 정책은 AGENTS.md 참고).
