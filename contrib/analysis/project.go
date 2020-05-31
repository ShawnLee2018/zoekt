package analysis

import (
	"errors"
	"os"
	"log"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	P4_BIN string
	GIT_BIN string
	CTAGS_BIN string
)

func init() {
	P4_BIN = os.Getenv("ZOEKT_P4_BIN")
	GIT_BIN = os.Getenv("ZOEKT_GIT_BIN")
	CTAGS_BIN = os.Getenv("ZOEKT_CTAGS_BIN")
}

// IProject project operator interface
type IProject interface {
	Sync() (map[string]string, error) // return filepath to store latest modified file list
	Compile() error // virtually compile project; store metadata into disk: dump commit message, build ast tree ...
	GetProjectType() string // return p4, git, ...
	GetFileTextContents(filepath, revision string) (string, error)
	GetFileBinaryContents(filepath, revision string, startOffset, endOffset int) ([]byte, error)
	GetFileByteLength(filepath, revision string) (int, error)
	GetFileHash(filepath, revision string) (string, error)
	GetBlameInfo(filepath, revision string, startLine, endLine int) ([]string, error)
	GetCommitInfo(filepath, revision string) ([]string, error)
}

type P4Project struct {
	Name string
	BaseDir string
	P4Port, P4User, P4Client string
}

func NewP4Project (projectName string, baseDir string, options map[string]string) *P4Project {
	if P4_BIN == "" {
		log.Panic("[E] ! cannot find p4 command")
	}
	// baseDir: absolute path
	port, ok := options["P4PORT"]
	if !ok {
		log.Printf("P/%s: [E] missing P4PORT\n", projectName)
		return nil
	}
	user, ok := options["P4USER"]
	if !ok {
		log.Printf("P/%s: [E] missing P4USER\n", projectName)
		return nil
	}
	client, ok := options["P4CLIENT"]
	if !ok {
		log.Printf("P/%s: [E] missing P4CLIENT\n", projectName)
		return nil
	}
	p := &P4Project{projectName, baseDir, port, user, client};
	return p
}

func (p *P4Project) prepareP4folder () error {
	p4folder := filepath.Join(p.BaseDir, ".p4")
	fileinfo, err := os.Stat(p4folder)
	if os.IsNotExist(err) {
		os.Mkdir(p4folder, 0755)
	} else if err != nil {
		return err
	} else if !fileinfo.IsDir() {
		return errors.New(".p4 has been used as a normal file not a directory")
	}

	p4config := filepath.Join(p4folder, "config")
	f, err := os.Create(p4config)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("P4PORT=%s\nP4USER=%s\nP4CLIENT=%s\n", p.P4Port, p.P4User, p.P4Client))
	if err != nil {
		return err
	}
	return nil
}

func (p *P4Project) clone () (map[string]string, error) {
	cmd := fmt.Sprintf(
		"P4PORT=%s P4USER=%s P4CLIENT=%s %s sync -f",
		p.P4Port, p.P4User, p.P4Client, P4_BIN,
	)
	log.Println(cmd)
	err := Exec2Lines(cmd, func (line string) {
		fmt.Println("p4:", line)
	})
	err = p.prepareP4folder()
	return nil, err
}
func (p *P4Project) sync () (map[string]string, error) {
	cmd := fmt.Sprintf(
		"P4PORT=%s P4USER=%s P4CLIENT=%s %s sync",
		p.P4Port, p.P4User, p.P4Client, P4_BIN,
	)
	log.Println(cmd)
	err := Exec2Lines(cmd, func (line string) {
		fmt.Println("p4:", line)
	})
	return nil, err
}
func (p *P4Project) Sync () (map[string]string, error) {
	fileinfo, err := os.Stat(p.BaseDir)
	if os.IsNotExist(err) {
		return p.clone()
	}
	if err != nil {
		return nil, err
	}
	if !fileinfo.IsDir() {
		return nil, errors.New(fmt.Sprintf("P/%s: [E] cannot clone repo since \"%s\" is not a directory", p.Name))
	}
	return p.sync()
}

func (p *P4Project) Compile () error {
	return nil
}

func (p *P4Project) GetProjectType () string {
	return "p4"
}

