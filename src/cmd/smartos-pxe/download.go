package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func downloadPlatform() error {
	resp, err := http.Get(downloadServer + downloadLatestPath)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	latestDir := strings.TrimSpace(string(bs))
	version := filepath.Base(latestDir)
	want := filepath.Join(datadir, "platform-"+version)
	if _, err := os.Stat(want); err == nil {
		return nil
	}

	url := downloadServer + latestDir + "/platform-" + version + ".tgz"
	if verbose {
		log.Println("Fetching", url)
	}
	resp, err = http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	if err := unpackTarGZ(datadir, resp.Body); err != nil {
		return err
	}

	if verbose {
		log.Println("Successfully downloaded new platform", version)
	}
	return nil
}

func unpackTarGZ(dstdir string, r io.Reader) error {
	tmpdir := fmt.Sprintf("%s.%d", dstdir, time.Now().UnixNano())

	gr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fi := hdr.FileInfo()
		path := filepath.Join(tmpdir, hdr.Name)

		if fi.IsDir() {
			err := os.MkdirAll(path, fi.Mode())
			if err != nil {
				return err
			}
			continue
		}

		if verbose {
			log.Println("Unpacking", hdr.Name)
		}

		fd, err := os.Create(path)
		if err != nil {
			return err
		}
		fd.Chmod(fi.Mode())
		if _, err := io.Copy(fd, tr); err != nil {
			return err
		}
		if err := fd.Close(); err != nil {
			return err
		}
	}

	tgts, _ := filepath.Glob(filepath.Join(tmpdir, "*"))
	for _, tgt := range tgts {
		base := filepath.Base(tgt)
		if err := os.Rename(tgt, filepath.Join(dstdir, base)); err != nil {
			return err
		}
	}
	os.RemoveAll(tmpdir)
	return nil
}
