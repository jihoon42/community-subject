package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Section is one row of a human evaluation sheet.
type Section struct {
	Name   string `json:"name"`
	Result string `json:"result"`
	Note   string `json:"note,omitempty"`
}

// Record is one signed evaluation record, machine or human, as the index
// shows it. Owner is the evaluatee: the directory under records/.
type Record struct {
	Path  string // absolute path of the JSON file
	File  string // base name
	Owner string
	Kind  string // "machine" or "human"

	SubjectID      string
	SubjectVersion string
	Date           string

	Signer string
	OK     bool
	Note   string

	// machine
	Passed, Failed, Total int
	Commit                string
	Uname                 string

	// human
	Evaluator string
	Flag      string
	Sections  []Section
}

// Owner groups the records of one evaluatee.
type Owner struct {
	Name    string
	Records []*Record
}

// Verification is the outcome of checking one record's signature.
type Verification struct {
	Signer string
	OK     bool
	Note   string
}

// Verifier runs kit/record/verify.sh against allowed_signers. The signer
// identity is discovered from the signature with ssh-keygen -Y
// find-principals, so a record never has to name its own signer.
type Verifier struct {
	Script         string
	AllowedSigners string
}

const namespace = "community-subject"

func (v Verifier) Verify(record string) Verification {
	sig := record + ".sig"
	if _, err := os.Stat(sig); err != nil {
		return Verification{Note: "서명 파일(.sig)이 없다"}
	}
	if _, err := os.Stat(v.AllowedSigners); err != nil {
		return Verification{Note: "allowed_signers 파일이 없다"}
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		return Verification{Note: "ssh-keygen을 찾을 수 없다"}
	}
	out, err := exec.Command("ssh-keygen", "-Y", "find-principals",
		"-s", sig, "-f", v.AllowedSigners).Output()
	if err != nil {
		return Verification{Note: "서명한 키가 allowed_signers에 없다"}
	}
	signer := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if signer == "" {
		return Verification{Note: "서명한 키가 allowed_signers에 없다"}
	}
	cmd := exec.Command(v.Script, record, v.AllowedSigners, signer)
	if err := cmd.Run(); err != nil {
		return Verification{Signer: signer, Note: "verify.sh 실패: 기록이 서명과 다르다"}
	}
	return Verification{Signer: signer, OK: true, Note: "verify.sh 종료 상태 0"}
}

type rawRecord struct {
	Kind    string `json:"kind"`
	Subject struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"subject"`
	Submission struct {
		Commit string `json:"commit"`
	} `json:"submission"`
	Environment struct {
		Uname string `json:"uname"`
	} `json:"environment"`
	Cases   []json.RawMessage `json:"cases"`
	Summary struct {
		Passed int `json:"passed"`
		Failed int `json:"failed"`
		Total  int `json:"total"`
	} `json:"summary"`
	Timestamp string    `json:"timestamp"`
	Evaluatee string    `json:"evaluatee"`
	Evaluator string    `json:"evaluator"`
	Date      string    `json:"date"`
	Flag      string    `json:"flag"`
	Sections  []Section `json:"sections"`
}

// loadRecord reads one record file and verifies it.
func loadRecord(path, owner string, v Verifier) (*Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw rawRecord
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	r := &Record{
		Path:           path,
		File:           filepath.Base(path),
		Owner:          owner,
		SubjectID:      raw.Subject.ID,
		SubjectVersion: raw.Subject.Version,
	}
	switch {
	case raw.Kind == "human" || (raw.Kind == "" && raw.Flag != ""):
		r.Kind = "human"
		r.Date = raw.Date
		r.Evaluator = raw.Evaluator
		r.Flag = raw.Flag
		r.Sections = raw.Sections
	case raw.Kind == "machine" || (raw.Kind == "" && raw.Cases != nil):
		r.Kind = "machine"
		r.Date = raw.Timestamp
		r.Passed, r.Failed, r.Total = raw.Summary.Passed, raw.Summary.Failed, raw.Summary.Total
		r.Commit = raw.Submission.Commit
		r.Uname = raw.Environment.Uname
	default:
		return nil, fmt.Errorf("%s: cannot tell machine from human record", path)
	}
	ver := v.Verify(path)
	r.Signer, r.OK, r.Note = ver.Signer, ver.OK, ver.Note
	if r.OK && r.Kind == "human" && r.Evaluator != "" && r.Evaluator != r.Signer {
		r.OK = false
		r.Note = "서명자와 기록의 evaluator가 다르다"
	}
	return r, nil
}

// loadOwners walks records/<owner>/*.json.
func loadOwners(recordsDir string, v Verifier) ([]*Owner, error) {
	entries, err := os.ReadDir(recordsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var owners []*Owner
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		o := &Owner{Name: e.Name()}
		files, err := filepath.Glob(filepath.Join(recordsDir, e.Name(), "*.json"))
		if err != nil {
			return nil, err
		}
		sort.Strings(files)
		for _, f := range files {
			r, err := loadRecord(f, o.Name, v)
			if err != nil {
				return nil, err
			}
			o.Records = append(o.Records, r)
		}
		if len(o.Records) > 0 {
			owners = append(owners, o)
		}
	}
	return owners, nil
}
