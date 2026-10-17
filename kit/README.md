# gauss_jordan 기계 평가 키트

커뮤니티 과제 `gauss_jordan`(v0.1)의 평가표 가운데 기계가 대신 확인할 수 있는 부분을 자동으로 돌리는 키트다. 과제 본문 §3.5의 예시와 평가표 1절부터 4절(입력과 오류 처리, 기약 행 사다리꼴, 계수와 행렬식, 역행렬)의 입력과 기대 출력을 그대로 파일로 옮겨 두었다. 제출물을 한 번 돌리면 케이스별 PASS/FAIL과 `result.json` 기록이 나오고, 평가자는 그 기록을 자기 SSH 키로 서명해 피평가자에게 준다. 서명은 서버 없이 누구나 검증할 수 있다.

키트가 대신하지 않는 것도 있다. 사전 확인(norminette, `make` 재링크, `nm -u`), 5절(누수, 넘침 감지와 유리수 구현 설명), 6절(보너스), 그리고 평가자가 즉석에서 만드는 3×3 행렬은 평가표대로 사람이 한다.

## 구성

```
kit/
  README.md                   이 문서
  tests/
    run.sh                    러너. 제출물을 모든 케이스로 돌리고 result.json을 쓴다
    cases/                    입력 파일. 케이스 하나에 파일 하나
    expected/                 기대 표준 출력. <케이스>.<모드>.out
    reference.py              참조 구현. 키트 자체를 검증하는 데만 쓴다
  record/
    sign.sh                   result.json을 평가자 키로 서명한다
    verify.sh                 서명을 검증한다
    allowed_signers.example   검증에 쓰는 allowed_signers 파일의 예
```

모드 이름은 네 가지다. `rref`(플래그 없음), `rank`(`-r`), `det`(`-d`), `inv`(`-i`). 인자 자체가 잘못된 케이스는 `args`로 적는다.

## 평가자가 테스트를 돌리는 법

키트는 제출 저장소 밖에 둔다(저장소에는 과제가 요구하는 파일만 있어야 한다). 제출 저장소에서 빌드한 뒤 러너에 실행 명령을 문자열 하나로 넘긴다.

```
cd <제출 저장소>
make
~/gauss_jordan_kit/tests/run.sh ./gauss_jordan
```

러너는 넘겨받은 명령 뒤에 플래그와 입력 파일 경로를 붙여 실행한다. 즉 `./gauss_jordan -d <파일>` 같은 꼴이다. 일부 케이스는 파일 이름 자리에 `-`를 주고 표준 입력으로 내용을 넣는다. 명령에 공백이 있어도 된다. 예를 들어 `"valgrind -q ./gauss_jordan"`처럼 넘기면 valgrind 아래에서 돌릴 수 있다.

판정 기준은 다음과 같다.

- 정상 케이스: 종료 상태 0이고 표준 출력이 `expected/` 파일과 바이트 단위로 같아야 한다. 각 줄 끝의 개행까지 비교하므로 줄 끝 공백이나 마지막 개행 누락은 FAIL이다.
- 오류 케이스: 종료 상태 1, 표준 출력은 비어 있고, 표준 오류에 `Error`가 들어 있어야 한다.
- 실행이 10초를 넘기면(`timeout` 명령이 있을 때) 그 케이스는 FAIL이다. `TIMEOUT_SEC`으로 바꿀 수 있다.

출력은 케이스마다 한 줄이다.

```
PASS system rref
PASS singular det
FAIL overflow rref
     stderr lacks Error: overflow|
...
summary: passed=76 failed=1 total=77
result: /home/me/gauss_jordan_kit/tests/result.json
```

FAIL 줄 아래에는 기대값과 실제 출력(개행은 `|`로 표시)이 붙는다. 러너의 종료 상태는 전부 통과하면 0, 하나라도 실패하면 1이다. 읽기 권한 없는 파일 케이스는 root로 돌리면 권한 검사가 되지 않아 건너뛴다(SKIP).

환경 변수:

- `RESULT`: 기록 파일 경로. 기본값은 키트의 `tests/result.json`.
- `SUBJECT`: 과제 문서 경로. 머리말(frontmatter)의 `id`와 `version`을 기록에 적는다. 기본값은 키트 상위 폴더의 `gauss_jordan_subject.md`, 없으면 `SUBJECT.md`, 그것도 없으면 `gauss_jordan` / `0.1`.
- `TIMEOUT_SEC`: 케이스 하나의 제한 시간(초). 기본값 10.

러너는 bash와 표준 도구(`cmp`, `grep`, `sed`, `awk`, `date`, `uname`, `sha256sum` 또는 `shasum`)만 쓴다. Python은 필요 없다.

### 케이스 목록

