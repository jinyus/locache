package locache

import (
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
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
		runJanitor(c, cfg.CleanUpInterval)
		runtime.SetFinalizer(C, stopJanitor)
	}
	return C, nil
}

//Caches the data provided and sets the expiration date
//based on the Time To Live(TTL) provided
func (c *locache) Set(key string, data []byte, TTL time.Duration) error {
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

	expiryDate := time.Now().Add(TTL)
	expiryDateFmt := []byte(fmt.Sprintf("%d\n", expiryDate.Unix()))

	//fmt.Printf("caching %s filename: %s\n", key, filename)
	if c.compress {
		zw := gzip.NewWriter(file)
		defer zw.Close()
		zw.ModTime = expiryDate
		//buf := bytes.NewBuffer(data)
		//_, err = io.Copy(zw, buf)
		_, err = zw.Write(data)
	} else {
		// Write expDate on the first line of the file
		if _, err = file.Write(expiryDateFmt); err == nil {
			_, err = file.Write(data)
		}
	}
	//fmt.Printf("done caching %s err: %v\n", key, err)
	return err
}

//Looks up the key in the cache,
//keyExpiredError will be returned if the data has expired
func (c *locache) Get(key string) ([]byte, error) {
	// Get encoded key
	filename := c.getFilename(key)

	// Lock for reading
	c.lock.RLock(filename)
	defer c.lock.RUnlock(filename)

	if ok, err := exists(filename); !ok || err != nil {
		return nil, fmt.Errorf("%s: %v", keyDoesntExistsError, err)
	}

	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var data []byte

	if c.compress {
		zr, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		if zr.ModTime.Before(time.Now()) {
			return nil, errors.New(keyExpiredError)
		}
		//buf := bytes.NewBuffer(data)
		if data, err = ioutil.ReadAll(zr); err != nil {
			return nil, fmt.Errorf("locach Get : error reading compressed file : %v ", err)
		}
		return data, nil
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
				return nil, fmt.Errorf("%s: %v", ttlParsingError, err)
			}
			if int64(ttl) < time.Now().Unix() {
				return nil, errors.New(keyExpiredError)
			}
			continue
		}
		data = append(data, []byte(scanner.Text()+"\n")...)
	}

	return data, nil
}

func (c *locache) Delete(key string) error {
	// Get encoded key
	filename := c.getFilename(key)

	// Lock for writing
	c.lock.Lock(filename)
	defer c.lock.Unlock(filename)

	if ok, err := exists(filename); ok {
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
	//println("\nRunning janitor, waiting to aquire token")
	//acquire token
	c.cleaningSemaphore <- struct{}{}
	defer func() {
		//println("token released")
		<-c.cleaningSemaphore
	}()
	//println("token acquired")

	cacheFiles := findFilesByExt(c.directory, c.fileExtension)

	for _, file := range cacheFiles {
		lockKey := path.Join(c.directory, file.Name())
		//println("found ", lockKey)
		if c.isExpired(lockKey) {
			//println("deleting ", lockKey, " from cache")
			err := c.deleteFile(lockKey)
			if err != nil {
				println("deletion error: ", err)
			}

		}
	}

}

func (c *locache) deleteFile(filename string) error {
	// Lock for writing
	c.lock.Lock(filename)
	defer c.lock.Unlock(filename)

	if ok, err := exists(filename); ok {
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

	if ok, err := exists(filename); !ok || err != nil {
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
