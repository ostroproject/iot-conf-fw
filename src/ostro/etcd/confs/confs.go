package confs

import (
	"log"
	"os"
	"io/ioutil"
	"bytes"
	"strings"
	"path/filepath"
	"errors"
	"crypto/md5"
	"encoding/json"
	"golang.org/x/sys/unix"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

const (
	HashAttr string = "user.hash"
	OriginAttr string = "user.origin"

	EtcdOriginated = "Etcd"
)

type Change struct {
	action string
	key string
	value string
}

func (me *Change) String() string {
	return me.action + " " + me.key + "=" + me.value
}

type ConFS struct {
	etcd client.Client
	keys client.KeysAPI
	watcher client.Watcher
	ctx context.Context
	prefix string
	cache string
	prefixError error
	lengthError error
	originError error
}

func NewConFS(servers []string, prefix, cache string) *ConFS {
	cfg := client.Config{Endpoints: servers}
	
	if len(prefix) > 0 && prefix[0] != '/' {
		log.Fatalf("Invalid prefix '%s': does not start with '/'\n", prefix)
	}

	if prefix == "/" {
		prefix = ""
	}

	if !strings.HasPrefix(cache, "/") {
		log.Fatalf("relative cache path: '%s'\n", cache)
	}

	etcd, err := client.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create etcd client: %v\n", err)
	}
	keys := client.NewKeysAPI(etcd)
	
	watcher := keys.Watcher(prefix, &client.WatcherOptions{0, true})
	
	ctx := context.TODO()
	prefixError := errors.New("prefix error")
	lengthError := errors.New("only part could be written")
	originError := errors.New("not Etcd originated")
	
	confs := &ConFS{etcd, keys, watcher, ctx, prefix, cache, prefixError, lengthError, originError}
	
	if err := confs.etcd.Sync(confs.ctx);  err != nil {
		log.Fatalf("Failed to sync: %v\n", err)
	}
	
	return confs
}

func (me *ConFS) _traverse_etcd_tree(node *client.Node, changes []Change) []Change {
	if node == nil {
		return changes
	}

	if !node.Dir {
		return append(changes, Change{"set", node.Key, node.Value})
	}

	changes_out := changes
	for _, subnode := range node.Nodes {
		changes_out = me._traverse_etcd_tree(subnode, changes_out)
	}

	return changes_out
}

func (me *ConFS) TraverseEtcdTree() ([]Change, error) {
	opts := client.GetOptions{Recursive:true, Sort:true, Quorum:false}

	resp, err := me.keys.Get(me.ctx, me.prefix, &opts)
	if err != nil {
		log.Printf("failed to travers on etcd tree '%s': %v\n", me.prefix, err)
		return nil, err
	}
	
	changes := me._traverse_etcd_tree(resp.Node, []Change{})
	
	return changes, nil
}

func (me *ConFS) PollForChanges() (<-chan []Change, <-chan error) {
	change_chan := make(chan []Change, 5)
	err_chan := make(chan error)
	
	go func() {
		for {
			resp , err := me.watcher.Next(me.ctx)
			if err != nil {
				err_chan <- err
			} else {
				change_chan <- []Change{{resp.Action, resp.Node.Key, resp.Node.Value}}
			}
		}

		close(err_chan)
		close(change_chan)
	} ()
	
	return change_chan, err_chan
}

func (me *ConFS) _get_cache_path(key string) (string, error) {
	if !strings.HasPrefix(key, me.prefix) {
		return "", me.prefixError
	}

	return me.cache + strings.TrimPrefix(key, me.prefix), nil
}