| 케이스 | 입력 | 평가표 항목 |
|---|---|---|
| `system` | §3.5의 3×4 행렬 | 2절 RREF, 3절 `-r` → 3, 비정사각 `-d`/`-i` → Error |
| `singular` | 1 2 3 / 4 5 6 / 7 8 9 | 2절 RREF, `-r` → 2, `-d` → 0, `-i` → `singular` |
| `inv` | 4 7 / 2 6 | `-d` → 10, `-i` → 3/5 -7/10 / -1/5 2/5 |
| `fractions` | 1/2 1/3 / 1/4 1/5 | `-d` → 1/60, `-i` → 12 -20 / -15 30 |
| `swap` | 0 2 1 / 1 0 3 | 첫 피벗 0, RREF, `-r` → 2, 2×3에 `-d` → Error |
| `zero` | 2×3 영행렬 | RREF 그대로, `-r` → 0 |
| `contradiction` | 2 4 -2 2 / 1 2 -1 1 / 3 6 -3 4 | RREF → 1 2 -1 0 / 0 0 0 1 / 0 0 0 0 |
| `one` | 1×1 행렬 5 | RREF → 1, `-d` → 5, `-i` → 1/5 |
| `row`, `col` | 1×3, 3×1 행렬 | 1×n, n×1 정상 처리 |
| `detsign` | 0 1 / 1 0 | 행 교환 홀수 번, `-d` → -1 |
| `unreduced` | 6/4 -10/5 / 3/2 -2 | 기약이 아닌 입력, 출력은 기약이고 0은 `0` |
| `whitespace` | `system`과 같은 행렬, 여분 공백과 마지막 개행 없음 | 1절, 결과가 같아야 함 |
| `*_stdin` | 위 파일을 `-`로 표준 입력에 넣음 | 1절 |
| `overflow` | 10^12 근처 항목 2×2 | 1절 넘침, 네 모드 모두 Error |
| `bad_token`, `bad_suffix`, `bad_dots` | `1 a 3`, `1 2 3x`, `1..2` | 숫자가 아닌 항목 |
| `zero_denominator`, `negative_denominator`, `empty_numerator` | `1/0`, `3/-4`, `/4` | 분모와 분자 오류 |
| `bad_header`, `bad_header_three`, `zero_rows`, `zero_cols`, `too_many_rows` | `3`, `3 3 3`, `0 3`, `3 0`, `101 1` | 첫 줄 오류 |
| `dimension_mismatch`, `dimension_mismatch_cols` | 줄 부족, 항목 초과 | 차원 불일치 |
| `empty` | 빈 파일 | 1절 |
| `no_args`, `three_args`, `unknown_flag`, `two_flags`, `missing_file`, `unreadable_file` | 인자와 파일 오류 | 1절 |

## 결과 기록 result.json

러너가 끝나면 아래 꼴의 기록을 쓴다. 키 순서는 고정이라 같은 환경에서 같은 결과면 같은 바이트가 나온다(시각만 다르다).

```json
{
  "subject": {"id": "gauss_jordan", "version": "0.1"},
  "submission": {"commit": "a361dc2caa03796334b59caaf09a22aab5d60b44"},
  "environment": {"uname": "Linux 6.8.0 x86_64", "compiler": "cc (Ubuntu 13.3.0) 13.3.0", "runner_sha256": "..."},
  "cases": [
    {"name": "system", "mode": "rref", "pass": true},
    ...
  ],
  "summary": {"passed": 77, "failed": 0, "total": 77},
  "timestamp": "2026-10-16T12:00:00Z"
}
```

- `submission.commit`은 러너를 실행한 현재 폴더가 Git 저장소면 `git rev-parse HEAD`, 아니면 `null`이다. 제출 저장소 안에서 실행해야 커밋이 기록된다.
- `environment.runner_sha256`은 `run.sh` 자체의 해시다. 어떤 러너로 돌렸는지 나중에 대조할 수 있다.
- 기록에는 42 슬롯이나 평가 일정이 들어가지 않는다(과제 §VII).

## 서명과 검증, 서버 없이

과제 §VII은 평가자가 평가 결과를 자기 키로 서명해 피평가자에게 주고, 서명은 서버 없이 검증할 수 있어야 한다고 적는다. 이 키트는 그 형식으로 OpenSSH의 SSH 서명(`ssh-keygen -Y`)을 쓴다. 42 카뎃은 이미 Git용 SSH 키를 갖고 있으니 새 도구가 필요 없고, 검증하는 쪽은 평가자의 공개키 한 줄만 있으면 된다. **OpenSSH 8.0 이상**이어야 `-Y sign`과 `-Y verify`가 있다. `ssh -V`로 확인한다.

### 평가자: 키와 신원

