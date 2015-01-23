package internal

import (
	"bufio"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

func (s *Server) runScript(path string) (err error) {
	var count int64
	defer func() {
		s.l.Debugf("script `%s` exited; emitted %d stat(s)", path, count)
	}()
	cmd := exec.Command(path)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		if e2 := cmd.Wait(); e2 != nil && err == nil {
			err = e2
		}
	}()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		s.handleMessage(line)
		count++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (s *Server) runScripts() {
	var scriptMutex sync.Mutex // protects currentlyRunning
	currentlyRunning := make(map[string]struct{})
	ticker := time.NewTicker(time.Duration(s.conf.Scripts.RunIntervalMS) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			files, err := ioutil.ReadDir(s.conf.Scripts.Path)
			if err != nil {
				s.l.Debugf("failed to read scripts in %s: %s", s.conf.Scripts.Path, err)
				s.metaInc("errors.run_scripts_list_dir")
				continue
			}
			scriptMutex.Lock()
			for _, file := range files {
				if !file.Mode().IsRegular() {
					continue
				}
				path := filepath.Join(s.conf.Scripts.Path, file.Name())
				if _, ok := currentlyRunning[path]; ok {
					s.l.Debugf("not running script because a previous instance is still running: %s", path)
					continue
				}
				s.l.Debugf("running script: %s", path)
				currentlyRunning[path] = struct{}{}
				go func(p string) {
					if err := s.runScript(p); err != nil {
						s.l.Debugf("error running script at %s: %s", p, err)
						s.metaInc("errors.run_script")
					}
					scriptMutex.Lock()
					delete(currentlyRunning, path)
					scriptMutex.Unlock()
				}(path)
			}
			scriptMutex.Unlock()
		case <-s.quit:
			return
		}
	}
}