func (me *ConFS) _set_file(key string, value string, silent bool) error {
	cache_path, err := me._get_cache_path(key)
	if err != nil {
		return err
	}
	
	var jsonData interface{}
	var content []byte
	if err := json.Unmarshal(bytes.Trim([]byte(value), " \t\n\r"), &jsonData); err != nil {
		log.Printf("'%s' contains invalid JSON '%s': %v\n", key, value, err)
		return err
	}
	content, err = json.Marshal(jsonData)
	if err != nil {
		log.Printf("Error during canonizing '%s': %v\n", key, err)
		return err
	}
	length := len(content)
	
	md5sum := md5.Sum([]byte(content))
	content_hash := make([]byte, md5.Size)
	for i, c := range md5sum {
		content_hash[i] = c
	}

	etcdOriginated := []byte(EtcdOriginated)

	cached_file_exist := false
	if info, err := os.Stat(cache_path); err == nil {
		cached_file_exist = true

		origin := make([]byte,len(etcdOriginated))
		_, err := unix.Getxattr(cache_path, OriginAttr, origin)
		if err != nil || !bytes.Equal(origin, etcdOriginated) {
			log.Printf("Can't copy '%s' <= '%s': %v\n", cache_path, content, me.originError)
			return me.originError
		}

		if int(info.Size()) == length {
			file_hash := make([]byte, md5.Size)
			_, err := unix.Getxattr(cache_path, HashAttr, file_hash)
			
			if err == nil && bytes.Equal(content_hash, file_hash) {
				if !silent {
					log.Printf("Do not copy '%s' <= '%s': No change\n", cache_path, content)
				}
				return nil
			}
		}
	}
    
	dir := filepath.Dir(cache_path)
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir,0755); err != nil {
			log.Printf("Can't copy '%s' <= '%s': %v\n", cache_path, content, err)
			return err
		}
		
		log.Printf("creating directory '%s'\n", dir)
	}

	log.Printf("copying '%s' <= '%s' (%x)\n", cache_path, content, content_hash)

	file, err := os.OpenFile(cache_path, os.O_RDWR | os.O_CREATE | os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Failed to open file '%s': %v\n", cache_path, err)
		return err
	}
	
	if written, err := file.Write(content);  err != nil || written != length {
		if err == nil {
			err = me.lengthError
		}
		log.Printf("Failed to write file '%s': %v\n", cache_path, err)
		file.Close()
		return err
	}
	
	if err := file.Close(); err != nil {
		log.Printf("Failed to close file '%s': %v\n", cache_path, err)
		return err
	}
	
	var flags int
	if cached_file_exist {
		flags = 2
	} else {
		flags = 1
	}
	if err := unix.Setxattr(cache_path, HashAttr, content_hash, flags);  err != nil {
		log.Printf("Failed to set '%s' xattr on file '%s': %v\n", HashAttr, cache_path, err)
		return err
	}
	if !cached_file_exist {
		if err := unix.Setxattr(cache_path, OriginAttr, etcdOriginated, flags);  err != nil {
			log.Printf("Failed to set '%s' xattr on file '%s': %v\n", OriginAttr, etcdOriginated, err)
			return err
		}
	}
	
	return nil
}

func (me *ConFS) _rm_file(key string, silent bool) error {
	cache_path, err := me._get_cache_path(key)
	if err != nil {
		return err
	}
	
	if _, err := os.Stat(cache_path); err == nil {
		log.Printf("removing '%s'\n", cache_path)

		if err := os.Remove(cache_path); err != nil {
			log.Printf("Failed to remove '%s': %v\n", cache_path, err)
			return err
		}
	} else {
		if !silent {
			log.Printf("Do not remove '%s': file does not exist\n", cache_path)
		}
	}

	dir_path := cache_path
	dir_prefix,_ := me._get_cache_path("")
	for {
		dir_path = filepath.Dir(dir_path)

		if dir_path == "/" || !strings.HasPrefix(dir_path, dir_prefix) {
			break
		}

		if info, err := ioutil.ReadDir(dir_path); err != nil || len(info) > 0 {
			break
		}

		if err := os.Remove(dir_path);  err != nil {
			log.Printf("Failed to remove empty directory '%s': %v\n", dir_path, err)
			return err
		} else {
			log.Printf("removing empty directory '%s'\n", dir_path)
		}
	}

	return nil
}

func (me *ConFS) SyncFiles(changes []Change, silent bool) error {
	for _, c := range changes {
		switch c.action {
		case "set":
			if err := me._set_file(c.key, c.value, silent); err != nil {
				return err
			}
		case "delete":
			if err := me._rm_file(c.key, silent); err != nil {
				return err
			}
		}
	}
	
	return nil
}