ed25519 키가 없으면 하나 만든다. 신원(identity)은 `<intra 로그인>@<캠퍼스>` 꼴을 권한다. 예: `jkim2@42gyeongsan`.

```
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519
```

### 평가자: 서명

```
record/sign.sh tests/result.json jkim2@42gyeongsan
```

`tests/result.json.sig`가 생긴다. 기본 키는 `~/.ssh/id_ed25519`이고 세 번째 인자로 다른 키를 줄 수 있다. 서명 이름공간(namespace)은 `community-subject`로 고정이라 같은 키로 만든 다른 용도의 SSH 서명과 섞이지 않는다. 스크립트는 끝에 검증자가 쓸 `allowed_signers` 한 줄도 출력하는데, 피평가자에게 `result.json`, `result.json.sig`와 함께 이 줄을 주면 된다.

### 누구나: 검증

검증하는 사람은 신뢰하는 평가자의 공개키를 `allowed_signers` 파일에 한 줄씩 적는다. 형식은 `record/allowed_signers.example`에 있다.

```
jkim2@42gyeongsan namespaces="community-subject" ssh-ed25519 AAAAC3Nza...
```

자기 키로 만들려면 다음 한 줄이면 된다.

```
(printf 'jkim2@42gyeongsan namespaces="community-subject" '; cut -d' ' -f1,2 ~/.ssh/id_ed25519.pub) > allowed_signers
```

그다음 검증한다.

```
record/verify.sh tests/result.json allowed_signers jkim2@42gyeongsan
```

종료 상태 0이면 그 기록은 그 신원이 서명한 바이트 그대로다. 기록의 한 글자라도 바뀌면(예: `"passed": 77`을 `78`로) 검증은 실패하고 종료 상태는 0이 아니다. 신원이 다르거나 `allowed_signers`에 없는 키로 서명되었을 때도 실패한다. 네트워크 연결이나 서버는 어디에도 필요 없다.

## 20초 데모

평가자가 자기 키와 `allowed_signers`를 준비해 두었다면 세 명령이면 된다. 제출 저장소 안에서 실행한다고 가정한다(키트는 `~/gauss_jordan_kit`에 풀어 두었다고 가정).

```
~/gauss_jordan_kit/tests/run.sh ./gauss_jordan
~/gauss_jordan_kit/record/sign.sh ~/gauss_jordan_kit/tests/result.json jkim2@42gyeongsan
~/gauss_jordan_kit/record/verify.sh ~/gauss_jordan_kit/tests/result.json allowed_signers jkim2@42gyeongsan
```

제출물이 없어도 참조 구현으로 같은 흐름을 볼 수 있다. 키트 폴더에서:

```
tests/run.sh "python3 tests/reference.py"
record/sign.sh tests/result.json jkim2@42gyeongsan
record/verify.sh tests/result.json allowed_signers jkim2@42gyeongsan
```

## 참조 구현 reference.py

`tests/reference.py`는 과제 §3.1부터 §3.4를 Python 3 표준 라이브러리만으로 옮긴 것이다. 키트의 기대 출력이 과제 본문과 맞는지 확인하는 용도이지 과제의 모범 답안이 아니다(과제는 C로 작성한다). 키트를 고쳤다면 `tests/run.sh "python3 tests/reference.py"`로 전부 PASS인지 확인한다.

넘침은 다음 규칙으로 흉내 낸다. 입력의 분자와 분모, 그리고 모든 연산의 결과(기약분수)에서 분자의 절댓값이나 분모가 2^63 - 1을 넘으면 Error다. 이는 과제가 허용하는 가장 너그러운 읽기다. 참조 구현이 Error를 내는 입력은 어떤 올바른 제출물도 Error를 내야 하고, 참조 구현이 답을 내는 입력에서 제출물이 (곱셈 중간값 때문에) Error를 내는 것은 과제가 허용한다. 키트의 케이스는 넘침 케이스를 빼면 값이 모두 작아서 이 차이가 드러나지 않는다.

입력 문법은 과제 §3.2를 엄격하게 읽었다. 첫 줄 뒤에는 정확히 `m`개의 줄이 와야 하므로 뒤에 빈 줄이 하나 더 있으면 Error다. 항목 구분은 하나 이상의 공백이고 줄 끝 공백은 무시한다. 부호는 `+`와 `-`를 받는다. 이런 경계는 키트 케이스에 넣지 않았다.

## 키트 자체 검증 기록

- `tests/run.sh "python3 tests/reference.py"`: 77 케이스 전부 PASS(root가 아닌 사용자로 돌리면 `unreadable_file`이 더해져 78).
- 서명 흐름: 임시 ed25519 키로 서명하고 검증해 종료 상태 0을 확인했다. 기록의 한 바이트를 바꾼 사본은 검증에 실패했다(`Signature verification failed: incorrect signature`).
