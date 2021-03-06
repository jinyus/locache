package locache

import (
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/GitbookIO/syncgroup"
)

const (
	keyDoesntExistsError = "locache : key doesn't exists"
	ttlParsingError      = "locache : error parsing first line of cache file"
	keyExpiredError      = "locache : key has expired"
)

type Locache struct {
	*locache
}

type locache struct {
	directory         string
	lock              *syncgroup.MutexGroup
	compress          bool
	janitor           *janitor
	cleaningSemaphore chan struct{}
	fileExtension     string
}

type Config struct {
	Directory       string
	UseCompression  bool
	CleanUpInterval time.Duration
}

//Creates a new Locache, gzip compression will be used if UseCompression is true
//Performance:
//	14000 reads/s without compression   (15kb read)
//  6000 reads/s with compression 		(6kb read)
//
//	250 writes/s without compression    (15kb written)
//  1000 writes/s with compression	    (6kb written)
func New(cfg *Config) (*Locache, error) {
	// Create Locache directory
	if err := os.MkdirAll(cfg.Directory, os.ModePerm); err != nil {
		return nil, fmt.Errorf("locache: New : could not make cache directory: %v", err)
	}

	// Create Locache instance
	var extension string
	if cfg.UseCompression {
		extension = ".gzip"
	} else {
		extension = ".cache"
	}
	c := &locache{
		directory:         cfg.Directory,
		lock:              syncgroup.NewMutexGroup(),
		compress:          cfg.UseCompression,
		cleaningSemaphore: make(chan struct{}, 1),
		fileExtension:     extension,
	}

	C := &Locache{c}

	if cfg.CleanUpInterval > 0 {
		//println("clean up interval", cfg.CleanUpInterval.String())
		runJanitor(c, cfg.CleanUpInterval)
		runtime.SetFinalizer(C, stopJanitor)
	}
	return C, nil
}

//Caches the data provided and sets the expiration date
//based on the Time To Live(TTL) provided
func (c *locache) Set(key string, item interface{}, expiryTimestamp int64) error {

	data, err := EncodeGob(item)
	if err != nil {
		return errors.New("locache : Set : could not encode item : " + err.Error())
	}
	// Get encoded key
	filename := c.getFilename(key)

	// Lock for writing
	c.lock.Lock(filename)
	defer c.lock.Unlock(filename)

	// Open file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	//fmt.Printf("caching %s filename: %s\n", key, filename)
	if c.compress {
		zw := gzip.NewWriter(file)
		defer zw.Close()
		zw.ModTime = time.Unix(expiryTimestamp, 0)
		//buf := bytes.NewBuffer(data)
		//_, err = io.Copy(zw, buf)
		_, err = zw.Write(data)
	} else {
		// Write expDate on the first line of the file
		expiryDateFmt := []byte(fmt.Sprintf("%d\n", expiryTimestamp))
		if _, err = file.Write(expiryDateFmt); err == nil {
			_, err = file.Write(data)
		}
	}
	//fmt.Printf("done caching %s err: %v\n", key, err)
	return err
}

//Looks up the key in the cache,
//keyExpiredError will be returned if the data has expired
func (c *locache) Get(key string, result interface{}) error {
	// Get encoded key
	filename := c.getFilename(key)

	// Lock for reading
	c.lock.RLock(filename)
	defer c.lock.RUnlock(filename)

	if ok, err := FileExists(filename); err != nil {
		return fmt.Errorf("locache : Get : error reading file : %v", err)
	} else if !ok {
		return errors.New(keyDoesntExistsError)
	}

	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	var data []byte

	if c.compress {
		zr, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer zr.Close()

		if zr.ModTime.Before(time.Now()) {
			return errors.New(keyExpiredError)
		}
		//buf := bytes.NewBuffer(data)
		if data, err = ioutil.ReadAll(zr); err != nil {
			return fmt.Errorf("locach  : Get : error reading compressed file : %v ", err)
		}
		return DecodeGob(data, result)
	}

	var count int

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if count == 0 {
			//reads first line to get expiry date
			count++
			txt := scanner.Text()
			ttl, err := strconv.Atoi(txt)
			if err != nil {
				return fmt.Errorf("%s: %v", ttlParsingError, err)
			}
			if int64(ttl) < time.Now().Unix() {
				return errors.New(keyExpiredError)
			}
			continue
		}
		data = append(data, []byte(scanner.Text()+"\n")...)
	}

	return DecodeGob(data, result)
}

func (c *locache) Delete(key string) error {
	// Get encoded key
	filename := c.getFilename(key)

	// Lock for writing
	c.lock.Lock(filename)
	defer c.lock.Unlock(filename)

	if ok, err := FileExists(filename); ok {
		return os.Remove(filename)
	} else {
		return err
	}
}

func (c *locache) DeleteAll() error {
	// Delete directory
	if err := os.RemoveAll(c.directory); err != nil {
		return err
	}
	// Recreate directory
	return os.MkdirAll(c.directory, os.ModePerm)
}

func (c *locache) DeleteExpired() {
	//acquire token
	c.cleaningSemaphore <- struct{}{}
	defer func() {
		//release token
		<-c.cleaningSemaphore
	}()

	cacheFiles := FindFilesByExt(c.directory, c.fileExtension)

	for _, file := range cacheFiles {
		lockKey := path.Join(c.directory, file.Name())
		if c.isExpired(lockKey) {
			err := c.deleteFile(lockKey)
			if err != nil {
				log.Println("deletion error: ", err)
			}

		}
	}

}

func (c *locache) deleteFile(filename string) error {
	// Lock for writing
	c.lock.Lock(filename)
	defer c.lock.Unlock(filename)

	if ok, err := FileExists(filename); ok {
		return os.Remove(filename)
	} else {
		return err
	}
}

func (c *locache) getFilename(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return path.Join(c.directory, hex.EncodeToString(hasher.Sum(nil))+c.fileExtension)
}

func (c *locache) isExpired(filename string) bool {
	c.lock.RLock(filename)
	defer c.lock.RUnlock(filename)

	if ok, err := FileExists(filename); !ok || err != nil {
		return false
	}
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	if c.compress {
		zr, err := gzip.NewReader(file)
		if err != nil {
			return false
		}
		defer zr.Close()

		if zr.ModTime.Before(time.Now()) {
			return true
		}
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		//reads first line to get expiry date
		txt := scanner.Text()
		ttl, err := strconv.Atoi(txt)
		if err != nil {
			return false
		}
		if int64(ttl) < time.Now().Unix() {
			return true
		}
		return false
	}
	return false
}
