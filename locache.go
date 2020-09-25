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
	expiryQueue          = "expiryQueue"
)

type Locache struct {
	*locache
}

type locache struct {
	directory    string
	lock         *syncgroup.MutexGroup
	compress     bool
	janitor      *janitor
	expiredItems map[string]bool
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
		return nil, err
	}

	// Create Locache instance
	c := &locache{
		directory:    cfg.Directory,
		lock:         syncgroup.NewMutexGroup(),
		compress:     cfg.UseCompression,
		expiredItems: make(map[string]bool),
	}

	C := &Locache{c}

	if cfg.CleanUpInterval > 0 {
		runJanitor(c, cfg.CleanUpInterval)
		runtime.SetFinalizer(C, stopJanitor)
	}
	return C, nil
}

func (c *locache) DeleteExpired() {
	println("deleting expired items")
	c.lock.RLock(expiryQueue)
	defer c.lock.RUnlock(expiryQueue)

	for key, _ := range c.expiredItems {
		println("found ", key, " in expiredItems")
		c.lock.Lock(key)
		if err := os.Remove(key); err != nil {
			fmt.Println("locache: DeleteExpired: deleted cache file failed: ", err)
		}
		delete(c.expiredItems, key)
		c.lock.Unlock(key)
	}

}

func (c *locache) addToExpiryQueue(filename string) {
	//lock map for writing
	c.lock.Lock(expiryQueue)
	defer c.lock.Unlock(expiryQueue)

	c.expiredItems[filename] = true
}
func (c *locache) removeFromExpiryQueue(filename string) {
	//lock map for writing
	c.lock.Lock(expiryQueue)
	defer c.lock.Unlock(expiryQueue)

	delete(c.expiredItems, filename)
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

	if err != nil {
		go c.removeFromExpiryQueue(filename)
	}

	return err
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
			go c.addToExpiryQueue(filename)
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

func (c *locache) Clean() error {
	// Delete directory
	if err := os.RemoveAll(c.directory); err != nil {
		return err
	}
	// Recreate directory
	return os.MkdirAll(c.directory, os.ModePerm)
}

func (c *locache) getFilename(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	var extension string
	if c.compress {
		extension = ".gzip"
	} else {
		extension = ".cache"
	}
	return path.Join(c.directory, hex.EncodeToString(hasher.Sum(nil))+extension)
}

// check file exist.
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
