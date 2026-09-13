# 보안 정책

## 지원 버전

`main` 브랜치(최신 커밋)만 보안 수정을 받는다. 태그 push로 릴리스 바이너리를 배포하지만,
과거 태그별 유지보수 브랜치나 보안 수정 백포트는 제공하지 않는다.

## 취약점 보고

**공개 Issue로 보고하지 마십시오.**

GitHub의 비공개 취약점 보고를 사용하십시오:
[Security → Report a vulnerability](https://github.com/inininax/go-fiber-starter/security/advisories/new)

보고 시 포함할 정보:

- 영향을 받는 커밋/브랜치
- 재현 절차(가능하면 최소 재현 코드 또는 curl 예제)
- 영향 범위 평가(인증 우회, 정보 노출, DoS 등)

### 응답 목표

| 단계 | 목표 시간 |
|---|---|
| 접수 확인 | 72시간 이내 |
| 심각도 판정 및 진행 계획 | 7일 이내 |
| 수정 배포 | 심각도에 따라 즉시~다음 주기 |

심각도 판정은 [CVSS v3.1](https://www.first.org/cvss/) 기준을 참고한다.

## 보안 설계 요약

이 프로젝트에 이미 적용된 통제와 운영 전제는 다음과 같다. 보고 전 확인하면 중복을 줄일 수 있다.

- JWT: HS256 강제(`WithValidMethods`), 만료 필수, 시크릿 ≥32바이트 fail-fast
- 자격증명: constant-time 비교, prod에서 데모 기본값 시작 차단
- 전송: helmet 보안 헤더, CORS(prod 와일드카드 거부), rate limit(전역 + 로그인 전용), body limit
- 관측: 수집 미들웨어에 도달한 요청의 실제 응답 상태 계측, 알 수 없는 서버 에러의 내부 메시지 미노출
- 의존성: dependabot(go modules + actions 주간). `govulncheck`는 필요 시 수행하는 수동 검사이며 현재 CI 필수 게이트가 아니다.

알려진 한계(스타터 범위)는 README Roadmap과 `docs/authenticator-sketch.md`를 참고하라.
