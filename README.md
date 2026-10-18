# 커뮤니티 과제 색인

카뎃이 만든 과제를 카뎃이 평가하고, 그 기록은 카뎃이 가진다. 이 저장소에는 과제 규격(`SPEC.md`), 기계 평가 키트(`kit/`), 첫 과제(`subjects/gauss_jordan/`), 서명된 평가 기록(`records/`), 그리고 이 모든 것을 읽어 정적 사이트를 만드는 생성기(`cmd/index`)가 들어 있다. 관리자도 서버도 없다. 과제도 기록도 Git에 평문으로 있고, 색인은 누구나 명령 하나로 다시 만든다.

## 다시 만들기

```
git clone <이 저장소> community-subject && cd community-subject
make site        # site/public/ 에 HTML을 쓴다. 생성하면서 모든 기록의 서명을 검증한다
make serve       # http://localhost:8000/ 에서 미리 본다
```

필요한 것은 Go(1.26 이상)와 OpenSSH 8.0 이상(`ssh-keygen -Y`)뿐이다. 의존성은 Markdown 렌더러 goldmark 하나다. JavaScript는 없고 CSS 파일 하나다. 같은 저장소에서 같은 명령을 돌리면 같은 사이트가 나온다.

생성되는 페이지:

- `site/public/index.html`: 이 프로젝트가 무엇인지, 원칙 셋, 과제 목록, 기록 페이지 목록
- `site/public/subjects/<id>/index.html`: 프런트매터 표, 과제 본문, 기계 평가 키트 안내, 평가 흐름, 완료·평가 건수
- `site/public/records/<피평가자>/index.html`: 피평가자 한 사람의 "내 기록". 기록마다 종류(기계·사람), 과제와 버전, 날짜, 서명자 신원, 검증 결과(OK·FAIL)

과제 페이지에는 건수만 적고 평가자와 피평가자의 쌍은 드러내지 않는다. 쌍은 피평가자의 기록 페이지에만 있다.

## 구성

```
SPEC.md                 커뮤니티 과제 규격 v0.1
kit/                    기계 평가 키트: tests/run.sh, record/sign.sh, record/verify.sh
subjects/<id>/          과제 하나. SUBJECT.md(프런트매터 + 본문)와 선택적 tests/
records/<피평가자>/      서명된 기록. <이름>.json 과 <이름>.json.sig 가 짝이다
allowed_signers         검증에 쓰는 신뢰 서명자 목록(OpenSSH allowed_signers 형식)
demo/keys/              데모 키의 공개키. 개인키는 저장소에 없다. 아래 참고
cmd/index/              생성기. 템플릿(html/template)과 CSS를 포함한다
scripts/verify-all.sh   모든 기록을 검증한다(make verify)
site/public/            생성 결과. Git에 넣지 않는다
```

## 과제 추가하기

1. `subjects/<id>/SUBJECT.md`를 만든다. 첫 줄부터 `---`로 감싼 프런트매터를 두고, 본문은 42 서브젝트 양식을 따른다. `subjects/gauss_jordan/SUBJECT.md`를 그대로 베끼면 된다.
2. 프런트매터 필드는 `SPEC.md` §1에 있다: `id`, `title`, `version`, `author`, `language`, `license`, `programming_language`, `estimated_hours`, `prerequisites`, `tags` 등. `id`는 폴더 이름과 같게 둔다. 목록은 `[a, b]`로 쓴다.
3. 테스트 입력과 기대 출력이 있으면 `subjects/<id>/tests/`에 둔다. 규격은 과제마다 `tests/cases/`와 `tests/expected/`를 권한다.
4. `make site`를 돌리면 홈의 과제 목록과 `subjects/<id>/index.html`이 생긴다. 커밋한다.

규격 §5에 따르면 둘 이상의 카뎃이 완료하고 평가한 과제가 목록에 오른다. 지금은 초안 단계라 생성기는 등록된 과제를 모두 보인다.

## 기록을 서명하고 검증하기

기록은 JSON 한 파일이고, 서명은 OpenSSH의 sshsig(`ssh-keygen -Y sign`), 이름공간은 `community-subject`다. 새 키 인프라는 없다. 카뎃이 이미 쓰는 SSH 키를 그대로 쓴다.

기계 기록은 키트의 러너가 만든다. 저장소 루트에서, 제출물 대신 참조 구현으로 돌리면:

```
SUBJECT=subjects/gauss_jordan/SUBJECT.md \
RESULT=records/<피평가자>/gauss_jordan-0.1-machine.json \
kit/tests/run.sh "python3 kit/tests/reference.py"
```

사람 기록은 평가자가 직접 쓴다. 필드는 `SPEC.md` §4대로 과제 id와 버전, 피평가자, 평가자, 날짜, 플래그, 섹션별 결과다. `records/demo-cadet/gauss_jordan-0.1-human.json`이 예다. `evaluator`는 서명 신원과 같아야 하고, 다르면 생성기가 FAIL로 표시한다.

서명과 검증:

```
kit/record/sign.sh   records/<피평가자>/<기록>.json <intra>@<campus> [비밀키]
kit/record/verify.sh records/<피평가자>/<기록>.json allowed_signers <intra>@<campus>
make verify          # records/ 아래 모든 기록을 한 번에
```

검증하는 쪽은 `allowed_signers`에 평가자의 공개키 한 줄만 있으면 된다. 형식은 `<신원> namespaces="community-subject" ssh-ed25519 <공개키>`이고, `kit/record/sign.sh`가 끝에 그 한 줄을 출력해 준다. 생성기와 `make verify`는 서명 키를 `allowed_signers`에서 찾아(`ssh-keygen -Y find-principals`) 신원을 알아낸 뒤 `kit/record/verify.sh`를 돌리므로 기록 파일 자체가 서명자를 적을 필요는 없다. 기록의 한 바이트라도 바뀌면 FAIL이다.

### 데모 키

`demo/keys/demo_evaluator_ed25519.pub`는 이 저장소의 샘플 기록을 서명한 **데모 키**의 공개키다. 신원은 `demo-evaluator@example`이고, `allowed_signers`의 데모 줄이 이 키다. 데모 신원이므로 누구에 대해서도 아무것도 증명하지 않는다. 개인키는 저장소에 없다. 샘플 기록을 고쳐서 다시 서명하려면 각자의 SSH 키로 서명하고, 그 공개키 한 줄을 `allowed_signers`에 넣는다. 실제 기록도 자기 SSH 키로 서명하고, `allowed_signers`에서 데모 줄은 지운다. 샘플 기록 둘(`records/demo-cadet/`)은 데모 키로 서명했고, 기계 기록은 키트의 참조 구현을 돌린 결과다.

## Telecadeta와의 관계

Telecadeta 앱은 평가 슬롯과 알림을 맡는다. 피평가자가 슬롯을 열면 어느 캠퍼스의 카뎃이든 예약할 수 있고, 시간이 되면 알림이 간다. 이 저장소는 과제와 기록을 맡는다. 과제 본문과 테스트는 Git에 있고, 평가 결과는 평가자가 서명해 피평가자에게 주는 파일이다. 앱은 이 저장소의 색인을 보여 주거나 기록의 사본을 들 수 있지만 진실은 서명된 파일이다. 서버는 어느 쪽에도 필요 없다. 앱이 사라져도 과제는 포크할 수 있고 기록은 검증되며 색인은 다시 만들 수 있다.

## 개발

```
make test    # go vet ./... 와 go test ./... (프런트매터 파싱, 서명 검증 OK·FAIL 분기, 사이트 생성)
make clean   # site/public 삭제
```