func (p *P4Project) GetFileTextContents (filepath, revision string) (string, error) {
	return "", nil
}

func (p *P4Project) GetFileBinaryContents (filepath, revision string) ([]byte, error) {
	return nil, nil
}

func (p *P4Project) GetFileHash (filepath, revision string) (string, error) {
	return "", nil
}

func (p *P4Project) GetFileByteLength (filepath, revision string) {
}

func (p *P4Project) GetBlameInfo (filepath, revision string, startLine, endLine int) ([]string, error) {
	return nil, nil
}

func (p *P4Project) GetCommitInfo (filepath, revision string) ([]string, error) {
	return nil, nil
}

type GitProject struct {
	Name string
	BaseDir string
	Url, Branch string
}

func NewGitProject (projectName string, baseDir string, options map[string]string) *GitProject {
	if GIT_BIN == "" {
		log.Panic("[E] ! cannot find git command")
	}
	// baseDir: absolute path
	url, ok := options["Url"]
	if !ok {
		log.Printf("P/%s: [E] missing Url\n", projectName)
		return nil
	}
	branch, ok := options["Branch"]
	if !ok {
		log.Printf("P/%s: [W] missing Branch; using default\n", projectName)
		branch = ""
	}
	p := &GitProject{projectName, baseDir, url, branch};
	return p
}

func (p *GitProject) getCurrentBranch () (string, error) {
	cmd := fmt.Sprintf("%s -C %s branch", GIT_BIN, p.BaseDir)
	log.Println(cmd)
	err := Exec2Lines(cmd, func (line string) {
		if strings.HasPrefix(line, "* ") {
			p.Branch = strings.Fields(line)[1]
		}
	})
	return p.Branch, err
}

func (p *GitProject) clone () (map[string]string, error) {
	cmd := ""
	if p.Branch == "" {
		cmd = fmt.Sprintf(
			"%s clone %s %s",
			GIT_BIN, p.Url, p.BaseDir,
		)
		log.Println(cmd)
		err := Exec2Lines(cmd, nil)
		if err != nil {
			return nil, err
		}
		p.getCurrentBranch()
	} else {
		cmd = fmt.Sprintf(
			"%s clone %s -b %s %s",
			GIT_BIN, p.Url, p.Branch, p.BaseDir,
		)
		log.Println(cmd)
		err := Exec2Lines(cmd, nil)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
func (p *GitProject) sync () (map[string]string, error) {
	cmd := fmt.Sprintf(
		"%s -C %s fetch --all",
		GIT_BIN, p.BaseDir,
	)
	log.Println(cmd)
	Exec2Lines(cmd, nil)
	if p.Branch == "" {
		p.getCurrentBranch()
	}
	cmd = fmt.Sprintf(
		"%s -C %s reset --hard origin/%s",
		GIT_BIN, p.BaseDir, p.Branch,
	)
	log.Println(cmd)
	err := Exec2Lines(cmd, nil)
	return nil, err
}
func (p *GitProject) Sync () (map[string]string, error) {
	fileinfo, err := os.Stat(p.BaseDir)
	if os.IsNotExist(err) {
		return p.clone()
	}
	if err != nil {
		return nil, err
	}
	if !fileinfo.IsDir() {
		return nil, errors.New(fmt.Sprintf("P/%s: [E] cannot clone repo since \"%s\" is not a directory", p.Name))
	}
	return p.sync()
}

func (p *GitProject) Compile () error {
	return nil
}

func (p *GitProject) GetProjectType () string {
	return "git"
}

func (p *GitProject) GetFileTextContents (filepath, revision string) (string, error) {
	return "", nil
}

func (p *GitProject) GetFileBinaryContents (filepath, revision string) ([]byte, error) {
	return nil, nil
}

func (p *GitProject) GetFileHash (filepath, revision string) (string, error) {
	return "", nil
}

func (p *GitProject) GetFileByteLength (filepath, revision string) {
}

func (p *GitProject) GetBlameInfo (filepath, revision string, startLine, endLine int) ([]string, error) {
	return nil, nil
}

func (p *GitProject) GetCommitInfo (filepath, revision string) ([]string, error) {
	return nil, nil
}